package buildcatalog

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Commands returns the read-only server build catalog commands.
func Commands(con *console.SliverClient) []*cobra.Command {
	return []*cobra.Command{
		catalogCommand("cores", clientpb.BuildSpecKind_BUILD_SPEC_CORE, con),
		catalogCommand("transforms", clientpb.BuildSpecKind_BUILD_SPEC_TRANSFORM, con),
	}
}

func catalogCommand(name string, kind clientpb.BuildSpecKind, con *console.SliverClient) *cobra.Command {
	root := &cobra.Command{
		Use:     name,
		Short:   "Inspect server build " + name,
		Args:    cobra.NoArgs,
		GroupID: consts.PayloadsHelpGroup,
		Run: func(cmd *cobra.Command, _ []string) {
			listSpecs(cmd, kind, con)
		},
	}
	flags.Bind(name, true, root, func(f *pflag.FlagSet) {
		f.String("target", "", "filter by target GOOS/GOARCH")
		f.String("availability", "all", "filter by all, available, or unavailable specs")
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List server build " + name,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			listSpecs(cmd, kind, con)
		},
	}
	infoCmd := &cobra.Command{
		Use:   "info <name>",
		Short: "Show a build spec and its supported capabilities",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			infoSpec(cmd, kind, args[0], con)
		},
	}
	infoCmd.Flags().Bool("yaml", false, "include the server's immutable YAML snapshot")
	root.AddCommand(listCmd, infoCmd)

	_ = root.RegisterFlagCompletionFunc("target", targetCompletion(kind, con))
	_ = root.RegisterFlagCompletionFunc("availability", func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return prefixValues([]string{"all", "available", "unavailable"}, toComplete), cobra.ShellCompDirectiveNoFileComp
	})
	for _, cmd := range []*cobra.Command{root, listCmd, infoCmd} {
		flags.BindFlagCompletions(cmd, func(completions *carapace.ActionMap) {
			(*completions)["target"] = carapace.ActionCallback(func(ctx carapace.Context) carapace.Action {
				values, _ := targetCompletion(kind, con)(cmd, nil, ctx.Value)
				return carapace.ActionValues(values...).Tag("server build targets")
			})
			(*completions)["availability"] = carapace.ActionValues("all", "available", "unavailable")
		})
	}
	carapace.Gen(infoCmd).PositionalCompletion(specNameAction(kind, con))
	return root
}

func queryFromFlags(cmd *cobra.Command, kind clientpb.BuildSpecKind) (*clientpb.BuildSpecQuery, error) {
	query := &clientpb.BuildSpecQuery{Kind: kind}
	target, _ := cmd.Flags().GetString("target")
	if target == "" {
		target, _ = cmd.InheritedFlags().GetString("target")
	}
	if target != "" {
		parts := strings.Split(target, "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("target must be GOOS/GOARCH (for example linux/amd64)")
		}
		query.TargetGOOS, query.TargetGOARCH = strings.ToLower(parts[0]), strings.ToLower(parts[1])
	}
	availability, _ := cmd.Flags().GetString("availability")
	if availability == "" {
		availability, _ = cmd.InheritedFlags().GetString("availability")
	}
	switch availability {
	case "", "all":
		query.Availability = clientpb.BuildSpecAvailability_BUILD_SPEC_ALL
	case "available":
		query.Availability = clientpb.BuildSpecAvailability_BUILD_SPEC_AVAILABLE
	case "unavailable":
		query.Availability = clientpb.BuildSpecAvailability_BUILD_SPEC_UNAVAILABLE
	default:
		return nil, fmt.Errorf("availability must be all, available, or unavailable")
	}
	return query, nil
}

func catalogContext(cmd *cobra.Command) (context.Context, context.CancelFunc) {
	timeout, _ := cmd.Flags().GetInt64("timeout")
	if timeout == 0 {
		timeout, _ = cmd.InheritedFlags().GetInt64("timeout")
	}
	if timeout <= 0 {
		timeout = flags.DefaultTimeout
	}
	return context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
}

func listSpecs(cmd *cobra.Command, kind clientpb.BuildSpecKind, con *console.SliverClient) {
	if con == nil || con.Rpc == nil {
		return
	}
	query, err := queryFromFlags(cmd, kind)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	ctx, cancel := catalogContext(cmd)
	defer cancel()
	result, err := con.Rpc.ListBuildSpecs(ctx, query)
	if err != nil {
		con.PrintErrorf("Failed to list build specs: %s\n", err)
		return
	}
	if len(result.Specs) == 0 {
		con.PrintInfof("No matching build specs\n")
		return
	}
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{"Name", "Version", "Digest", "Source", "Available", "Providers"})
	for _, spec := range result.Specs {
		providers := make([]string, 0, len(spec.Providers))
		for _, provider := range spec.Providers {
			state := "offline"
			if provider.Online {
				state = "online"
			}
			providers = append(providers, provider.Name+" ("+state+")")
		}
		slices.Sort(providers)
		sources := make([]string, 0, len(spec.Origins))
		for _, origin := range spec.Origins {
			sources = append(sources, origin.Source)
		}
		slices.Sort(sources)
		tw.AppendRow(table.Row{spec.Ref.Name, spec.Ref.Version, spec.Ref.Digest, strings.Join(sources, ", "), spec.Available, strings.Join(providers, ", ")})
	}
	con.Printf("%s\n", tw.Render())
}

