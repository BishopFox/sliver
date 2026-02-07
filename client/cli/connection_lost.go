package cli

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

func handleConnectionLost(ln *grpc.ClientConn) {
	if ln == nil {
		return
	}
	currentState := ln.GetState()
	// currentState should be "Ready" when the connection is established.
	if ln.WaitForStateChange(context.Background(), currentState) {
		newState := ln.GetState()
		// newState will be "Idle" if the connection is lost.
		if newState == connectivity.Idle {
			fmt.Println("\nLost connection to server. Exiting now.")
			os.Exit(1)
		}
	}
}
