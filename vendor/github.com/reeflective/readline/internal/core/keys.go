package core

import (
	"errors"
	"io"
	"os"
	"regexp"
	"sync"

	"github.com/reeflective/readline/inputrc"
	"github.com/reeflective/readline/internal/strutil"
)

const (
	keyScanBufSize = 1024
)

// Stdin is used by the Keys struct to read and write keys.
// It can be overwritten to use other file descriptors or
// custom io.Readers, such as the one used on Windows.
var Stdin io.ReadCloser = os.Stdin

var rxRcvCursorPos = regexp.MustCompile(`\x1b\[([0-9]+);([0-9]+)R`)

// Keys is used read, manage and use keys input by the shell user.
type Keys struct {
	buf       []byte      // Keys read and waiting to be used.
	matched   []rune      // Keys that have been successfully matched against a bind.
	macroKeys []rune      // Keys that have been fed by a macro.
	mustWait  bool        // Keys are in the stack, but we must still read stdin.
	waiting   bool        // Currently waiting for keys on stdin.
	reading   bool        // Currently reading keys out of the main loop.
	keysOnce  chan []byte // Passing keys from the main routine.
	cursor    chan []byte // Cursor coordinates has been read on stdin.
	resize    chan bool   // Resize events on Windows are sent on stdin.

	cfg   *inputrc.Config // Configuration file used for meta key settings
	mutex sync.RWMutex    // Concurrency safety
}

// WaitAvailableKeys waits until an input key is either read from standard input,
// or directly returns if the key stack still/already has available keys.
func WaitAvailableKeys(keys *Keys, cfg *inputrc.Config) {
	keys.cfg = cfg

	if len(keys.buf) > 0 && !keys.mustWait {
		return
	}

	// The macro engine might have fed some keys
	if len(keys.macroKeys) > 0 {
		return
	}

	keys.mutex.Lock()
	keys.waiting = true
	keys.cursor = make(chan []byte)
	keys.mutex.Unlock()

	defer func() {
		keys.mutex.Lock()
		keys.waiting = false
		keys.mutex.Unlock()
	}()

	for {
		// Start reading from os.Stdin in the background.
		// We will either read keyBuf from user, or an EOF
		// send by ourselves, because we pause reading.
		keyBuf, err := keys.readInputFiltered()
		if err != nil && errors.Is(err, io.EOF) {
			return
		}

		if len(keyBuf) == 0 {
			continue
		}

		switch {
		case keys.reading:
			keys.keysOnce <- keyBuf
			continue

		default:
			// When convert-meta is on, any meta-prefixed bind should
			// be stripped and replaced with an escape meta instead.
			if keys.cfg != nil && keys.cfg.GetBool("convert-meta") {
				keyBuf = []byte(strutil.ConvertMeta([]rune(string(keyBuf))))
			}

			keys.mutex.RLock()
			keys.buf = append(keys.buf, keyBuf...)
			keys.mutex.RUnlock()
		}

		return
	}
}

// PopKey is used to pop a key off the key stack without
// yet marking this key as having matched a bind command.
func PopKey(keys *Keys) (key byte, empty bool) {
	switch {
	case len(keys.buf) > 0:
		key = keys.buf[0]
		keys.buf = keys.buf[1:]
	case len(keys.macroKeys) > 0:
		key = byte(keys.macroKeys[0])
		keys.macroKeys = keys.macroKeys[1:]
	default:
		return byte(0), true
	}

	return key, false
}

// PeekKey returns the first key in the stack, without removing it.
func PeekKey(keys *Keys) (key byte, empty bool) {
	switch {
	case len(keys.buf) > 0:
		key = keys.buf[0]
	case len(keys.macroKeys) > 0:
		key = byte(keys.macroKeys[0])
	default:
		return byte(0), true
	}

	return key, false
}

// MatchedKeys is used to indicate how many keys have been evaluated against the shell
// commands in the dispatching process (regardless of if a command was matched or not).
// This function should normally not be used by external users of the library.
func MatchedKeys(keys *Keys, matched []byte, args ...byte) {
	if len(matched) > 0 {
		keys.matched = []rune(string(matched))
	}

	if len(args) > 0 {
		keys.buf = append(args, keys.buf...)
	}

	keys.mustWait = false
}

