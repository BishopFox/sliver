package handlers

var (
	rpcHandlers = map[string]interface{}{}
)

// GetRPCHandlers - Returns a map of server-side msg handlers
func GetRPCHandlers() map[string]interface{} {
	return rpcHandlers
}
