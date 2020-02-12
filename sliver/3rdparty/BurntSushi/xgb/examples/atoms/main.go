package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

var (
	flagRequests    int
	flagGOMAXPROCS  int
	flagCpuProfName string
	flagMemProfName string
)

func init() {
	flag.IntVar(&flagRequests, "requests", 100000, "Number of atoms to intern.")
	flag.IntVar(&flagGOMAXPROCS, "cpu", 1, "Value of GOMAXPROCS.")
	flag.StringVar(&flagCpuProfName, "cpuprof", "cpu.prof",
		"Name of CPU profile file.")
	flag.StringVar(&flagMemProfName, "memprof", "mem.prof",
		"Name of memory profile file.")

	flag.Parse()

	runtime.GOMAXPROCS(flagGOMAXPROCS)
}

func seqNames(n int) []string {
	names := make([]string, n)
	for i := range names {
		names[i] = fmt.Sprintf("NAME%d", i)
	}
	return names
}

func main() {
	X, err := xgb.NewConn()
	if err != nil {
		log.Fatal(err)
	}

	names := seqNames(flagRequests)

	fcpu, err := os.Create(flagCpuProfName)
	if err != nil {
		log.Fatal(err)
	}
	defer fcpu.Close()
	pprof.StartCPUProfile(fcpu)
	defer pprof.StopCPUProfile()

	start := time.Now()
	cookies := make([]xproto.InternAtomCookie, flagRequests)
	for i := 0; i < flagRequests; i++ {
		cookies[i] = xproto.InternAtom(X,
			false, uint16(len(names[i])), names[i])
	}
	for _, cookie := range cookies {
		cookie.Reply()
	}
	fmt.Printf("Exec time: %s\n\n", time.Since(start))

	fmem, err := os.Create(flagMemProfName)
	if err != nil {
		log.Fatal(err)
	}
	defer fmem.Close()
	pprof.WriteHeapProfile(fmem)

	memStats := &runtime.MemStats{}
	runtime.ReadMemStats(memStats)

	// This isn't right. I'm not sure what's wrong.
	lastGcTime := time.Unix(int64(memStats.LastGC/1000000000),
		int64(memStats.LastGC-memStats.LastGC/1000000000))

	fmt.Printf("Alloc: %d\n", memStats.Alloc)
	fmt.Printf("TotalAlloc: %d\n", memStats.TotalAlloc)
	fmt.Printf("LastGC: %s\n", lastGcTime)
	fmt.Printf("PauseTotalNs: %d\n", memStats.PauseTotalNs)
	fmt.Printf("PauseNs: %d\n", memStats.PauseNs)
	fmt.Printf("NumGC: %d\n", memStats.NumGC)
}
