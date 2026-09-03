package handlers

import (
	"fmt"
	"testing"
	uuid "uuid"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"google.golang.org/protobuf/proto"
)

func TestPivotSessionEnvelopeRequiresOwningImmediateConnection(t *testing.T) {
	owner := core.NewImplantConnection("mtls", "owner")
	intruder := core.NewImplantConnection("mtls", "intruder")
	pivot := core.NewPivotSession([]*sliverpb.PivotPeer{{PeerID: 1}, {PeerID: 2}})
	pivot.ImplantConn = core.NewImplantConnection(core.PivotTransportName, "pivot")
	pivot.ImmediateImplantConn = owner
	core.PivotSessions.Store(pivot.ID, pivot)
	t.Cleanup(func() {
		owner.Close()
		intruder.Close()
		pivot.ImplantConn.Close()
		core.PivotSessions.CompareAndDelete(pivot.ID, pivot)
	})
	peerEnvelope := &sliverpb.PivotPeerEnvelope{
		PivotSessionID: uuidBytes(uuid.MustParse(pivot.ID)),
		Data:           []byte("not encrypted"),
	}

	if response := sessionEnvelopeHandler(intruder, peerEnvelope); response != nil {
		t.Fatal("non-owning connection received a pivot response")
	}
	if !pivotSessionExists(pivot.ID) {
		t.Fatal("non-owning connection removed the pivot")
	}
	select {
	case <-pivot.ImplantConn.Done():
		t.Fatal("non-owning connection closed the pivot")
	default:
	}

	owner.Close()
	if response := sessionEnvelopeHandler(owner, peerEnvelope); response != nil {
		t.Fatal("closed owning connection received a pivot response")
	}
	if pivotSessionExists(pivot.ID) {
		t.Fatal("closed owning connection retained the pivot")
	}
	select {
	case <-pivot.ImplantConn.Done():
	default:
		t.Fatal("closed owning connection did not close the pivot")
	}
}

func TestPivotPeerFailureHandlerRemovesAuthorizedImmediateChild(t *testing.T) {
	resetPivotFailureTestState(t)

	reporterConn, reporterSession := addPivotFailureTestSession(t, 300, "wg")
	_, unrelatedSession := addPivotFailureTestSession(t, 900, "wg")
	childSession, childPivot := addPivotFailureTestPivotSession(t, 200, 200, 300)
	grandchildSession, grandchildPivot := addPivotFailureTestPivotSession(t, 100, 100, 200, 300)
	unrelatedChildSession, unrelatedChildPivot := addPivotFailureTestPivotSession(t, 800, 800, 900)

	pivotPeerFailureHandler(reporterConn, mustMarshalPivotPeerFailure(t, 200))

	if got := core.Sessions.Get(reporterSession.ID); got == nil {
		t.Fatalf("reporting session was removed")
	}
	if got := core.Sessions.Get(childSession.ID); got != nil {
		t.Fatalf("child session still present after authorized failure")
	}
	if got := core.Sessions.Get(grandchildSession.ID); got != nil {
		t.Fatalf("grandchild session still present after authorized failure")
	}
	if got := core.Sessions.Get(unrelatedSession.ID); got == nil {
		t.Fatalf("unrelated direct session was removed")
	}
	if got := core.Sessions.Get(unrelatedChildSession.ID); got == nil {
		t.Fatalf("unrelated pivot session was removed")
	}
	if pivotSessionExists(childPivot.ID) {
		t.Fatalf("child pivot session still present after authorized failure")
	}
	if pivotSessionExists(grandchildPivot.ID) {
		t.Fatalf("grandchild pivot session still present after authorized failure")
	}
	if !pivotSessionExists(unrelatedChildPivot.ID) {
		t.Fatalf("unrelated pivot session was removed")
	}
	for _, pivot := range []*core.Pivot{childPivot, grandchildPivot} {
		select {
		case <-pivot.ImplantConn.Done():
		default:
			t.Fatalf("removed pivot connection %s remained open", pivot.ID)
		}
	}
	select {
	case <-unrelatedChildPivot.ImplantConn.Done():
		t.Fatal("unrelated pivot connection was closed")
	default:
	}
}

