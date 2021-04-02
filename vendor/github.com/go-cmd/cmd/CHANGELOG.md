# go-cmd/cmd Changelog

## v1.3

### v1.3.0 (2020-10-31)

* Fixed last line not flushed if incomplete (PR #48) (@greut)
* Added ErrNotStarted
* Changed Stop() to return ErrNotStarted (issue #16)

## v1.2

### v1.2.1 (2020-07-11)

* Added `StartWithStdin(io.Reader)` (PR #50) (@juanenriqueescobar)

### v1.2.0 (2020-01-26)

* Changed streaming output: channels are closed after command stops (issue #26)
* Updated examples/blocking-streaming

## v1.1

### v1.1.0 (2019-11-24)

* Added support for Windows (PR #24) (@hfrappier, et al.)

## v1.0

### v1.0.6 (2019-09-30)

* Use go mod (PR #37) (@akkumar)

### v1.0.5 (2019-07-20)

* Fixed typo in README (PR #28) (@alokrajiv)
* Added `Cmd.Clone()` (PR #35) (@croqaz)
* Code cleanup (PR #34) (@croqaz)

### v1.0.4 (2018-11-22)

* Fixed no output: Buffered=false, Streaming=false
* Added `Cmd.Dir` to set `os/exec.Cmd.Dir` (PR #25) (@tkschmidt)

### v1.0.3 (2018-05-13)

* Added `Cmd.Env` to set `os/exec.Cmd.Env` (PR #14) (@robothor)

### v1.0.2 (2018-04-28)

* Changed `Running()` to `Done() <-chan struct{}` to match `Context.Done()` and support multiple waiters (PR #13)

### v1.0.1 (2018-04-22)

* Fixed errors in example code (PR #9) (@anshap1719)
* Added NewCmdOptions, Options, OutputBuffer, and OutputStream
* Added example code

### v1.0.0 (2017-03-22)

* First release.
