package carapace

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/rsteube/carapace/internal/env"
	"github.com/rsteube/carapace/internal/uid"
	"github.com/rsteube/carapace/pkg/ps"
)

var LOG = log.New(ioutil.Discard, "", log.Flags())

func init() {
	if !env.Log() {
		return
	}

	tmpdir := fmt.Sprintf("%v/carapace", os.TempDir())
	if err := os.MkdirAll(tmpdir, os.ModePerm); err != nil {
		log.Fatal(err.Error())
	}

	file := fmt.Sprintf("%v/%v.log", tmpdir, uid.Executable())
	if logfileWriter, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o666); err != nil {
		log.Fatal(err.Error())
	} else {
		LOG = log.New(logfileWriter, ps.DetermineShell()+" ", log.Flags()|log.Lmsgprefix|log.Lmicroseconds)
	}
}
