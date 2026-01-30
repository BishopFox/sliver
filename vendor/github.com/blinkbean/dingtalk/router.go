package dingtalk

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

var cmdTable = make(map[string]*command)

type command struct {
	executor ExecFunc
	isAdmin  bool // 是否需要管理员权限
	arity    int  // 参数个数
}

func RegisterCommand(name string, execFunc ExecFunc, arity int, isAdmin bool) {
	cmdTable[name] = &command{
		executor: execFunc,
		arity:    arity,
		isAdmin:  isAdmin,
	}
}

func validateArity(arity int, cmdArgs []string) bool {
	argNum := len(cmdArgs)
	if arity >= 0 {
		return argNum == arity
	}
	return argNum >= -arity
}

func execDingCommand(msg outGoingModel) []byte {
	content := msg.Text.Content
	keyAndArgs := strings.Split(strings.TrimSpace(content), " ")
	cmdName := strings.ToLower(keyAndArgs[0])
	cmd, ok := cmdTable[cmdName]
	if !ok {
		return NewTextMsg("ERR: unregistered command '" + cmdName + "'").Marshaler()
	}
	if !validateArity(cmd.arity, keyAndArgs) {
		return NewTextMsg("ERR: wrong number of arguments for '" + cmdName + "' command").Marshaler()
	}
	if cmd.isAdmin && msg.IsAdmin != cmd.isAdmin {
		return NewTextMsg("ERR '" + cmdName + "': you have no right to do this operation").Marshaler()
	}
	return cmd.executor(keyAndArgs[1:])
}

// MyHandler实现Handler接口

type OutGoingHandler struct{}

func (h *OutGoingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body := r.Body
	var err error
	var buf []byte
	obj := outGoingModel{}
	buf, err = io.ReadAll(body)
	if err != nil {
		return
	}
	err = json.Unmarshal(buf, &obj)
	if err != nil {
		return
	}
	msg := execDingCommand(obj)
	if msg == nil {
		return
	}
	_, _ = w.Write(msg)
}
