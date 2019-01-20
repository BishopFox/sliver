package main

var (
	linuxHandlers = map[string]interface{}{
		"task": taskHandler,
		"ping": pingHandler,
	}
)

func getSystemHandlers() map[string]interface{} {
	return linuxHandlers
}

func taskHandler(msg []byte) {

}
