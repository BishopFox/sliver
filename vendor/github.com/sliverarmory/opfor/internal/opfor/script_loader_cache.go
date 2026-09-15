package opfor

import (
	"crypto/sha256"
	"errors"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const scriptLoaderCompilerCacheVersion = "sleep-2.1-opfor-v1"

// ScriptLoaderCache is an explicit, concurrency-safe shared compilation
// capability for Sleep ScriptLoader.setGlobalCache. Runtimes remain isolated
// unless an importer deliberately supplies the same cache to each of them.
// Cached Programs are immutable; loading one still creates independent script
// globals, callbacks, and lifecycle state.
type ScriptLoaderCache struct {
	mu          sync.Mutex
	programs    map[scriptLoaderCacheKey]*Program
	generations map[string]uint64
}

type scriptLoaderCacheKey struct {
	identity      string
	charset       string
	conversion    bool
	environments  string
	generation    uint64
	compiler      string
	contentSHA256 [sha256.Size]byte
}

// NewScriptLoaderCache constructs an empty shared ScriptLoader compilation
// cache. Supplying it to more than one Runtime is the opt-in sharing boundary.
func NewScriptLoaderCache() *ScriptLoaderCache {
	return &ScriptLoaderCache{
		programs:    make(map[scriptLoaderCacheKey]*Program),
		generations: make(map[string]uint64),
	}
}

func canonicalScriptLoaderCacheIdentity(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(name))
}

func (cache *ScriptLoaderCache) compile(
	name string,
	charset string,
	conversion bool,
	environments string,
	data []byte,
	compile func() (*Program, error),
) (*Program, error) {
	if compile == nil {
		return nil, errors.New("opfor: ScriptLoader cache compile function is nil")
	}
	if cache == nil {
		return compile()
	}
	identity := canonicalScriptLoaderCacheIdentity(name)
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.programs == nil {
		cache.programs = make(map[scriptLoaderCacheKey]*Program)
	}
	if cache.generations == nil {
		cache.generations = make(map[string]uint64)
	}
	key := scriptLoaderCacheKey{
		identity:      identity,
		charset:       strings.ToLower(strings.TrimSpace(charset)),
		conversion:    conversion,
		environments:  environments,
		generation:    cache.generations[identity],
		compiler:      scriptLoaderCompilerCacheVersion,
		contentSHA256: sha256.Sum256(data),
	}
	if program := cache.programs[key]; program != nil {
		return program, nil
	}
	program, err := compile()
	if err != nil {
		return nil, err
	}
	cache.programs[key] = program
	return program, nil
}

func (runtime *Runtime) scriptLoaderEnvironmentFingerprint() string {
	if runtime == nil {
		return ""
	}
	runtime.mu.RLock()
	keywords := make([]string, 0, len(runtime.environments))
	for keyword := range runtime.environments {
		keywords = append(keywords, keyword)
	}
	sort.Strings(keywords)
	var fingerprint strings.Builder
	for _, keyword := range keywords {
		fingerprint.WriteString(strconv.Quote(keyword))
		fingerprint.WriteByte('=')
		fingerprint.WriteString(strconv.Itoa(int(runtime.environments[keyword])))
		fingerprint.WriteByte(';')
	}
	runtime.mu.RUnlock()
	return fingerprint.String()
}

func (cache *ScriptLoaderCache) touch(name string) {
	if cache == nil {
		return
	}
	identity := canonicalScriptLoaderCacheIdentity(name)
	cache.mu.Lock()
	if cache.generations == nil {
		cache.generations = make(map[string]uint64)
	}
	cache.generations[identity]++
	for key := range cache.programs {
		if key.identity == identity {
			delete(cache.programs, key)
		}
	}
	cache.mu.Unlock()
}

func (cache *ScriptLoaderCache) len() int {
	if cache == nil {
		return 0
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return len(cache.programs)
}
