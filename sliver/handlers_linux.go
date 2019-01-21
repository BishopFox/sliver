package main

var (
	linuxHandlers = map[string]interface{}{
		"ping": pingHandler,
	}
)

func getSystemHandlers() map[string]interface{} {
	return linuxHandlers
}
