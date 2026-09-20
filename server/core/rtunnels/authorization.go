// Package rtunnels owns reverse-port-forward authorization and relay state.
package rtunnels

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

const (
	authorizationIDBytes           = 32
	maxAuthorizationIDAttempts     = 16
	maxConnectionsPerAuthorization = 64
	maxConnectionsPerSession       = 256
	maxConnectionsGlobal           = 2048
)

// ErrInvalidSessionID and the related errors report rejected authorizations.
var (
	ErrInvalidSessionID             = errors.New("invalid reverse port forward session ID")
	ErrInvalidForwardAddress        = errors.New("invalid reverse port forward address")
	ErrUnknownAuthorization         = errors.New("unknown reverse port forward authorization")
	ErrAuthorizationSession         = errors.New("reverse port forward authorization belongs to another session")
	ErrAuthorizationRevoked         = errors.New("reverse port forward authorization is revoked")
	ErrAuthorizationActive          = errors.New("reverse port forward authorization is already active")
	ErrAuthorizationIDRequired      = errors.New("reverse port forward authorization ID is required")
	ErrAuthorizationConnectionLimit = errors.New("reverse port forward authorization connection limit reached")
	ErrSessionConnectionLimit       = errors.New("reverse port forward session connection limit reached")
	ErrGlobalConnectionLimit        = errors.New("reverse port forward global connection limit reached")
	ErrAuthorizationReservation     = errors.New("invalid reverse port forward connection reservation")
	ErrDuplicateListenerID          = errors.New("reverse port forward listener ID is already registered")
	ErrAuthorizationIDGeneration    = errors.New("failed to generate a unique reverse port forward authorization ID")
)

// AuthorizationID is an opaque, teamserver-generated capability identifying a
// reverse port forward authorization. Implants must not be able to choose it.
type AuthorizationID string

func (id AuthorizationID) String() string {
	return string(id)
}

// AuthorizationState describes the server-owned lifecycle of a reverse port
// forward listener authorization.
type AuthorizationState uint8

// AuthorizationStarting and the related states describe authorization lifecycle.
const (
	AuthorizationStarting AuthorizationState = iota + 1
	AuthorizationActive
	AuthorizationRevoked
)

func (state AuthorizationState) String() string {
	switch state {
	case AuthorizationStarting:
		return "starting"
	case AuthorizationActive:
		return "active"
	case AuthorizationRevoked:
		return "revoked"
	default:
		return "unknown"
	}
}

// Authorization is an immutable snapshot of a server-owned reverse port
// forward authorization. Mutating a snapshot cannot affect the registry.
type Authorization struct {
	AuthorizationID         AuthorizationID
	SessionID               string
	BindAddress             string
	Address                 string
	KeepAlive               int32
	State                   AuthorizationState
	ImplantListenerID       uint32
	HasListenerID           bool
	RequiresAuthorizationID bool
}

// dialPlan is destination data private to the registry and broker. Keeping it
// unexported prevents implant handlers from receiving an address they could
// accidentally redial outside the broker.
type dialPlan struct {
	AuthorizationID AuthorizationID
	Address         string
	KeepAlive       int32
}

type authorizationRecord struct {
	authorization      Authorization
	sequence           uint64
	revoked            context.Context
	cancel             context.CancelFunc
	connections        map[uint64]net.Conn
	pendingConnections uint64
	nextConnID         uint64
}

// connectionReservation is a once-consumable claim on all connection quota
// scopes. It keeps the owning record alive after revocation removes registry
// indexes, so late dial completion can release safely without looking up a
// newly-created authorization that happens to reuse the same opaque ID.
// Every field is protected by Registry.mu.
type connectionReservation struct {
	record *authorizationRecord
	active bool
}

// Registry owns all reverse port forward authorizations. Its indexes contain
// only server-generated IDs and destinations derived from operator requests.
type Registry struct {
	mu sync.RWMutex

	random io.Reader

	byID          map[AuthorizationID]*authorizationRecord
	bySession     map[string]map[AuthorizationID]*authorizationRecord
	byDestination map[string]map[AuthorizationID]*authorizationRecord
	byListener    map[string]map[uint32]*authorizationRecord

	sessionConnectionCounts map[string]uint64
	totalConnectionCount    uint64
	nextSequence            uint64
}

