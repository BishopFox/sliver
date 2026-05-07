package certificates

import (
	"context"
	"sort"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

// CertificateCommonNameCompleter completes common names for stored certificates.
func CertificateCommonNameCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		if con == nil || con.Rpc == nil {
			return carapace.ActionMessage("not connected")
		}

		values, err := certificateCommonNameValues(con)
		if err != nil {
			return carapace.ActionMessage("failed to list certificates %s", err)
		}
		if len(values) == 0 {
			return carapace.ActionMessage("no certificates")
		}

		return carapace.ActionValues(values...).Tag("common name").Usage("certificate common name")
	})
}

func registerCertificateCNFlagCompletion(cmd *cobra.Command, name string, con *console.SliverClient) {
	if cmd == nil {
		return
	}
	if _, ok := cmd.GetFlagCompletionFunc(name); ok {
		return
	}
	if cmd.Flags().Lookup(name) == nil && cmd.PersistentFlags().Lookup(name) == nil {
		return
	}
	_ = cmd.RegisterFlagCompletionFunc(name, func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		values, err := certificateCommonNameValues(con)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return filterCompletionValues(values, toComplete), cobra.ShellCompDirectiveNoFileComp
	})
}

func certificateCommonNameValues(con *console.SliverClient) ([]string, error) {
	if con == nil || con.Rpc == nil {
		return nil, nil
	}
	certInfo, err := con.Rpc.GetCertificateInfo(context.Background(), &clientpb.CertificatesReq{
		CategoryFilters: AllTransports | AllRoles,
	})
	if err != nil {
		return nil, err
	}

	unique := map[string]struct{}{}
	for _, cert := range certInfo.Info {
		if cert.CN == "" {
			continue
		}
		unique[cert.CN] = struct{}{}
	}

	values := make([]string, 0, len(unique))
	for name := range unique {
		values = append(values, name)
	}
	sort.Strings(values)
	return values, nil
}

func filterCompletionValues(values []string, prefix string) []string {
	if prefix == "" {
		return values
	}

	filtered := make([]string, 0, len(values))
	for _, value := range values {
		candidate := value
		if tab := strings.IndexByte(value, '\t'); tab >= 0 {
			candidate = value[:tab]
		}
		if strings.HasPrefix(candidate, prefix) {
			filtered = append(filtered, value)
		}
	}

	return filtered
}
