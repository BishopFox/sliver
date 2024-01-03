package crack

import (
	"context"
	"fmt"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/rsteube/carapace"
)

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
			desc := fmt.Sprintf("%s (%s chunks, chunk size: %s)", byteCountSI(file.UncompressedSize), len(file.Chunks), byteCountSI(file.ChunkSize))
			results = append(results, desc)
		}

		return carapace.ActionValuesDescribed(results...).Tag("hcstat2 files")
	})
}

func CrackWordlistCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		hcstat2, err := con.Rpc.CrackFilesList(context.Background(), &clientpb.CrackFile{Type: clientpb.CrackFileType_MARKOV_HCSTAT2})
		if err != nil {
			return carapace.ActionMessage("failed to fetch crack files: %s", err.Error())
		}

		results := make([]string, 0)

		for _, file := range hcstat2.Files {
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
		hcstat2, err := con.Rpc.CrackFilesList(context.Background(), &clientpb.CrackFile{Type: clientpb.CrackFileType_MARKOV_HCSTAT2})
		if err != nil {
			return carapace.ActionMessage("failed to fetch crack files: %s", err.Error())
		}

		results := make([]string, 0)

		for _, file := range hcstat2.Files {
			if file.Type != clientpb.CrackFileType_RULES {
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
