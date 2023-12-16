package carapace

import (
	"fmt"
	"strings"
	"sync"

	"github.com/rsteube/carapace/internal/common"
	"github.com/rsteube/carapace/internal/uid"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// TODO storage needs better naming and structure

type entry struct {
	flag          ActionMap
	flagMutex     sync.RWMutex
	positional    []Action
	positionalAny *Action
	dash          []Action
	dashAny       *Action
	preinvoke     func(cmd *cobra.Command, flag *pflag.Flag, action Action) Action
	prerun        func(cmd *cobra.Command, args []string)
	bridged       bool
	initialized   bool
}

type _storage map[*cobra.Command]*entry

var storageMutex sync.RWMutex

func (s _storage) get(cmd *cobra.Command) *entry {
	storageMutex.RLock()
	e, ok := s[cmd]
	storageMutex.RUnlock()

	if !ok {
		storageMutex.Lock()
		defer storageMutex.Unlock()
		if e, ok = s[cmd]; !ok {
			e = &entry{}
			s[cmd] = e
		}
	}
	return e
}

var bridgeMutex sync.Mutex

func (s _storage) bridge(cmd *cobra.Command) {
	if entry := storage.get(cmd); !entry.bridged {
		bridgeMutex.Lock()
		defer bridgeMutex.Unlock()

		if entry := storage.get(cmd); !entry.bridged {
			cobra.OnInitialize(func() {
				if !entry.initialized {
					bridgeMutex.Lock()
					defer bridgeMutex.Unlock()

					if !entry.initialized {
						registerValidArgsFunction(cmd)
						registerFlagCompletion(cmd)
						entry.initialized = true
					}

				}
			})
			entry.bridged = true
		}
	}
}

func (s _storage) hasFlag(cmd *cobra.Command, name string) bool {
	if flag := cmd.LocalFlags().Lookup(name); flag == nil && cmd.HasParent() {
		return s.hasFlag(cmd.Parent(), name)
	} else {
		entry := s.get(cmd)
		entry.flagMutex.RLock()
		defer entry.flagMutex.RUnlock()
		_, ok := entry.flag[name]
		return ok
	}
}

func (s _storage) getFlag(cmd *cobra.Command, name string) Action {
	if flag := cmd.LocalFlags().Lookup(name); flag == nil && cmd.HasParent() {
		return s.getFlag(cmd.Parent(), name)
	} else {
		entry := s.get(cmd)
		entry.flagMutex.RLock()
		defer entry.flagMutex.RUnlock()

		flagAction, ok := entry.flag[name]
		if !ok {
			if f, ok := cmd.GetFlagCompletionFunc(name); ok {
				flagAction = ActionCobra(f)
			}
		}

		a := s.preinvoke(cmd, flag, flagAction)

		return ActionCallback(func(c Context) Action { // TODO verify order of execution is correct
			invoked := a.Invoke(c)
			if invoked.meta.Usage == "" {
				invoked.meta.Usage = flag.Usage
			}
			return invoked.ToA()
		})
	}
}

func (s _storage) preRun(cmd *cobra.Command, args []string) {
	if entry := s.get(cmd); entry.prerun != nil {
		LOG.Printf("executing PreRun for %#v with args %#v", cmd.Name(), args)
		entry.prerun(cmd, args)
	}
}

func (s _storage) preinvoke(cmd *cobra.Command, flag *pflag.Flag, action Action) Action {
	a := action
	if entry := s.get(cmd); entry.preinvoke != nil {
		a = ActionCallback(func(c Context) Action {
			return entry.preinvoke(cmd, flag, action)
		})
	}

	if cmd.HasParent() {
		return s.preinvoke(cmd.Parent(), flag, a)
	}
	return a
}

func (s _storage) hasPositional(cmd *cobra.Command, index int) bool {
	entry := s.get(cmd)
	isDash := common.IsDash(cmd)

	// TODO fallback to cobra defined completion if exists

	switch {
	case !isDash && len(entry.positional) > index:
		return true
	case !isDash:
		return entry.positionalAny != nil
	case len(entry.dash) > index:
		return true
	default:
		return entry.dashAny != nil
	}
}

func (s _storage) getPositional(cmd *cobra.Command, index int) Action {
	entry := s.get(cmd)
	isDash := common.IsDash(cmd)

	var a Action
	switch {
	case !isDash && len(entry.positional) > index:
		a = entry.positional[index]
	case !isDash:
		if entry.positionalAny != nil {
			a = *entry.positionalAny
		} else {
			a = ActionCobra(cmd.ValidArgsFunction)
		}
	case len(entry.dash) > index:
		a = entry.dash[index]
	default:
		if entry.dashAny != nil {
			a = *entry.dashAny
		} else {
			a = ActionCobra(cmd.ValidArgsFunction)
		}
	}
	a = s.preinvoke(cmd, nil, a)

	return ActionCallback(func(c Context) Action {
		invoked := a.Invoke(c)
		if invoked.meta.Usage == "" && len(strings.Fields(cmd.Use)) > 1 {
			invoked.meta.Usage = cmd.Use
		}
		return invoked.ToA()
	})
}

func (s _storage) check() []string {
	errors := make([]string, 0)
	for cmd, entry := range s {
		func() {
			entry.flagMutex.RLock()
			defer entry.flagMutex.RUnlock()

			for name := range entry.flag {
				if flag := cmd.LocalFlags().Lookup(name); flag == nil {
					errors = append(errors, fmt.Sprintf("unknown flag for %s: %s\n", uid.Command(cmd), name))
				}
			}
		}()
	}
	return errors
}

var storage = make(_storage)
