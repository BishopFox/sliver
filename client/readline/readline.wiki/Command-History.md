
## Multiple history sources

Two different history sources can be plugged on the shell:
- A main one, used with `Ctrl-R`
- A secondary one, used with `Ctrl-E`

I did this because for some projects were users might need multiple clients connected to a server at once,
they can either have a user-centralized history record, and an in-memory one for the console lifetime.

*Note that the history is automatically put in Search Mode, so that what you type is passed to the search filter.*

![history](https://github.com/bishopfox/sliver/client/readline/blob/assets/history.gif)


## Writing a custom history

You can write a custom history and bind it to one of the two history sources if it satisfies this interface:

```go
type History interface {
	// Append takes the line and returns an updated number of lines or an error
	Write(string) (int, error)

	// GetLine takes the historic line number and returns the line or an error
	GetLine(int) (string, error)

	// Len returns the number of history lines
	Len() int

	// Dump returns everything in readline. The return is an interface{} because
	// not all LineHistory implementations will want to structure the history in
	// the same way. And since Dump() is not actually used by the readline API
	// internally, this methods return can be structured in whichever way is most
	// convenient for your own applications (or even just create an empty
	//function which returns `nil` if you don't require Dump() either)
	Dump() interface{}
}
```

The following code shows how to write a history satisfying the `History` interface, that I wrote for
sending history to a gRPC server, which would store and filter commands for a given user.

```go
// ClientHistory - Writes and queries only the Client's history
type ClientHistory struct {
	LinesSinceStart int // Keeps count of line since session
	items           []string
}

// Write - Sends the last command to the server for saving
func (h *ClientHistory) Write(s string) (int, error) {

	res, err := transport.RPC.AddToHistory(context.Background(),
		&clientpb.AddCmdHistoryRequest{Line: s})
	if err != nil {
		return 0, err
	}

	// The server sent us back the whole user history,
	// so we give it to the user history (the latter never
	// actually uses its Write() method.
	UserHist.cache = res.Lines

	h.items = append(h.items, s)
	return len(h.items), nil
}

// GetLine returns a line from history
func (h *ClientHistory) GetLine(i int) (string, error) {
	if len(h.items) == 0 {
		return "", nil
	}
	return h.items[i], nil
}

// Len returns the number of lines in history
func (h *ClientHistory) Len() int {
	return len(h.items)
}

// Dump returns the entire history
func (h *ClientHistory) Dump() interface{} {
	return h.items
}
```
