//go:build linux && !android && (386 || amd64 || (arm && arm.7) || arm64 || ppc64le || riscv64)

package memmod

import "fmt"

func initLinuxDynAPI() error {
	modules, err := runtimeModules()
	if err != nil {
		return err
	}

	dlopenAddr, err := resolveRuntimeAPISymbol(modules, "dlopen")
	if err != nil {
		return fmt.Errorf("resolve runtime symbol dlopen: %w", err)
	}
	dlsymAddr, err := resolveRuntimeAPISymbol(modules, "dlsym")
	if err != nil {
		return fmt.Errorf("resolve runtime symbol dlsym: %w", err)
	}
	dlerrorAddr, err := resolveRuntimeAPISymbol(modules, "dlerror")
	if err != nil {
		return fmt.Errorf("resolve runtime symbol dlerror: %w", err)
	}
	dlvsymAddr, _ := resolveRuntimeAPISymbol(modules, "dlvsym")
	dlcloseAddr, _ := resolveRuntimeAPISymbol(modules, "dlclose")

	linuxAPI = linuxDynAPI{
		dlopen:        dlopenAddr,
		dlsym:         dlsymAddr,
		dlvsym:        dlvsymAddr,
		dlclose:       dlcloseAddr,
		dlerror:       dlerrorAddr,
		defaultHandle: 0,
	}
	return nil
}
