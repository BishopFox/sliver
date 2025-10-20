package console

// AddInterrupt registers a handler to run when the console receives a given
// interrupt error from the underlying readline shell. Mainly two interrupt
// signals are concerned: io.EOF (returned when pressing CtrlD), and console.ErrCtrlC.
// Many will want to use this to switch menus. Note that these interrupt errors only
// work when the console is NOT currently executing a command, only when reading input.
func (m *Menu) AddInterrupt(err error, handler func(c *Console)) {
	m.mutex.RLock()
	m.interruptHandlers[err] = handler
	m.mutex.RUnlock()
}

// DelInterrupt removes one or more interrupt handlers from the menu registered ones.
// If no error is passed as argument, all handlers are removed.
func (m *Menu) DelInterrupt(errs ...error) {
	m.mutex.RLock()
	if len(errs) == 0 {
		m.interruptHandlers = make(map[error]func(c *Console))
	} else {
		for _, err := range errs {
			delete(m.interruptHandlers, err)
		}
	}
	m.mutex.RUnlock()
}

func (m *Menu) handleInterrupt(err error) {
	m.console.mutex.RLock()
	m.console.isExecuting = true
	m.console.mutex.RUnlock()

	defer func() {
		m.console.mutex.RLock()
		m.console.isExecuting = false
		m.console.mutex.RUnlock()
	}()

	// TODO: this is not a very, very safe way of comparing
	// errors. I'm not sure what to right now with this, but
	// from my (unreliable) expectations and usage, I see and
	// use things like errors.New(os.Interrupt.String()), so
	// the string itself is likely to change in the future.
	//
	// But if people use their own third-party errors... nothing is guaranteed.
	for herr, handler := range m.interruptHandlers {
		if err.Error() == herr.Error() {
			handler(m.console)
		}
	}
}