func TestPivotPeerFailureHandlerRejectsNonImmediateChild(t *testing.T) {
	resetPivotFailureTestState(t)

	reporterConn, _ := addPivotFailureTestSession(t, 300, "wg")
	childSession, childPivot := addPivotFailureTestPivotSession(t, 200, 200, 300)
	grandchildSession, grandchildPivot := addPivotFailureTestPivotSession(t, 100, 100, 200, 300)

	pivotPeerFailureHandler(reporterConn, mustMarshalPivotPeerFailure(t, 100))

	if got := core.Sessions.Get(childSession.ID); got == nil {
		t.Fatalf("child session was removed by non-immediate failure report")
	}
	if got := core.Sessions.Get(grandchildSession.ID); got == nil {
		t.Fatalf("grandchild session was removed by non-immediate failure report")
	}
	if !pivotSessionExists(childPivot.ID) {
		t.Fatalf("child pivot session was removed by non-immediate failure report")
	}
	if !pivotSessionExists(grandchildPivot.ID) {
		t.Fatalf("grandchild pivot session was removed by non-immediate failure report")
	}
	for _, pivot := range []*core.Pivot{childPivot, grandchildPivot} {
		select {
		case <-pivot.ImplantConn.Done():
			t.Fatalf("pivot connection %s was closed by non-immediate failure report", pivot.ID)
		default:
		}
	}
}

func TestPivotPeerFailureHandlerRequiresOwningConnectionIdentity(t *testing.T) {
	resetPivotFailureTestState(t)

	ownerConn, _ := addPivotFailureTestSession(t, 300, "wg")
	intruderConn, _ := addPivotFailureTestSession(t, 300, "mtls")
	childSession, childPivot := addPivotFailureTestPivotSession(t, 200, 200, 300)
	childPivot.ImmediateImplantConn = ownerConn

	pivotPeerFailureHandler(intruderConn, mustMarshalPivotPeerFailure(t, 200))

	if got := core.Sessions.Get(childSession.ID); got == nil {
		t.Fatal("same-PeerID connection removed another connection's pivot session")
	}
	if !pivotSessionExists(childPivot.ID) {
		t.Fatal("same-PeerID connection removed another connection's pivot registry entry")
	}
	select {
	case <-childPivot.ImplantConn.Done():
		t.Fatal("same-PeerID connection closed another connection's pivot")
	default:
	}

	pivotPeerFailureHandler(ownerConn, mustMarshalPivotPeerFailure(t, 200))
	if got := core.Sessions.Get(childSession.ID); got != nil {
		t.Fatal("owning connection did not remove its failed pivot session")
	}
	if pivotSessionExists(childPivot.ID) {
		t.Fatal("owning connection did not remove its failed pivot registry entry")
	}
}

func TestSessionRemoveCascadesThroughRegisteredPivotDescendants(t *testing.T) {
	resetPivotFailureTestState(t)

	parentConn, parentSession := addPivotFailureTestSession(t, 300, "wg")
	childSession, childPivot := addPivotFailureTestPivotSession(t, 200, 200, 300)
	grandchildSession, grandchildPivot := addPivotFailureTestPivotSession(t, 100, 100, 200, 300)

	core.Sessions.Remove(parentSession.ID)
	for _, sessionID := range []string{parentSession.ID, childSession.ID, grandchildSession.ID} {
		if got := core.Sessions.Get(sessionID); got != nil {
			t.Fatalf("session %s survived parent removal", sessionID)
		}
	}
	for _, pivot := range []*core.Pivot{childPivot, grandchildPivot} {
		if pivotSessionExists(pivot.ID) {
			t.Fatalf("pivot %s survived parent removal", pivot.ID)
		}
		select {
		case <-pivot.ImplantConn.Done():
		default:
			t.Fatalf("pivot connection %s survived parent removal", pivot.ID)
		}
	}
	select {
	case <-parentConn.Done():
		t.Fatal("removing a session closed its shared parent transport connection")
	default:
	}
}

func TestSessionRemoveClosesPivotParentAndDescendants(t *testing.T) {
	resetPivotFailureTestState(t)

	topLevelConn, topLevelSession := addPivotFailureTestSession(t, 300, "wg")
	pivotSession, pivot := addPivotFailureTestPivotSession(t, 200, 200, 300)
	childSession, childPivot := addPivotFailureTestPivotSession(t, 100, 100, 200, 300)
	pivot.ImmediateImplantConn = topLevelConn
	childPivot.ImmediateImplantConn = pivot.ImplantConn

	core.Sessions.Remove(pivotSession.ID)

	if got := core.Sessions.Get(topLevelSession.ID); got == nil {
		t.Fatal("removing a pivot session removed its shared top-level transport session")
	}
	for _, sessionID := range []string{pivotSession.ID, childSession.ID} {
		if got := core.Sessions.Get(sessionID); got != nil {
			t.Fatalf("session %s survived pivot-parent removal", sessionID)
		}
	}
	for _, removedPivot := range []*core.Pivot{pivot, childPivot} {
		if pivotSessionExists(removedPivot.ID) {
			t.Fatalf("pivot %s survived pivot-parent removal", removedPivot.ID)
		}
		select {
		case <-removedPivot.ImplantConn.Done():
		default:
			t.Fatalf("pivot connection %s survived pivot-parent removal", removedPivot.ID)
		}
	}
	select {
	case <-topLevelConn.Done():
		t.Fatal("removing a pivot session closed its shared top-level transport connection")
	default:
	}
}

