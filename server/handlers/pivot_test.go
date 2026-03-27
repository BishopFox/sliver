package handlers

import (
	"fmt"
	"testing"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"google.golang.org/protobuf/proto"
)

func TestPivotPeerFailureHandlerRemovesAuthorizedImmediateChild(t *testing.T) {
	resetPivotFailureTestState(t)

	reporterConn, reporterSession := addTestSession(t, 300, "wg")
	_, unrelatedSession := addTestSession(t, 900, "wg")
	childSession, childPivot := addTestPivotSession(t, 200, 200, 300)
	grandchildSession, grandchildPivot := addTestPivotSession(t, 100, 100, 200, 300)
	unrelatedChildSession, unrelatedChildPivot := addTestPivotSession(t, 800, 800, 900)

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
}

func TestPivotPeerFailureHandlerRejectsNonImmediateChild(t *testing.T) {
	resetPivotFailureTestState(t)

	reporterConn, _ := addTestSession(t, 300, "wg")
	childSession, childPivot := addTestPivotSession(t, 200, 200, 300)
	grandchildSession, grandchildPivot := addTestPivotSession(t, 100, 100, 200, 300)

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
}

func addTestSession(t *testing.T, peerID int64, transport string) (*core.ImplantConnection, *core.Session) {
	t.Helper()

	conn := core.NewImplantConnection(transport, fmt.Sprintf("peer-%d", peerID))
	session := core.NewSession(conn)
	session.Name = fmt.Sprintf("peer-%d", peerID)
	session.PeerID = peerID
	core.Sessions.Add(session)
	return conn, session
}

func addTestPivotSession(t *testing.T, originID int64, chain ...int64) (*core.Session, *core.Pivot) {
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
	core.PivotSessions.Store(pivot.ID, pivot)

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

	for _, session := range core.Sessions.All() {
		core.Sessions.Remove(session.ID)
	}
	core.PivotSessions.Range(func(key, value interface{}) bool {
		core.PivotSessions.Delete(key)
		return true
	})

	t.Cleanup(func() {
		for _, session := range core.Sessions.All() {
			core.Sessions.Remove(session.ID)
		}
		core.PivotSessions.Range(func(key, value interface{}) bool {
			core.PivotSessions.Delete(key)
			return true
		})
	})
}
