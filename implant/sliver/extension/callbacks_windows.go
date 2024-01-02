package extension

type CallbackArray []uintptr

const (
	SendOutputCallback = iota
	SendErrorCallback
)

var callbacks CallbackArray

func init() {
	callbacks = make(CallbackArray, 2)
}
