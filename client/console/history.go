package console

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

// "github.com/maxlandon/wiregost/client/connection"
// clientpb "github.com/maxlandon/wiregost/proto/v1/gen/go/client"

var (
	// ClientHist - Client console history
	ClientHist = &ClientHistory{LinesSinceStart: 1}
	// UserHist - User history
	UserHist = &UserHistory{LinesSinceStart: 1}
)

// This file manages all command history flux for this console. The user can request
// 2 different lists of commands: the history for this console only (identified by its
// unique ID) with Ctrl-r, or the history for all the user's consoles, with Ctrl-R.

// ClientHistory - Writes and queries only the Client's history
type ClientHistory struct {
	LinesSinceStart int // Keeps count of line since session
}

// Write - Sends the last command to the server for saving
func (h *ClientHistory) Write(s string) (int, error) {

	// res, err := connection.ConnectionRPC.AddToHistory(context.Background(),
	//         &clientpb.AddCmdHistoryRequest{Line: s, Client: cctx.Client})
	// if err != nil {
	//         return 0, err
	// }

	// if !res.Doublon {
	//         h.LinesSinceStart++
	// }
	return h.LinesSinceStart, nil
}

// GetLine returns a line from history
func (h *ClientHistory) GetLine(i int) (string, error) {

	// res, err := connection.ConnectionRPC.GetHistory(context.Background(),
	//         &clientpb.HistoryRequest{
	//                 AllConsoles: false,
	//                 Index:       int32(i),
	//                 Client:      cctx.Client,
	//         })
	// if err != nil {
	//         return "", err
	// }
	// h.LinesSinceStart = int(res.HistLength)
	//
	// return res.Line, nil
	return "", nil
}

// Len returns the number of lines in history
func (h *ClientHistory) Len() int {
	return h.LinesSinceStart
}

// Dump returns the entire history
func (h *ClientHistory) Dump() interface{} {
	return nil
}

// UserHistory - Only in charge of queries for the User's history
type UserHistory struct {
	LinesSinceStart int // Keeps count of line since session
}

func (h *UserHistory) Write(s string) (int, error) {
	h.LinesSinceStart++
	return h.LinesSinceStart, nil
}

// GetLine returns a line from history
func (h *UserHistory) GetLine(i int) (string, error) {

	// res, err := connection.ConnectionRPC.GetHistory(context.Background(),
	//         &clientpb.HistoryRequest{
	//                 AllConsoles: true,
	//                 Index:       int32(i),
	//                 Client:      cctx.Client,
	//         })
	// if err != nil {
	//         return "", err
	// }
	// h.LinesSinceStart = int(res.HistLength)
	//
	// return res.Line, nil
	return "", nil
}

// Len returns the number of lines in history
func (h *UserHistory) Len() int {
	return h.LinesSinceStart
}

// Dump returns the entire history
func (h *UserHistory) Dump() interface{} {
	return nil
}
