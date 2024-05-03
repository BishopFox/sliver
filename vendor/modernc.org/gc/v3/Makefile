.PHONY:	all clean edit editor test test2 back report report2 parser2 benchmarks benchmarks2 mem memgo race nreport build_all_targets

all:

build_all_targets:
	GOOS=darwin GOARCH=amd64 go test -c -o /dev/null
	GOOS=darwin GOARCH=arm64 go test -c -o /dev/null
	GOOS=freebsd GOARCH=386 go test -c -o /dev/null
	GOOS=freebsd GOARCH=amd64 go test -c -o /dev/null
	GOOS=freebsd GOARCH=arm go test -c -o /dev/null
	GOOS=freebsd GOARCH=arm64 go test -c -o /dev/null
	GOOS=illumos GOARCH=amd64 go test -c -o /dev/null
	GOOS=linux GOARCH=386 go test -c -o /dev/null
	GOOS=linux GOARCH=amd64 go test -c -o /dev/null
	GOOS=linux GOARCH=arm go test -c -o /dev/null
	GOOS=linux GOARCH=arm64 go test -c -o /dev/null
	GOOS=linux GOARCH=ppc64le go test -c -o /dev/null
	GOOS=linux GOARCH=riscv64 go test -c -o /dev/null
	GOOS=linux GOARCH=s390x go test -c -o /dev/null
	GOOS=netbsd GOARCH=386 go test -c -o /dev/null
	GOOS=netbsd GOARCH=amd64 go test -c -o /dev/null
	GOOS=netbsd GOARCH=arm go test -c -o /dev/null
	GOOS=openbsd GOARCH=386 go test -c -o /dev/null
	GOOS=openbsd GOARCH=amd64 go test -c -o /dev/null
	GOOS=openbsd GOARCH=arm64 go test -c -o /dev/null
	GOOS=windows GOARCH=386 go test -c -o /dev/null
	GOOS=windows GOARCH=amd64 go test -c -o /dev/null
	GOOS=windows GOARCH=arm64 go test -c -o /dev/null

clean:
	rm -f cpu.test mem.test *.out
	go clean

edit:
	@touch log
	@if [ -f "Session.vim" ]; then gvim -S & else gvim -p Makefile all_test.go gc.go & fi

editor:
	gofmt -l -s -w *.go
	go test -c -o /dev/null 2>&1 | tee log-editor

race:
	go test -v -failfast -heap -race 2>&1 | tee log-test
	@git diff testdata/ || true
	@git status
	@grep TOTAL log-test || true
	grep 'FAIL\|TODO' log-test || true

test:
	go test -v -failfast -trctodo -exterr -heap 2>&1 | tee log-test
	@git diff testdata/ || true
	@git status
	@grep TOTAL log-test || true
	grep 'FAIL\|TODO' log-test || true

test2:
	go test -v -failfast -trctodo -exterr -src $$HOME/src 2>&1 | tee log-test2
	@git diff testdata/ || true
	@git status
	@grep TOTAL log-test2 || true
	grep 'FAIL\|TODO' log-test2 || true

parser2:
	go test -v -failfast -run TestParser -src $$HOME/src 2>&1 | tee log-parser2
	@git diff testdata/ || true
	@git status
	@grep TOTAL log-parser2 || true
	grep 'FAIL\|TODO' log-parser2 || true

back:
	go test -v -failfast -noback 2>&1 | tee log-back
	@git diff testdata/ || true
	@git status
	@grep TOTAL log-back || true
	grep 'FAIL\|TODO' log-back || true

nreport:
	touch log-nreport
	cp log-nreport log-nreport0
	go test -v -failfast -run TestParser -heap -nreport 2>&1 | tee log-nreport
	@git diff testdata/ || true
	@git status
	@grep TOTAL log-report || true
	grep 'FAIL\|TODO' log-report || true

report:
	go test -v -failfast -run TestParser -report 2>&1 | tee log-report
	@git diff testdata/ || true
	@git status
	@grep TOTAL log-report || true
	grep 'FAIL\|TODO' log-report || true

report2:
	go test -v -failfast -run TestParser -src $$HOME/src -report 2>&1 | tee log-report2
	@git diff testdata/ || true
	@git status
	@grep TOTAL log-report2 || true
	grep 'FAIL\|TODO' log-report2 || true


benchmarks:
	go test -v -run @ -bench . 2>&1 | tee log-benchmarks

benchmarks2:
	go test -v -run @ -bench . -bsrc $$HOME/src 2>&1 | tee log-benchmarks2

mem:
	go test -run @ -bench BenchmarkParser -memprofile mem.out
	go tool pprof --lines --alloc_space *.test mem.out

memgo:
	go test -run @ -bench BenchmarkGoParser -memprofile mem.out
	go tool pprof --lines --alloc_space *.test mem.out
