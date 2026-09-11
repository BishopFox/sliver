package crack

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

type crackJobCompletionEntry struct {
	id          string
	description string
}

// CrackJobIDCompleter completes durable crack job IDs.
func CrackJobIDCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(_ carapace.Context) carapace.Action {
		entries, err := crackJobCompletionEntries(con)
		if err != nil {
			return carapace.ActionMessage("failed to fetch crack jobs: %s", err.Error())
		}

		values := make([]string, 0, len(entries)*2)
		for _, entry := range entries {
			values = append(values, entry.id, entry.description)
		}
		return carapace.ActionValuesDescribed(values...).Tag("crack jobs")
	})
}

func registerCrackJobIDCompletion(cmd *cobra.Command, con *console.SliverClient) {
	if cmd == nil || cmd.ValidArgsFunction != nil {
		return
	}

	cmd.ValidArgsFunction = func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		entries, err := crackJobCompletionEntries(con)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		values := make([]string, 0, len(entries))
		for _, entry := range entries {
			if strings.HasPrefix(entry.id, toComplete) {
				values = append(values, fmt.Sprintf("%s\t%s", entry.id, entry.description))
			}
		}
		return values, cobra.ShellCompDirectiveNoFileComp
	}
}

func crackJobCompletionEntries(con *console.SliverClient) ([]crackJobCompletionEntry, error) {
	if con == nil || con.Rpc == nil {
		return nil, nil
	}
	jobs, err := con.Rpc.CrackJobs(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil, err
	}
	if jobs == nil {
		return nil, nil
	}

	entries := make([]crackJobCompletionEntry, 0, len(jobs.GetJobs()))
	for _, job := range jobs.GetJobs() {
		if job == nil || strings.TrimSpace(job.GetID()) == "" {
			continue
		}
		entries = append(entries, crackJobCompletionEntry{
			id: job.GetID(),
			description: fmt.Sprintf(
				"%s, created %s, keyspace %s, %d task(s), %d result(s)",
				safeCrackCell(job.GetStatus().String()),
				formatCrackTime(job.GetCreatedAt()),
				valueOrDash(job.GetKeyspace()),
				len(job.GetTasks()),
				job.GetResultCount(),
			),
		})
	}
	return entries, nil
}

func CrackHcstat2Completer(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		hcstat2, err := con.Rpc.CrackFilesList(context.Background(), &clientpb.CrackFile{Type: clientpb.CrackFileType_MARKOV_HCSTAT2})
		if err != nil {
			return carapace.ActionMessage("failed to fetch crack files: %s", err.Error())
		}

		results := make([]string, 0)

		for _, file := range hcstat2.Files {
			if file.Type != clientpb.CrackFileType_MARKOV_HCSTAT2 {
				continue
			}

			results = append(results, file.Name)
			desc := fmt.Sprintf("%s (%d chunks, chunk size: %s)", byteCountSI(file.UncompressedSize), len(file.Chunks), byteCountSI(file.ChunkSize))
			results = append(results, desc)
		}

		return carapace.ActionValuesDescribed(results...).Tag("hcstat2 files")
	})
}

func CrackWordlistCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		wordlists, err := con.Rpc.CrackFilesList(context.Background(), &clientpb.CrackFile{Type: clientpb.CrackFileType_WORDLIST})
		if err != nil {
			return carapace.ActionMessage("failed to fetch crack files: %s", err.Error())
		}

		results := make([]string, 0)

		for _, file := range wordlists.Files {
			if file.Type != clientpb.CrackFileType_WORDLIST {
				continue
			}

			results = append(results, file.Name)
			createdAt, modifiedAt := time.Unix(file.CreatedAt, 0).Format("03:04:05"), time.Unix(file.LastModified, 0).Format("03:04:05")
			desc := fmt.Sprintf("%s (Created: %s, modified: %s)", byteCountSI(file.UncompressedSize), createdAt, modifiedAt)
			results = append(results, desc)
		}

		return carapace.ActionValuesDescribed(results...).Tag("wordlists")
	})
}

func CrackRulesCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		rules, err := con.Rpc.CrackFilesList(context.Background(), &clientpb.CrackFile{Type: clientpb.CrackFileType_RULES})
		if err != nil {
			return carapace.ActionMessage("failed to fetch crack files: %s", err.Error())
		}

		results := make([]string, 0)

		for _, file := range rules.Files {
			if file.Type != clientpb.CrackFileType_RULES {
				continue
			}

			results = append(results, file.Name)
			createdAt, modifiedAt := time.Unix(file.CreatedAt, 0).Format("03:04:05"), time.Unix(file.LastModified, 0).Format("03:04:05")
			desc := fmt.Sprintf("%s (Created: %s, modified: %s)", byteCountSI(file.UncompressedSize), createdAt, modifiedAt)
			results = append(results, desc)
		}

		return carapace.ActionValuesDescribed(results...).Tag("rules")
	})
}

func byteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

func byteCountIEC(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(b)/float64(div), "KMGTPE"[exp])
}
