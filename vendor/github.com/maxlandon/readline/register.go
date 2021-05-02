package readline

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/evilsocket/islazy/tui"
)

// registers - Contains all memory registers resulting from delete/paste/search
// or other operations in the command line input.
type registers struct {
	unnamed            []rune            // Unnamed register, used by default
	num                map[int][]rune    // numbered registers (0-9)
	alpha              map[string][]rune // lettered registers ( a-z )
	ro                 map[string][]rune // read-only registers ( . % : )
	registerSelectWait bool              // The user wants to use a still unidentified register
	onRegister         bool              // We have identified the register, and acting on it.
	currentRegister    rune              // Any of the read/write registers ("/num/alpha)
	mutex              *sync.Mutex
}

func (rl *Instance) initRegisters() {
	rl.registers = &registers{
		num:   make(map[int][]rune, 10),
		alpha: make(map[string][]rune, 52),
		ro:    map[string][]rune{},
		mutex: &sync.Mutex{},
	}
}

// saveToRegister - Passing a function that will move around the line in the desired way, we get
// the number of Vim iterations and we save the resulting string to the appropriate buffer.
// It's the same as saveToRegisterTokenize, but without the need to generate tokenized &
// cursor-pos-actualized versions of the input line.
func (rl *Instance) saveToRegister(adjust int) {

	// Get the current cursor position and go the length specified.
	var begin = rl.pos
	var end = rl.pos
	end += adjust
	if end > len(rl.line)-1 {
		end = len(rl.line)
	} else if end < 0 {
		end = 0
	}

	var buffer []rune
	if end < begin {
		buffer = rl.line[end:begin]
	} else {
		buffer = rl.line[begin:end]
	}

	// Make an immutable copy of the buffer before saving it
	buf := string(buffer)

	// Put the buffer in the appropriate registers.
	// By default, always in the unnamed one first.
	rl.saveBufToRegister([]rune(buf))
}

// saveToRegisterTokenize - Passing a function that will move around the line in the desired way, we get
// the number of Vim iterations and we save the resulting string to the appropriate buffer. Because we
// need the cursor position to be really moved around between calls to the jumper, we also need the tokeniser.
func (rl *Instance) saveToRegisterTokenize(tokeniser tokeniser, jumper func(tokeniser) int, vii int) {

	// The register is going to have to heavily manipulate the cursor position.
	// Remember the original one first, for the end.
	var beginPos = rl.pos

	// Get the current cursor position and go the length specified.
	var begin = rl.pos
	for i := 1; i <= vii; i++ {
		rl.moveCursorByAdjust(jumper(tokeniser))
	}
	var end = rl.pos
	rl.pos = beginPos

	if end > len(rl.line)-1 {
		end = len(rl.line)
	} else if end < 0 {
		end = 0
	}

	var buffer []rune
	if end < begin {
		buffer = rl.line[end:begin]
	} else {
		buffer = rl.line[begin:end]
	}

	// Make an immutable copy of the buffer before saving it
	buf := string(buffer)

	// Put the buffer in the appropriate registers.
	// By default, always in the unnamed one first.
	rl.saveBufToRegister([]rune(buf))
}

// saveBufToRegister - Instead of computing the buffer ourselves based on an adjust,
// let the caller pass directly this buffer, yet relying on the register system to
// determine which register will store the buffer.
func (rl *Instance) saveBufToRegister(buffer []rune) {

	// We must make an immutable version of the buffer first.
	buf := string(buffer)

	// When exiting this function the currently selected register is dropped,
	defer rl.registers.resetRegister()

	// If the buffer is empty, just return
	if len(buffer) == 0 || buf == "" {
		return
	}

	// Put the buffer in the appropriate registers.
	// By default, always in the unnamed one first.
	rl.registers.unnamed = []rune(buf)

	// If there is an active register, directly give it the buffer.
	// Check if its a numbered or lettered register, and put it in.
	if rl.registers.onRegister {
		num, err := strconv.Atoi(string(rl.registers.currentRegister))
		if err == nil && num < 10 {
			rl.registers.writeNumberedRegister(num, []rune(buf), false)
		} else if err != nil {
			rl.registers.writeAlphaRegister([]rune(buf))
		}
	} else {
		// Or, if no active register and if there is room on the numbered ones,
		rl.registers.writeNumberedRegister(0, []rune(buf), true)
	}
}

// The user asked to paste a buffer onto the line, so we check from which register
// we are supposed to select the buffer, and return it to the caller for insertion.
func (rl *Instance) pasteFromRegister() (buffer []rune) {

	// When exiting this function the currently selected register is dropped,
	defer rl.registers.resetRegister()

	// If no actively selected register, return the unnamed buffer
	if !rl.registers.registerSelectWait && !rl.registers.onRegister {
		return rl.registers.unnamed
	}
	activeRegister := string(rl.registers.currentRegister)

	// Else find the active register, and return its content.
	num, err := strconv.Atoi(activeRegister)

	// Either from the numbered ones.
	if err == nil {
		buf, found := rl.registers.num[num]
		if found {
			return buf
		}
		return
	}
	// or the lettered ones
	buf, found := rl.registers.alpha[activeRegister]
	if found {
		return buf
	}
	// Or the read-only ones
	buf, found = rl.registers.ro[activeRegister]
	if found {
		return buf
	}

	return
}

