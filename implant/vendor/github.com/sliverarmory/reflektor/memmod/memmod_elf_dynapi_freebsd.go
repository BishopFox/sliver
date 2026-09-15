//go:build freebsd && (amd64 || arm64)

package memmod

import (
	"fmt"

	"github.com/ebitengine/purego"
)

func initLinuxDynAPI() error {
	resolve := func(name string) (uintptr, error) {
		address, err := purego.Dlsym(purego.RTLD_DEFAULT, name)
		if err != nil {
			return 0, fmt.Errorf("resolve runtime symbol %s: %w", name, err)
		}
		if address == 0 {
			return 0, fmt.Errorf("resolve runtime symbol %s: symbol address is zero", name)
		}
		return address, nil
	}

	dlopenAddr, err := resolve("dlopen")
	if err != nil {
		return err
	}
	dlsymAddr, err := resolve("dlsym")
	if err != nil {
		return err
	}
	dlerrorAddr, err := resolve("dlerror")
	if err != nil {
		return err
	}
	dlvsymAddr, err := resolve("dlvsym")
	if err != nil {
		return err
	}
	dlcloseAddr, err := resolve("dlclose")
	if err != nil {
		return err
	}

	linuxAPI = linuxDynAPI{
		dlopen:        dlopenAddr,
		dlsym:         dlsymAddr,
		dlvsym:        dlvsymAddr,
		dlclose:       dlcloseAddr,
		dlerror:       dlerrorAddr,
		defaultHandle: purego.RTLD_DEFAULT,
	}
	return nil
}
