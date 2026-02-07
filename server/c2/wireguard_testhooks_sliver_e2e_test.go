//go:build server && go_sqlite && sliver_e2e

package c2

import "net"

// HandleWGSliverConnectionForTest exposes the raw connection handler to external
// (package `c2_test`) end-to-end tests without shipping test-only symbols in
// production builds.
func HandleWGSliverConnectionForTest(conn net.Conn) {
	handleWGSliverConnection(conn)
}