// MatchedPrefix is similar to MatchedKeys, except that the provided keys
// should not be flushed, since they only matched some binds by prefix and
// that we need more keys for an exact match (or failure).
func MatchedPrefix(keys *Keys, prefix ...byte) {
	if len(prefix) == 0 {
		return
	}

	keys.mutex.Lock()
	defer keys.mutex.Unlock()

	// Our keys are still considered unread, but they have been:
	// if there is no more keys in the stack, the next blocking
	// call to WaitAvailableKeys() should block for new keys.
	keys.mustWait = len(keys.buf) == 0
	keys.buf = append(prefix, keys.buf...)
	keys.matched = []rune(string(prefix))
}

// PopForce is used to force-remove a key from the buffer, without marking
// it as having matched a bind command. This is used, for example, when the
// escape has been handled specially as a Vim escape.
func PopForce(keys *Keys) (key byte, empty bool) {
	switch {
	case len(keys.buf) > 0:
		key = keys.buf[0]
		keys.buf = keys.buf[1:]
	case len(keys.macroKeys) > 0:
		key = byte(keys.macroKeys[0])
		keys.macroKeys = keys.macroKeys[1:]
	default:
		return byte(0), true
	}

	// Force the macro recorder to use the matched keys.
	keys.mustWait = false

	return key, false
}

// MacroKeys returns the keys that have matched a given command, and thus can be recorded
// as a part of the current macro. This function is different from keys.Caller() in that it
// won't return keys that have only matched a prefix, to avoid recording them twice.
func MacroKeys(keys *Keys) []rune {
	if keys.mustWait {
		return nil
	}

	return keys.matched
}

// FlushUsed drops the keys that have matched a given command.
func FlushUsed(keys *Keys) {
	keys.mutex.Lock()
	keys.matched = nil
	defer keys.mutex.Unlock()
}

// ReadKey reads keys from stdin like Read(), but immediately
// returns them instead of storing them in the stack, along with
// an indication on whether this key is an escape/abort one.
func (k *Keys) ReadKey() (key rune, isAbort bool) {
	k.mutex.RLock()
	k.keysOnce = make(chan []byte)
	k.reading = true
	k.mutex.RUnlock()

	defer func() {
		k.mutex.RLock()
		k.reading = false
		k.mutex.RUnlock()
	}()

	switch {
	case len(k.macroKeys) > 0:
		key = k.macroKeys[0]
		k.macroKeys = k.macroKeys[1:]

	case k.waiting:
		buf := <-k.keysOnce
		key = []rune(string(buf))[0]
	default:
		buf, _ := k.readInputFiltered()
		key = []rune(string(buf))[0]
	}

	// Always mark those keys as matched, so that
	// if the macro engine is recording, it will
	// capture them
	k.matched = append(k.matched, key)

	return key, key == inputrc.Esc
}

// Pop removes the first byte in the key stack (first read) and returns it.
// It returns either a key and the empty boolean set to false, or if no keys
// are present, returns a zero rune and empty set to true.
// The key bytes returned by this function are not those that have been
// matched against the current command. The keys returned here are only
// keys that have not yet been dispatched. (ex: di" will match vim delete-to,
// then select-inside, but the quote won't match a command and will be passed
// to select-inside. This function Pop() will thus return the quote.)
func (k *Keys) Pop() (key byte, empty bool) {
	switch {
	case len(k.buf) > 0:
		key = k.buf[0]
		k.buf = k.buf[1:]
	case len(k.macroKeys) > 0:
		key = byte(k.macroKeys[0])
		k.macroKeys = k.macroKeys[1:]
	default:
		return byte(0), true
	}

	k.matched = append(k.matched, rune(key))

	return key, false
}

// Caller returns the keys that have matched the command currently being ran.
func (k *Keys) Caller() (keys []rune) {
	return k.matched
}

// Feed can be used to directly add keys to the stack.
// If begin is true, the keys are added on the top of
// the stack, otherwise they are being appended to it.
func (k *Keys) Feed(begin bool, keys ...rune) {
	if len(keys) == 0 {
		return
	}

	keyBuf := []rune(string(keys))

	k.mutex.Lock()
	defer k.mutex.Unlock()

	if begin {
		k.macroKeys = append(keyBuf, k.macroKeys...)
	} else {
		k.macroKeys = append(k.macroKeys, keyBuf...)
	}
}

func (k *Keys) extractCursorPos(keys []byte) (cursor, remain []byte) {
	if !rxRcvCursorPos.Match(keys) {
		return cursor, keys
	}

	allCursors := rxRcvCursorPos.FindAll(keys, -1)
	cursor = allCursors[len(allCursors)-1]
	remain = rxRcvCursorPos.ReplaceAll(keys, nil)

	return
}
