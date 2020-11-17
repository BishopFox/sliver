package cloudflare

// RouteRoot represents the name of the route namespace
type RouteRoot string

const (
	// AccountRouteRoot is the accounts route namespace
	AccountRouteRoot RouteRoot = "accounts"

	// ZoneRouteRoot is the zones route namespace
	ZoneRouteRoot RouteRoot = "zones"

	// Used for testing
	accountID = "01a7362d577a6c3019a474fd6f485823"
	zoneID    = "d56084adb405e0b7e32c52321bf07be6"
)
