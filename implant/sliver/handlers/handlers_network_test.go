package handlers

import (
	"testing"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

func TestNetstatHandlerRespondsWithoutTCP(t *testing.T) {
	request, err := proto.Marshal(&sliverpb.NetstatReq{UDP: true})
	if err != nil {
		t.Fatal(err)
	}

	called := false
	netstatHandler(request, func(data []byte, handlerErr error) {
		called = true
		if handlerErr != nil {
			t.Fatalf("netstatHandler callback error: %v", handlerErr)
		}
		response := &sliverpb.Netstat{}
		if err := proto.Unmarshal(data, response); err != nil {
			t.Fatalf("decode netstat response: %v", err)
		}
	})
	if !called {
		t.Fatal("netstatHandler did not respond to a request without TCP")
	}
}
