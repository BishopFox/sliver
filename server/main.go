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
	Filename      string
	Send          chan pb.Envelope
	Resp          map[string]chan pb.Envelope
	RespMutex     *sync.RWMutex
}

// Job - Manages background jobs
type Job struct {
	ID       int
	Name     string
	Protocol string
	Port     uint16
	JobCtrl  chan bool
}

// Event - Sliver connect/disconnect
type Event struct {
	Sliver    *Sliver
	Job       *Job
	EventType string
}

var (
	sliverServerVersion = "0.0.4"

	// Yea I'm lazy, it'd be better not to use mutex
	hiveMutex = &sync.RWMutex{}
	hive      = &map[int]*Sliver{}
	hiveID    = new(int)

	// Yea I'm lazy, it'd be better not to use mutex
	jobMutex = &sync.RWMutex{}
	jobs     = &map[int]*Job{}
	jobID    = new(int)
)

func main() {
	unpack := flag.Bool("unpack", false, "force unpack assets")
	version := flag.Bool("version", false, "print version number")
	flag.Parse()

	if *version {
		fmt.Printf("v%s\n", sliverServerVersion)
		os.Exit(0)
	}

	appDir := GetRootAppDir()
	if _, err := os.Stat(path.Join(appDir, goDirName)); os.IsNotExist(err) || *unpack {
		fmt.Println(Info + "First time setup, please wait ... ")
		log.Println("Unpacking assets ... ")
		SetupAssets()
	}

	logFile := initLogging(appDir)
	defer logFile.Close()

	startConsole()
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

// getJobID - Returns an incremental nonce as an id
func getJobID() int {
	newID := (*jobID) + 1
	(*jobID)++
	return newID
}
