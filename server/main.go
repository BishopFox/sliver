package main

import (
	"crypto/rand"
	"crypto/sha256"
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"path"
	"sync"
	"time"

	pb "sliver/protobuf"
)

const (
	// randomIDSize - Size of the TunnelID in bytes
	randomIDSize = 8

	logFileName = "sliver.log"
	timeout     = 30 * time.Second
	readBufSize = 1024
)

// Sliver implant
type Sliver struct {
	ID            int
	Name          string
	Hostname      string
	Username      string
	UID           string
	GID           string
	Os            string
	Arch          string
	RemoteAddress string
	PID           int32
	Send          chan pb.Envelope
	Resp          map[string]chan pb.Envelope
}

// Event - Sliver connect/disconnect
type Event struct {
	Sliver    *Sliver
	EventType string
}

var (
	sliverServerVersion = "0.0.3"
	server              *string
	serverLPort         *int

	// Yea I'm lazy, it'd be better not to use mutex
	hiveMutex = &sync.RWMutex{}
	hive      = &map[int]*Sliver{}
	hiveID    = new(int)
)

func main() {
	server = flag.String("server", "", "bind server address")
	serverLPort = flag.Int("server-lport", 8888, "bind listen port")
	forceUnpack := flag.Bool("unpack", false, "force unpack assets")
	version := flag.Bool("version", false, "print version number")
	flag.Parse()

	if *version {
		fmt.Printf("v%s\n", sliverServerVersion)
		os.Exit(0)
	}

	appDir := GetRootAppDir()
	logFile := initLogging(appDir)
	defer logFile.Close()
	if _, err := os.Stat(path.Join(appDir, goDirName)); os.IsNotExist(err) || *forceUnpack {
		fmt.Println(Info + "First time setup, please wait ... ")
		log.Println("Unpacking assets ... ")
		SetupAssets()
	}

	events := make(chan Event, 128)

	log.Println("Starting listeners ...")
	ln, err := startMutualTLSListener(*server, uint16(*serverLPort), events)
	if err != nil {
		log.Printf("Failed to start server")
		fmt.Printf("\r"+Warn+"Failed to start server %v", err)
		return
	}

	defer func() {
		ln.Close()
		hiveMutex.Lock()
		defer hiveMutex.Unlock()
		for _, sliver := range *hive {
			close(sliver.Send)
		}
	}()

	startConsole(events)
}

// Initialize logging
func initLogging(appDir string) *os.File {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	logFile, err := os.OpenFile(path.Join(appDir, logFileName), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	log.SetOutput(logFile)
	return logFile
}

// GetRootAppDir - Get the Sliver app dir ~/.sliver/
func GetRootAppDir() string {
	user, _ := user.Current()
	dir := path.Join(user.HomeDir, ".sliver")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}
	return dir
}

// randomID - Generate random ID of randomIDSize bytes
func randomID() string {
	randBuf := make([]byte, 64) // 64 bytes of randomness
	rand.Read(randBuf)
	digest := sha256.Sum256(randBuf)
	return fmt.Sprintf("%x", digest[:randomIDSize])
}

// getHiveID - Returns an incremental nonce as an id
func getHiveID() int {
	newID := (*hiveID) + 1
	(*hiveID)++
	return newID
}