// setActiveRegister - The user has typed "<regiserID>, and we don't know yet
// if we are about to copy to/from it, so we just set as active, so that when
// the action to perform on it will be asked, we know which one to use.
func (r *registers) setActiveRegister(reg rune) {
	defer func() {
		// We now have an active, identified register
		r.registerSelectWait = false
		r.onRegister = true
	}()

	// Numbered
	num, err := strconv.Atoi(string(reg))
	if err == nil && num < 10 {
		r.currentRegister = reg
		return
	}
	// Read-only
	_, found := r.ro[string(reg)]
	if found {
		r.currentRegister = reg
		return
	}

	// Else, lettered
	r.currentRegister = reg
}

// writeNumberedRegister - Add a buffer to one of the numbered registers
// Pass a number above 10 to indicate we just push it on the num stack.
func (r *registers) writeNumberedRegister(idx int, buf []rune, push bool) {
	// No numbered register above 10
	if len(r.num) > 10 {
		return
	}
	// No push to the stack if we are already using 9
	var max int
	if push {
		for i := range r.num {
			if i > max && string(r.num[i]) != string(buf) {
				max = i
			}
		}
		if max < 9 {
			r.num[max+1] = buf
		}
	} else {
		// Add to the stack with the specified register
		r.num[idx] = buf
	}
}

// writeAlphaRegister - Either adds a buffer to a new/existing letterd register,
// or appends to this new/existing register if the currently active register is
// the uppercase letter for this register.
func (r *registers) writeAlphaRegister(buffer []rune) {
	appendRegs := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	appended := false
	for _, char := range appendRegs {
		if char == r.currentRegister {
			real := strings.ToLower(string(r.currentRegister))
			_, exists := r.alpha[real]
			if exists {
				r.alpha[real] = append(r.alpha[real], buffer...)
			} else {
				r.alpha[real] = buffer
			}
			appended = true
		}
	}
	if !appended {
		r.alpha[string(r.currentRegister)] = buffer
	}
}

// resetRegister - there is no currently active register anymore,
// nor we are currently setting one as active.
func (r *registers) resetRegister() {
	r.currentRegister = ' '
	r.registerSelectWait = false
	r.onRegister = false
}

// The user can show registers completions and insert, no matter the cursor position.
func (rl *Instance) completeRegisters() (groups []*CompletionGroup) {

	// We set the hint exceptionally
	hint := BLUE + "-- registers --" + RESET
	rl.hintText = []rune(hint)

	// Make the groups
	anonRegs := &CompletionGroup{
		DisplayType:  TabDisplayMap,
		MaxLength:    20,
		Descriptions: map[string]string{},
	}

	// Unnamed (the added space is because we must have a unique key.
	// This space is trimmed when the buffer is being passed to users)
	anonRegs.Suggestions = append(anonRegs.Suggestions, string(rl.registers.unnamed))
	anonRegs.Descriptions[string(rl.registers.unnamed)] = DIM + "\"" + "\"" + RESET

	groups = append(groups, anonRegs)

	// Numbered registers
	numRegs := &CompletionGroup{
		Name:         tui.DIM + "num ([0-9])" + tui.RESET,
		DisplayType:  TabDisplayMap,
		MaxLength:    20,
		Descriptions: map[string]string{},
	}
	var nums []int
	for reg := range rl.registers.num {
		nums = append(nums, reg)
	}
	sort.Ints(nums)
	for _, val := range nums {
		buf := rl.registers.num[val]
		numRegs.Suggestions = append(numRegs.Suggestions, string(buf))
		numRegs.Descriptions[string(buf)] = fmt.Sprintf("%s\"%d%s", DIM, val, RESET)
	}

	if len(numRegs.Suggestions) > 0 {
		groups = append(groups, numRegs)
	}

	// Letter registers
	alphaRegs := &CompletionGroup{
		Name:         tui.DIM + "alpha ([a-z], [A-Z])" + tui.RESET,
		DisplayType:  TabDisplayMap,
		MaxLength:    20,
		Descriptions: map[string]string{},
	}
	var lett []string
	for reg := range rl.registers.alpha {
		lett = append(lett, reg)
	}
	sort.Strings(lett)
	for _, reg := range lett {
		buf := rl.registers.alpha[reg]
		alphaRegs.Suggestions = append(alphaRegs.Suggestions, string(buf))
		alphaRegs.Descriptions[string(buf)] = DIM + "\"" + reg + RESET
	}

	if len(alphaRegs.Suggestions) > 0 {
		groups = append(groups, alphaRegs)
	}

	return
}
