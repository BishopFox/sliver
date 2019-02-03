package handlers

var (
	clientHandlers = map[string]interface{}{}
)

// GetClientHandlers - Returns a map of server-side msg handlers
func GetClientHandlers() map[string]interface{} {
	return clientHandlers
}
