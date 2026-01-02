package mcp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	clientmcp "github.com/bishopfox/sliver/client/mcp"
	"github.com/bishopfox/sliver/client/version"
	mcpclient "github.com/mark3labs/mcp-go/client"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
	"github.com/spf13/cobra"
)

// McpConsoleCmd connects to an MCP server and starts a minimal console.
func McpConsoleCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	rawTransport, _ := cmd.Flags().GetString("transport")
	transport, err := clientmcp.ParseTransport(rawTransport)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	targetURL, _ := cmd.Flags().GetString("url")
	if targetURL == "" {
		cfg := clientmcp.GetStatus().Config.WithDefaults()
		cfg.Transport = transport
		targetURL, err = cfg.EndpointURL()
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}

	client, err := newConsoleClient(transport, targetURL)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	defer client.Close()

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	_, err = client.Initialize(ctx, mcpapi.InitializeRequest{
		Params: mcpapi.InitializeParams{
			ProtocolVersion: mcpapi.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcpapi.Implementation{
				Name:    "sliver-mcp-console",
				Version: version.Version,
			},
			Capabilities: mcpapi.ClientCapabilities{},
		},
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	con.PrintInfof("Connected to MCP server (%s)\n", targetURL)
	runConsoleREPL(ctx, client, cmd.InOrStdin(), cmd.OutOrStdout())
}

func newConsoleClient(transport clientmcp.Transport, targetURL string) (*mcpclient.Client, error) {
	switch transport {
	case clientmcp.TransportHTTP:
		return mcpclient.NewStreamableHttpClient(targetURL)
	case clientmcp.TransportSSE:
		return mcpclient.NewSSEMCPClient(targetURL)
	default:
		return nil, fmt.Errorf("unsupported transport %q", transport)
	}
}

func runConsoleREPL(ctx context.Context, client *mcpclient.Client, in io.Reader, out io.Writer) {
	reader := bufio.NewReader(in)
	for {
		fmt.Fprint(out, "mcp> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Fprintln(out)
				return
			}
			fmt.Fprintf(out, "read error: %s\n", err)
			return
		}
		line = strings.TrimSpace(line)
		switch line {
		case "":
			continue
		case "exit", "quit":
			return
		case "help", "?":
			printConsoleHelp(out)
		case "tools":
			printConsoleTools(ctx, client, out)
		default:
			fmt.Fprintf(out, "unknown command: %s\n", line)
		}
	}
}

func printConsoleHelp(out io.Writer) {
	fmt.Fprintln(out, "commands: tools, help, exit")
}

func printConsoleTools(ctx context.Context, client *mcpclient.Client, out io.Writer) {
	tools, err := client.ListTools(ctx, mcpapi.ListToolsRequest{})
	if err != nil {
		fmt.Fprintf(out, "tools error: %s\n", err)
		return
	}
	if len(tools.Tools) == 0 {
		fmt.Fprintln(out, "no tools available")
		return
	}
	for _, tool := range tools.Tools {
		if tool.Description != "" {
			fmt.Fprintf(out, "- %s: %s\n", tool.Name, tool.Description)
			continue
		}
		fmt.Fprintf(out, "- %s\n", tool.Name)
	}
}
