package main

var (
	darwinHandlers = map[string]interface{}{
		"task": taskHandler,
	}
)

func getSystemHandlers() map[string]interface{} {
	return darwinHandlers
}

func taskHandler(msg []byte) {

}
