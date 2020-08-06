// +build windows

package memprofile

import (
	"fmt"

	"golang.org/x/sys/windows"
)

var profiles []*MemProfile

func GetProfile(name string) (p *MemProfile, err error) {
	for _, prof := range profiles {
		if prof.Name == name {
			p = prof
			break
		}
	}
	if p == nil {
		err = fmt.Errorf("profile %s not found", name)
	}
	return
}

func AddProfile(p *MemProfile) {
	profiles = append(profiles, p)
}

func GetDefault() (*MemProfile, error) {
	return GetProfile("default")
}

func (m *MemProfile) Inject(hProc windows.Handle, data []byte, args []byte, rwx bool) error {

}

func init() {
	basicAlloc := BasicAllocator{}
	exec := ThreadExecutor{}
	AddProfile(&MemProfile{
		Allocator: basicAlloc,
		Executor:  exec,
		Name:      "default",
	})
}