func infoSpec(cmd *cobra.Command, kind clientpb.BuildSpecKind, name string, con *console.SliverClient) {
	if con == nil || con.Rpc == nil {
		return
	}
	query, err := queryFromFlags(cmd, kind)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	query.Name = name
	ctx, cancel := catalogContext(cmd)
	defer cancel()
	result, err := con.Rpc.ListBuildSpecs(ctx, query)
	if err != nil {
		con.PrintErrorf("Failed to inspect build spec: %s\n", err)
		return
	}
	if len(result.Specs) == 0 {
		con.PrintErrorf("No matching build spec named %q\n", name)
		return
	}
	showYAML, _ := cmd.Flags().GetBool("yaml")
	for _, listed := range result.Specs {
		spec, err := con.Rpc.GetBuildSpec(ctx, &clientpb.BuildSpecRequest{Kind: kind, Ref: listed.Ref})
		if err != nil {
			con.PrintErrorf("Failed to load exact build spec: %s\n", err)
			return
		}
		con.Printf("%s@%s\nDigest: %s\nExecutor: %s\nAvailable: %t\n", spec.Ref.Name, spec.Ref.Version, spec.Ref.Digest, spec.ExecutorID, spec.Available)
		if spec.RuntimeProtocol != "" {
			con.Printf("Runtime: %s (%s)\n", spec.RuntimeProtocol, spec.CommandSource)
		}
		for _, origin := range spec.Origins {
			con.Printf("Source: %s", origin.Source)
			if origin.Provider != "" {
				con.Printf(" (%s)", origin.Provider)
			}
			con.Printf("\n")
		}
		for _, provider := range spec.Providers {
			state := "offline"
			if provider.Online {
				state = "online"
			}
			con.Printf("Provider: %s (%s, last seen %s)\n", provider.Name, state, formatUnix(provider.LastSeenUnix))
		}
		tw := table.NewWriter()
		tw.SetStyle(settings.GetTableStyle(con))
		tw.AppendHeader(table.Row{"Targets", "Mode", "Protocols", "Input", "Output", "Formats", "Toolchain"})
		for _, capability := range spec.Capabilities {
			if query.TargetGOOS != "" && !capabilitySupportsTarget(capability, query.TargetGOOS, query.TargetGOARCH) {
				continue
			}
			targets := make([]string, 0, len(capability.Targets))
			for _, target := range capability.Targets {
				if query.TargetGOOS != "" && (target.GOOS != query.TargetGOOS || target.GOARCH != query.TargetGOARCH) {
					continue
				}
				targets = append(targets, target.GOOS+"/"+target.GOARCH)
			}
			toolchain := capability.ToolchainDriver
			if len(capability.ToolchainRequirements) > 0 {
				toolchain += " (" + strings.Join(capability.ToolchainRequirements, ", ") + ")"
			}
			tw.AppendRow(table.Row{
				strings.Join(targets, ", "), capability.Mode, strings.Join(capability.Protocols, ", "),
				artifactString(capability.Input), artifactString(capability.Output), strings.Join(capability.OperatorFormats, ", "), toolchain,
			})
		}
		con.Printf("%s\n", tw.Render())
		if showYAML {
			con.Printf("%s\n", string(spec.YAML))
		}
	}
}

func capabilitySupportsTarget(capability *clientpb.BuildSpecCapability, goos, goarch string) bool {
	for _, target := range capability.Targets {
		if target.GOOS == goos && target.GOARCH == goarch {
			return true
		}
	}
	return false
}

func artifactString(artifact *clientpb.BuildArtifactType) string {
	if artifact == nil {
		return "—"
	}
	result := artifact.Kind + "/" + artifact.Container
	if artifact.GoBuildMode != "" {
		result += " [" + artifact.GoBuildMode + "]"
	}
	return result
}

func formatUnix(value int64) string {
	if value <= 0 {
		return "unknown"
	}
	return time.Unix(value, 0).UTC().Format(time.RFC3339)
}

func prefixValues(values []string, prefix string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			result = append(result, value)
		}
	}
	return result
}

func targetCompletion(kind clientpb.BuildSpecKind, con *console.SliverClient) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if con == nil || con.Rpc == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(flags.DefaultTimeout)*time.Second)
		defer cancel()
		result, err := con.Rpc.ListBuildSpecs(ctx, &clientpb.BuildSpecQuery{Kind: kind})
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		values := map[string]struct{}{}
		for _, spec := range result.Specs {
			for _, capability := range spec.Capabilities {
				for _, target := range capability.Targets {
					values[target.GOOS+"/"+target.GOARCH] = struct{}{}
				}
			}
		}
		ordered := make([]string, 0, len(values))
		for target := range values {
			ordered = append(ordered, target)
		}
		slices.Sort(ordered)
		return prefixValues(ordered, toComplete), cobra.ShellCompDirectiveNoFileComp
	}
}

func specNameAction(kind clientpb.BuildSpecKind, con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(_ carapace.Context) carapace.Action {
		if con == nil || con.Rpc == nil {
			return carapace.ActionMessage("Not connected to a server")
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(flags.DefaultTimeout)*time.Second)
		defer cancel()
		result, err := con.Rpc.ListBuildSpecs(ctx, &clientpb.BuildSpecQuery{Kind: kind})
		if err != nil {
			return carapace.ActionMessage("Catalog unavailable: %s", err.Error())
		}
		names := make([]string, 0, len(result.Specs))
		seen := map[string]struct{}{}
		for _, spec := range result.Specs {
			if _, exists := seen[spec.Ref.Name]; exists {
				continue
			}
			seen[spec.Ref.Name] = struct{}{}
			names = append(names, spec.Ref.Name)
		}
		slices.Sort(names)
		return carapace.ActionValues(names...).Tag("build spec names")
	})
}