// NewRegistry creates an empty reverse port forward authorization registry.
func NewRegistry() *Registry {
	return newRegistry(rand.Reader)
}

func newRegistry(random io.Reader) *Registry {
	return &Registry{
		random:                  random,
		byID:                    map[AuthorizationID]*authorizationRecord{},
		bySession:               map[string]map[AuthorizationID]*authorizationRecord{},
		byDestination:           map[string]map[AuthorizationID]*authorizationRecord{},
		byListener:              map[string]map[uint32]*authorizationRecord{},
		sessionConnectionCounts: map[string]uint64{},
	}
}

// canonicalizeAddress validates and canonicalizes a TCP host:port address. It
// normalizes IP literals, hostname case, and IPv6 bracket formatting so legacy
// requests can only select an already-authorized operator destination.
func canonicalizeAddress(address string) (string, error) {
	address = strings.TrimSpace(address)
	host, portString, err := net.SplitHostPort(address)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidForwardAddress, err)
	}
	if host == "" {
		return "", fmt.Errorf("%w: host is empty", ErrInvalidForwardAddress)
	}
	port, err := strconv.ParseUint(portString, 10, 16)
	if err != nil || port == 0 {
		return "", fmt.Errorf("%w: port must be between 1 and 65535", ErrInvalidForwardAddress)
	}

	host, err = canonicalHost(host)
	if err != nil {
		return "", err
	}
	return net.JoinHostPort(host, strconv.FormatUint(port, 10)), nil
}

func canonicalHost(host string) (string, error) {
	if ip, err := netip.ParseAddr(host); err == nil {
		return ip.Unmap().String(), nil
	}

	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if host == "" || len(host) > 253 {
		return "", fmt.Errorf("%w: invalid hostname", ErrInvalidForwardAddress)
	}
	for _, label := range strings.Split(host, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", fmt.Errorf("%w: invalid hostname", ErrInvalidForwardAddress)
		}
		for _, char := range label {
			if unicode.IsLetter(char) || unicode.IsDigit(char) || char == '-' || char == '_' {
				continue
			}
			return "", fmt.Errorf("%w: invalid hostname", ErrInvalidForwardAddress)
		}
	}
	return host, nil
}

// Begin creates a Starting authorization for an operator-selected destination.
// Prefer BeginSpec when the operator's bind address is available.
func (registry *Registry) Begin(sessionID string, address string, keepAlive int32) (AuthorizationID, error) {
	return registry.BeginSpec(sessionID, "", address, keepAlive)
}

