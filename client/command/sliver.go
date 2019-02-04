package command

import (
	"bytes"
	"fmt"
	consts "sliver/client/constants"
	pb "sliver/protobuf/client"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

func sessions(ctx *grumble.Context, rpc RPCServer) {
	resp := rpc(&pb.Envelope{
		Type: consts.SessionsStr,
		Data: []byte{},
	}, defaultTimeout)
	if resp == nil {
		fmt.Printf(Warn + "Command timeout\n")
		return
	}
	sessions := &pb.Sessions{}
	proto.Unmarshal(resp.Data, sessions)

	slivers := map[int32]*pb.Sliver{}
	for _, sliver := range sessions.Slivers {
		slivers[sliver.ID] = sliver
	}
	if 0 < len(slivers) {
		printSlivers(slivers)
	} else {
		fmt.Printf(Info + "No slivers connected\n")
	}
}

/*
	So this method is a little more complex than you'd maybe think,
	this is because Go's tabwriter aligns columns by counting bytes
	and since we want to modify the color of the active sliver row
	the number of bytes per row won't line up. So we render the table
	into a buffer and note which row the active sliver is in. Then we
	write each line to the term and insert the ANSI codes just before
	we display the row.
*/
func printSlivers(sessions map[int32]*pb.Sliver) {
	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	// Column Headers
	fmt.Fprintln(table, "\nID\tName\tTransport\tRemote Address\tUsername\tOperating System\t")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("ID")),
		strings.Repeat("=", len("Name")),
		strings.Repeat("=", len("Transport")),
		strings.Repeat("=", len("Remote Address")),
		strings.Repeat("=", len("Username")),
		strings.Repeat("=", len("Operating System")))

	// Sort the keys becuase maps have a randomized order
	var keys []int
	for _, sliver := range sessions {
		keys = append(keys, int(sliver.ID))
	}
	sort.Ints(keys) // Fucking Go can't sort int32's, so we convert to/from int's

	activeIndex := -1
	for index, key := range keys {
		sliver := sessions[int32(key)]
		if ActiveSliver.Sliver != nil && ActiveSliver.Sliver.ID == sliver.ID {
			activeIndex = index + 3 // Two lines for the headers
		}
		fmt.Fprintf(table, "%d\t%s\t%s\t%s\t%s\t%s\t\n",
			sliver.ID, sliver.Name, sliver.Transport, sliver.RemoteAddress, sliver.Username,
			fmt.Sprintf("%s/%s", sliver.OS, sliver.Arch))
	}
	table.Flush()

	if activeIndex != -1 {
		lines := strings.Split(outputBuf.String(), "\n")
		for lineNumber, line := range lines {
			if lineNumber == activeIndex {
				fmt.Printf("%s%s%s\n", green, line, normal)
			} else {
				fmt.Printf("%s\n", line)
			}
		}
	} else {
		fmt.Printf(outputBuf.String())
	}
}

func use(ctx *grumble.Context, rpc RPCServer) {
	if len(ctx.Args) == 0 {
		fmt.Printf(Warn + "Missing sliver name or session number, see `help use`\n")
		return
	}
	resp := rpc(&pb.Envelope{
		Type: consts.SessionsStr,
		Data: []byte{},
	}, defaultTimeout)
	sessions := &pb.Sessions{}
	proto.Unmarshal(resp.Data, sessions)

	for _, sliver := range sessions.Slivers {
		if string(sliver.ID) == ctx.Args[0] || sliver.Name == ctx.Args[0] {
			ActiveSliver.SetActiveSliver(sliver)
			break
		}
	}
	fmt.Printf(Warn+"Invalid sliver name or session number '%s'\n", ctx.Args[0])
}

func background(ctx *grumble.Context, rpc RPCServer) {
	ActiveSliver.SetActiveSliver(nil)
}

func kill(ctx *grumble.Context, rpc RPCServer) {

}

func info(ctx *grumble.Context, rpc RPCServer) {

}

func generate(ctx *grumble.Context, rpc RPCServer) {

}

func ping(ctx *grumble.Context, rpc RPCServer) {

}

func getPID(ctx *grumble.Context, rpc RPCServer) {

}

func getUID(ctx *grumble.Context, rpc RPCServer) {

}

func getGID(ctx *grumble.Context, rpc RPCServer) {

}

func whoami(ctx *grumble.Context, rpc RPCServer) {

}

func ps(ctx *grumble.Context, rpc RPCServer) {

}

func procdump(ctx *grumble.Context, rpc RPCServer) {

}
