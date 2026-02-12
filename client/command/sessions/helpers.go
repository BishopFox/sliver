package sessions

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox
	Copyright (C) 2021 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
*/

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

var (
	// ErrNoSessions - No sessions available.
	// ErrNoSessions - No 会话 available.
	ErrNoSessions = errors.New("no sessions")
	// ErrNoSelection - No selection made.
	// ErrNoSelection - No 选择 made.
	ErrNoSelection = errors.New("no selection")
)

// SelectSession - Interactive menu for the user to select an session, optionally only display live sessions.
// SelectSession - Interactive 菜单供用户选择 session，可选择仅显示实时 sessions.
func SelectSession(onlyAlive bool, con *console.SliverClient) (*clientpb.Session, error) {
	sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil, err
	}
	if len(sessions.Sessions) == 0 {
		return nil, ErrNoSessions
	}

	sessionsMap := map[string]*clientpb.Session{}
	for _, session := range sessions.GetSessions() {
		sessionsMap[session.ID] = session
	}

	// Sort the keys because maps have a randomized order, these keys must be ordered for the selection
	// Sort 键，因为地图具有随机顺序，这些键必须按顺序进行选择
	// to work properly since we rely on the index of the user's selection to find the session in the map
	// 正常工作，因为我们依赖用户选择的索引来查找地图中的 session
	var keys []string
	for _, session := range sessions.Sessions {
		keys = append(keys, session.ID)
	}
	sort.Strings(keys)

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	// Column Headers
	for _, key := range keys {
		session := sessionsMap[key]
		if onlyAlive && session.IsDead {
			continue
		}
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\n",
			session.ID,
			session.Name,
			session.RemoteAddress,
			session.Hostname,
			session.Username,
			fmt.Sprintf("%s/%s", session.OS, session.Arch),
		)
	}
	table.Flush()

	options := strings.Split(outputBuf.String(), "\n")
	options = options[:len(options)-1] // Remove the last empty option
	options = options[:len(options)-1] // Remove 最后一个空选项
	selected := ""
	_ = forms.Select("Select a session:", options, &selected)
	if selected == "" {
		return nil, ErrNoSelection
	}

	// Go from the selected option -> index -> session
	// 所选选项中的 Go -> 索引 -> session
	for index, option := range options {
		if option == selected {
			return sessionsMap[keys[index]], nil
		}
	}
	return nil, ErrNoSelection
}