// BeginSpec creates a Starting authorization before the start request is sent
// to the implant. The destination and bind address are immutable thereafter.
func (registry *Registry) BeginSpec(sessionID string, bindAddress string, address string, keepAlive int32) (AuthorizationID, error) {
	if sessionID == "" {
		return "", ErrInvalidSessionID
	}
	canonicalAddress, err := canonicalizeAddress(address)
	if err != nil {
		return "", err
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.initializeLocked()

	id, err := registry.newAuthorizationIDLocked()
	if err != nil {
		return "", err
	}
	revoked, cancel := context.WithCancel(context.Background())
	registry.nextSequence++
	record := &authorizationRecord{
		authorization: Authorization{
			AuthorizationID: id,
			SessionID:       sessionID,
			BindAddress:     bindAddress,
			Address:         canonicalAddress,
			KeepAlive:       keepAlive,
			State:           AuthorizationStarting,
		},
		sequence:    registry.nextSequence,
		revoked:     revoked,
		cancel:      cancel,
		connections: map[uint64]net.Conn{},
	}

	registry.byID[id] = record
	if registry.bySession[sessionID] == nil {
		registry.bySession[sessionID] = map[AuthorizationID]*authorizationRecord{}
	}
	registry.bySession[sessionID][id] = record
	if registry.byDestination[sessionID] == nil {
		registry.byDestination[sessionID] = map[AuthorizationID]*authorizationRecord{}
	}
	registry.byDestination[sessionID][id] = record
	return id, nil
}

// Activate binds an implant listener ID to a Starting authorization. A
// duplicate implant-selected listener ID never replaces or orphans the existing
// record: the rejected candidate is revoked atomically.
func (registry *Registry) Activate(sessionID string, id AuthorizationID, listenerID uint32) error {
	return registry.ActivateProtocol(sessionID, id, listenerID, false)
}

// ActivateProtocol binds an implant listener ID and records whether the
// implant proved support for authorization IDs by echoing the issued ID in its
// start response. Legacy address lookup is disabled for capable listeners, so
// a new-implant protocol regression fails closed while older implants retain
// exact-address compatibility.
func (registry *Registry) ActivateProtocol(sessionID string, id AuthorizationID, listenerID uint32, requiresAuthorizationID bool) error {
	registry.mu.Lock()
	registry.initializeLocked()
	record, err := registry.lookupLocked(sessionID, id)
	if err != nil {
		registry.mu.Unlock()
		return err
	}
	if record.authorization.State == AuthorizationRevoked {
		registry.mu.Unlock()
		return ErrAuthorizationRevoked
	}
	if record.authorization.State == AuthorizationActive {
		if record.authorization.HasListenerID && record.authorization.ImplantListenerID == listenerID {
			registry.mu.Unlock()
			return nil
		}
		registry.mu.Unlock()
		return ErrAuthorizationActive
	}

	if registry.byListener[sessionID] == nil {
		registry.byListener[sessionID] = map[uint32]*authorizationRecord{}
	}
	if existing := registry.byListener[sessionID][listenerID]; existing != nil && existing != record && existing.authorization.State != AuthorizationRevoked {
		connections := registry.revokeLocked(record)
		registry.mu.Unlock()
		closeConnections(connections)
		return ErrDuplicateListenerID
	}

	record.authorization.State = AuthorizationActive
	record.authorization.ImplantListenerID = listenerID
	record.authorization.HasListenerID = true
	record.authorization.RequiresAuthorizationID = requiresAuthorizationID
	registry.byListener[sessionID][listenerID] = record
	registry.mu.Unlock()
	return nil
}

func dialPlanForRecord(record *authorizationRecord) (dialPlan, error) {
	if record.authorization.State == AuthorizationRevoked {
		return dialPlan{}, ErrAuthorizationRevoked
	}
	return dialPlan{
		AuthorizationID: record.authorization.AuthorizationID,
		Address:         record.authorization.Address,
		KeepAlive:       record.authorization.KeepAlive,
	}, nil
}

// Lookup returns an immutable snapshot for an authorization.
func (registry *Registry) Lookup(sessionID string, id AuthorizationID) (Authorization, bool) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	record, err := registry.lookupLocked(sessionID, id)
	if err != nil {
		return Authorization{}, false
	}
	return record.authorization, true
}

// LookupListener returns an immutable snapshot for an implant listener ID.
func (registry *Registry) LookupListener(sessionID string, listenerID uint32) (Authorization, bool) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	record := registry.byListener[sessionID][listenerID]
	if record == nil || record.authorization.State == AuthorizationRevoked {
		return Authorization{}, false
	}
	return record.authorization, true
}

// List returns server-authoritative Starting and Active listeners for a session.
func (registry *Registry) List(sessionID string) []Authorization {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	records := registry.bySession[sessionID]
	authorizations := make([]Authorization, 0, len(records))
	for _, record := range records {
		if record.authorization.State != AuthorizationRevoked {
			authorizations = append(authorizations, record.authorization)
		}
	}
	sort.Slice(authorizations, func(i, j int) bool {
		return authorizations[i].AuthorizationID < authorizations[j].AuthorizationID
	})
	return authorizations
}

// Revoke prevents new acquisitions and closes pending or active broker-owned
// connections for an authorization. It is idempotent.
func (registry *Registry) Revoke(sessionID string, id AuthorizationID) bool {
	registry.mu.Lock()
	record, err := registry.lookupLocked(sessionID, id)
	if err != nil || record.authorization.State == AuthorizationRevoked {
		registry.mu.Unlock()
		return false
	}
	connections := registry.revokeLocked(record)
	registry.mu.Unlock()
	closeConnections(connections)
	return true
}

// RevokeListener revokes by the implant's listener ID. It does not depend on a
// successful implant stop response and is idempotent.
func (registry *Registry) RevokeListener(sessionID string, listenerID uint32) bool {
	registry.mu.Lock()
	record := registry.byListener[sessionID][listenerID]
	if record == nil || record.authorization.State == AuthorizationRevoked {
		registry.mu.Unlock()
		return false
	}
	connections := registry.revokeLocked(record)
	registry.mu.Unlock()
	closeConnections(connections)
	return true
}