func TestPivotPeerFailureClosesUnregisteredImmediateChild(t *testing.T) {
	resetPivotFailureTestState(t)

	reporterConn, _ := addPivotFailureTestSession(t, 300, "wg")
	unregistered := core.NewPivotSession([]*sliverpb.PivotPeer{{PeerID: 200}, {PeerID: 300}})
	unregistered.OriginID = 200
	unregistered.ImplantConn = core.NewImplantConnection(core.PivotTransportName, "unregistered")
	unregistered.ImmediateImplantConn = reporterConn
	core.PivotSessions.Store(unregistered.ID, unregistered)

	unrelated := core.NewPivotSession([]*sliverpb.PivotPeer{{PeerID: 800}, {PeerID: 900}})
	unrelated.OriginID = 800
	unrelated.ImplantConn = core.NewImplantConnection(core.PivotTransportName, "unrelated")
	core.PivotSessions.Store(unrelated.ID, unrelated)
	t.Cleanup(func() {
		unregistered.ImplantConn.Close()
		unrelated.ImplantConn.Close()
		core.PivotSessions.CompareAndDelete(unregistered.ID, unregistered)
		core.PivotSessions.CompareAndDelete(unrelated.ID, unrelated)
	})

	pivotPeerFailureHandler(reporterConn, mustMarshalPivotPeerFailure(t, 200))
	if pivotSessionExists(unregistered.ID) {
		t.Fatal("unregistered immediate child remained in pivot map")
	}
	select {
	case <-unregistered.ImplantConn.Done():
	default:
		t.Fatal("unregistered immediate child connection remained open")
	}
	if !pivotSessionExists(unrelated.ID) {
		t.Fatal("unrelated unregistered pivot was removed")
	}
	select {
	case <-unrelated.ImplantConn.Done():
		t.Fatal("unrelated unregistered pivot connection was closed")
	default:
	}
}

func addPivotFailureTestSession(t *testing.T, peerID int64, transport string) (*core.ImplantConnection, *core.Session) {
	t.Helper()

	conn := core.NewImplantConnection(transport, fmt.Sprintf("peer-%d", peerID))
	t.Cleanup(conn.Close)
	session := core.NewSession(conn)
	session.Name = fmt.Sprintf("peer-%d", peerID)
	session.PeerID = peerID
	core.Sessions.Add(session)
	return conn, session
}

func addPivotFailureTestPivotSession(t *testing.T, originID int64, chain ...int64) (*core.Session, *core.Pivot) {
	t.Helper()

	peers := make([]*sliverpb.PivotPeer, 0, len(chain))
	for _, peerID := range chain {
		peers = append(peers, &sliverpb.PivotPeer{
			PeerID: peerID,
			Name:   fmt.Sprintf("peer-%d", peerID),
		})
	}

	pivot := core.NewPivotSession(peers)
	pivot.OriginID = originID
	pivot.ImplantConn = core.NewImplantConnection(core.PivotTransportName, fmt.Sprintf("pivot-%d", originID))
	if len(chain) > 1 {
		for _, session := range core.Sessions.All() {
			if session.PeerID == chain[1] {
				pivot.ImmediateImplantConn = session.Connection
				break
			}
		}
		if pivot.ImmediateImplantConn == nil {
			t.Fatalf("missing immediate parent session for pivot peer %d", chain[1])
		}
	}
	core.PivotSessions.Store(pivot.ID, pivot)
	t.Cleanup(func() {
		core.ClosePivotSession(pivot.ID)
		pivot.ImplantConn.Close()
	})

	session := core.NewSession(pivot.ImplantConn)
	session.Name = fmt.Sprintf("peer-%d", originID)
	session.PeerID = originID
	core.Sessions.Add(session)

	return session, pivot
}

func mustMarshalPivotPeerFailure(t *testing.T, peerID int64) []byte {
	t.Helper()

	data, err := proto.Marshal(&sliverpb.PivotPeerFailure{
		PeerID: peerID,
		Type:   sliverpb.PeerFailureType_SEND_FAILURE,
	})
	if err != nil {
		t.Fatalf("marshal peer failure: %v", err)
	}
	return data
}

func pivotSessionExists(pivotID string) bool {
	_, ok := core.PivotSessions.Load(pivotID)
	return ok
}

func resetPivotFailureTestState(t *testing.T) {
	t.Helper()

	cleanup := func() {
		for _, session := range core.Sessions.All() {
			core.Sessions.Remove(session.ID)
			if session.Connection != nil {
				session.Connection.Close()
			}
		}
		core.PivotSessions.Range(func(key, value interface{}) bool {
			if pivotID, ok := key.(string); ok {
				core.ClosePivotSession(pivotID)
			}
			return true
		})
	}
	cleanup()
	t.Cleanup(cleanup)
}
