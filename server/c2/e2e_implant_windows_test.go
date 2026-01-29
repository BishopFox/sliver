//go:build windows

package c2_test

import implantHandlers "github.com/bishopfox/sliver/implant/sliver/handlers"

func dispatchHandler(handler implantHandlers.RPCHandler, data []byte, resp implantHandlers.RPCResponse) {
	implantHandlers.WrapperHandler(handler, data, resp)
}