// RevokeSession revokes every authorization for a disconnected session, closes
// its broker-owned connections, and removes authorization tombstones. It is
// idempotent and never affects another session.
func (registry *Registry) RevokeSession(sessionID string) int {
	registry.mu.Lock()
	records := registry.bySession[sessionID]
	connections := make([]net.Conn, 0)
	revoked := 0
	for id, record := range records {
		if record.authorization.State != AuthorizationRevoked {
			revoked++
			connections = append(connections, registry.revokeLocked(record)...)
		}
		delete(registry.byID, id)
	}
	delete(registry.bySession, sessionID)
	delete(registry.byDestination, sessionID)
	delete(registry.byListener, sessionID)
	registry.mu.Unlock()
	closeConnections(connections)
	return revoked
}

func (registry *Registry) lookupLocked(sessionID string, id AuthorizationID) (*authorizationRecord, error) {
	if id == "" {
		return nil, ErrUnknownAuthorization
	}
	record := registry.byID[id]
	if record == nil {
		return nil, ErrUnknownAuthorization
	}
	if record.authorization.SessionID != sessionID {
		return nil, ErrAuthorizationSession
	}
	return record, nil
}

func (registry *Registry) latestDestinationLocked(sessionID string, address string) *authorizationRecord {
	var latest *authorizationRecord
	for _, record := range registry.byDestination[sessionID] {
		if record.authorization.Address != address || record.authorization.State == AuthorizationRevoked {
			continue
		}
		if latest == nil || latest.sequence < record.sequence {
			latest = record
		}
	}
	return latest
}

func (registry *Registry) revokeLocked(record *authorizationRecord) []net.Conn {
	if record.authorization.State == AuthorizationRevoked {
		return nil
	}
	record.authorization.State = AuthorizationRevoked
	record.cancel()

	sessionID := record.authorization.SessionID
	authorizationID := record.authorization.AuthorizationID
	connectionCount := record.pendingConnections + uint64(len(record.connections))
	record.pendingConnections = 0
	registry.releaseConnectionCountLocked(sessionID, connectionCount)
	if record.authorization.HasListenerID {
		if listeners := registry.byListener[sessionID]; listeners != nil {
			if listeners[record.authorization.ImplantListenerID] == record {
				delete(listeners, record.authorization.ImplantListenerID)
			}
			if len(listeners) == 0 {
				delete(registry.byListener, sessionID)
			}
		}
	}
	if destinations := registry.byDestination[sessionID]; destinations != nil {
		delete(destinations, authorizationID)
		if len(destinations) == 0 {
			delete(registry.byDestination, sessionID)
		}
	}
	delete(registry.byID, authorizationID)
	if sessionAuthorizations := registry.bySession[sessionID]; sessionAuthorizations != nil {
		delete(sessionAuthorizations, authorizationID)
		if len(sessionAuthorizations) == 0 {
			delete(registry.bySession, sessionID)
		}
	}

	connections := make([]net.Conn, 0, len(record.connections))
	for connectionID, connection := range record.connections {
		connections = append(connections, connection)
		delete(record.connections, connectionID)
	}
	return connections
}

func closeConnections(connections []net.Conn) {
	for _, connection := range connections {
		_ = connection.Close()
	}
}

func (registry *Registry) newAuthorizationIDLocked() (AuthorizationID, error) {
	buffer := make([]byte, authorizationIDBytes)
	for range maxAuthorizationIDAttempts {
		if _, err := io.ReadFull(registry.random, buffer); err != nil {
			return "", fmt.Errorf("%w: %v", ErrAuthorizationIDGeneration, err)
		}
		id := AuthorizationID(base64.RawURLEncoding.EncodeToString(buffer))
		if registry.byID[id] == nil {
			return id, nil
		}
	}
	return "", ErrAuthorizationIDGeneration
}

