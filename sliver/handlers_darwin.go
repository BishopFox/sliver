package main

var (
	darwinHandlers = map[string]interface{}{
		"task": taskHandler,
		"ping": pingHandler,
	}
)

func getSystemHandlers() map[string]interface{} {
	return darwinHandlers
}

func taskHandler(msg []byte) {

}