func (registry *Registry) initializeLocked() {
	if registry.random == nil {
		registry.random = rand.Reader
	}
	if registry.byID == nil {
		registry.byID = map[AuthorizationID]*authorizationRecord{}
	}
	if registry.bySession == nil {
		registry.bySession = map[string]map[AuthorizationID]*authorizationRecord{}
	}
	if registry.byDestination == nil {
		registry.byDestination = map[string]map[AuthorizationID]*authorizationRecord{}
	}
	if registry.byListener == nil {
		registry.byListener = map[string]map[uint32]*authorizationRecord{}
	}
	if registry.sessionConnectionCounts == nil {
		registry.sessionConnectionCounts = map[string]uint64{}
	}
}

func (registry *Registry) reserve(sessionID string, id AuthorizationID, legacyAddress string) (dialPlan, context.Context, *connectionReservation, error) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	var record *authorizationRecord
	var err error
	if id != "" {
		record, err = registry.lookupLocked(sessionID, id)
	} else {
		canonicalAddress, canonicalErr := canonicalizeAddress(legacyAddress)
		if canonicalErr != nil {
			return dialPlan{}, nil, nil, canonicalErr
		}
		record = registry.latestDestinationLocked(sessionID, canonicalAddress)
		if record == nil {
			err = ErrUnknownAuthorization
		} else if record.authorization.RequiresAuthorizationID {
			err = ErrAuthorizationIDRequired
		}
	}
	if err != nil {
		return dialPlan{}, nil, nil, err
	}
	if record.pendingConnections+uint64(len(record.connections)) >= maxConnectionsPerAuthorization {
		return dialPlan{}, nil, nil, ErrAuthorizationConnectionLimit
	}
	if registry.sessionConnectionCounts[sessionID] >= maxConnectionsPerSession {
		return dialPlan{}, nil, nil, ErrSessionConnectionLimit
	}
	if registry.totalConnectionCount >= maxConnectionsGlobal {
		return dialPlan{}, nil, nil, ErrGlobalConnectionLimit
	}
	plan, err := dialPlanForRecord(record)
	if err != nil {
		return dialPlan{}, nil, nil, err
	}
	record.pendingConnections++
	registry.sessionConnectionCounts[sessionID]++
	registry.totalConnectionCount++
	reservation := &connectionReservation{record: record, active: true}
	return plan, record.revoked, reservation, nil
}

func (registry *Registry) registerConnection(sessionID string, id AuthorizationID, reservation *connectionReservation, connection net.Conn) (uint64, error) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if reservation == nil || !reservation.active || reservation.record == nil {
		return 0, ErrAuthorizationReservation
	}
	record, err := registry.lookupLocked(sessionID, id)
	if err != nil {
		return 0, err
	}
	if record != reservation.record || record.pendingConnections == 0 {
		return 0, ErrAuthorizationReservation
	}
	if record.authorization.State == AuthorizationRevoked {
		return 0, ErrAuthorizationRevoked
	}
	reservation.active = false
	record.pendingConnections--
	record.nextConnID++
	record.connections[record.nextConnID] = connection
	return record.nextConnID, nil
}

func (registry *Registry) releaseReservation(reservation *connectionReservation) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if reservation == nil || !reservation.active || reservation.record == nil {
		return
	}
	reservation.active = false
	record := reservation.record
	if record.authorization.State == AuthorizationRevoked {
		return
	}
	if record.pendingConnections == 0 {
		return
	}
	record.pendingConnections--
	registry.releaseConnectionCountLocked(record.authorization.SessionID, 1)
}

func (registry *Registry) unregisterConnection(record *authorizationRecord, connectionID uint64) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if record == nil {
		return
	}
	if _, ok := record.connections[connectionID]; !ok {
		return
	}
	delete(record.connections, connectionID)
	registry.releaseConnectionCountLocked(record.authorization.SessionID, 1)
}

func (registry *Registry) releaseConnectionCountLocked(sessionID string, count uint64) {
	if count == 0 {
		return
	}
	sessionCount := registry.sessionConnectionCounts[sessionID]
	if count > sessionCount {
		count = sessionCount
	}
	if count == 0 {
		return
	}
	if count >= sessionCount {
		delete(registry.sessionConnectionCounts, sessionID)
	} else {
		registry.sessionConnectionCounts[sessionID] = sessionCount - count
	}
	if count > registry.totalConnectionCount {
		count = registry.totalConnectionCount
	}
	if count >= registry.totalConnectionCount {
		registry.totalConnectionCount = 0
	} else {
		registry.totalConnectionCount -= count
	}
}
