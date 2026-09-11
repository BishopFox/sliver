package crack

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/util"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/klauspost/compress/zstd"
	"github.com/spf13/cobra"
)

func localCrackFileInfo(path string) (os.FileInfo, error) {
	return localCrackFileInfoWith(path, os.Stat)
}

func localCrackFileInfoWith(path string, stat func(string) (os.FileInfo, error)) (os.FileInfo, error) {
	info, err := stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat local crack file %q: %w", path, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("local crack file %q is a directory", path)
	}
	return info, nil
}

// CrackWordlistsCmd - Manage GPU cracking stations.
func CrackWordlistsCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	wordlists, err := con.Rpc.CrackFilesList(context.Background(), &clientpb.CrackFile{Type: clientpb.CrackFileType_WORDLIST})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if len(wordlists.Files) == 0 {
		con.PrintInfof("No wordlists, add some using `crack wordlists add`\n")
	} else {
		PrintCrackFiles(wordlists, con)
		con.Println()
		con.Printf("Disk quota %02.2f%% - %s of %s\n",
			(float64(wordlists.CurrentDiskUsage)/float64(wordlists.MaxDiskUsage))*100,
			util.ByteCountBinary(wordlists.CurrentDiskUsage),
			util.ByteCountBinary(wordlists.MaxDiskUsage),
		)
	}
}

// CrackRulesCmd - Manage GPU cracking stations.
func CrackRulesCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	rules, err := con.Rpc.CrackFilesList(context.Background(), &clientpb.CrackFile{Type: clientpb.CrackFileType_RULES})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if len(rules.Files) == 0 {
		con.PrintInfof("No rules files, add some using `crack rules add`\n")
	} else {
		PrintCrackFiles(rules, con)
		con.Println()
		con.Printf("Disk quota %02.2f%% - %s of %s\n",
			(float64(rules.CurrentDiskUsage)/float64(rules.MaxDiskUsage))*100,
			util.ByteCountBinary(rules.CurrentDiskUsage),
			util.ByteCountBinary(rules.MaxDiskUsage),
		)
	}
}

// CrackHcstat2Cmd - Manage GPU cracking stations.
func CrackHcstat2Cmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	hcstat2, err := con.Rpc.CrackFilesList(context.Background(), &clientpb.CrackFile{Type: clientpb.CrackFileType_MARKOV_HCSTAT2})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if len(hcstat2.Files) == 0 {
		con.PrintInfof("No hcstat2 files, add some using `crack hcstat2 add`\n")
	} else {
		PrintCrackFiles(hcstat2, con)
		con.Println()
		con.Printf("Disk quota %02.2f%% - %s of %s\n",
			(float64(hcstat2.CurrentDiskUsage)/float64(hcstat2.MaxDiskUsage))*100,
			util.ByteCountBinary(hcstat2.CurrentDiskUsage),
			util.ByteCountBinary(hcstat2.MaxDiskUsage),
		)
	}
}

func PrintCrackFiles(crackFiles *clientpb.CrackFiles, con *console.SliverClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{"Name", "Size"})
	for _, file := range crackFiles.Files {
		tw.AppendRow(table.Row{file.Name, util.ByteCountBinary(file.UncompressedSize)})
	}
	con.Printf("%s\n", tw.Render())
}

func PrintCrackFilesByType(crackFiles *clientpb.CrackFiles, con *console.SliverClient) {
	wordlistTable := table.NewWriter()
	wordlistTable.SetTitle(console.StyleBold.Render("Wordlists"))
	wordlistTable.SetStyle(settings.GetTableStyle(con))
	wordlistTable.AppendHeader(table.Row{"Name", "Size"})

	rulesTable := table.NewWriter()
	rulesTable.SetTitle(console.StyleBold.Render("Rules"))
	rulesTable.SetStyle(settings.GetTableStyle(con))
	rulesTable.AppendHeader(table.Row{"Name", "Size"})

	hcTable := table.NewWriter()
	hcTable.SetTitle(console.StyleBold.Render("Markov Hcstat2"))
	hcTable.SetStyle(settings.GetTableStyle(con))
	hcTable.AppendHeader(table.Row{"Name", "Size"})

	wordlists := 0
	rules := 0
	hc := 0
	for _, file := range crackFiles.Files {
		switch file.Type {
		case clientpb.CrackFileType_WORDLIST:
			wordlistTable.AppendRow(table.Row{file.Name, util.ByteCountBinary(file.UncompressedSize)})
			wordlists++
		case clientpb.CrackFileType_RULES:
			rulesTable.AppendRow(table.Row{file.Name, util.ByteCountBinary(file.UncompressedSize)})
			rules++
		case clientpb.CrackFileType_MARKOV_HCSTAT2:
			hcTable.AppendRow(table.Row{file.Name, util.ByteCountBinary(file.UncompressedSize)})
			hc++
		}
	}

	if wordlists > 0 {
		con.Printf("%s\n", wordlistTable.Render())
	}
	if rules > 0 {
		if wordlists > 0 {
			con.Println()
		}
		con.Printf("%s\n", rulesTable.Render())
	}
	if hc > 0 {
		if wordlists > 0 || rules > 0 {
			con.Println()
		}
		con.Printf("%s\n", hcTable.Render())
	}
	con.Println()
	con.Printf("%d wordlists, %d rules, %d hcstat2 files\n", wordlists, rules, hc)
	con.Printf("Disk quota %02.2f%% - %s of %s\n",
		(float64(crackFiles.CurrentDiskUsage)/float64(crackFiles.MaxDiskUsage))*100,
		util.ByteCountBinary(crackFiles.CurrentDiskUsage),
		util.ByteCountBinary(crackFiles.MaxDiskUsage),
	)
}

// CrackWordlistsAddCmd - Manage GPU cracking stations.
func CrackWordlistsAddCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	name, _ := cmd.Flags().GetString("name")

	var localPath string
	if len(args) > 0 {
		localPath = args[0]
	}
	if localPath == "" {
		con.PrintErrorf("No path specified, see --help\n")
		return
	}
	wordlistStat, err := localCrackFileInfo(localPath)
	if err != nil {
		con.PrintErrorf("Invalid wordlist file: %s\n", err)
		return
	}
	if name == "" {
		name = wordlistStat.Name()
	}
	wordlist, err := os.Open(localPath)
	if err != nil {
		con.PrintErrorf("Failed to open file: %s\n", err)
		return
	}
	defer wordlist.Close()

	crackFile, err := con.Rpc.CrackFileCreate(context.Background(), &clientpb.CrackFile{
		Type:             clientpb.CrackFileType_WORDLIST,
		Name:             name,
		UncompressedSize: wordlistStat.Size(),
		IsCompressed:     true,
	})
	if err != nil {
		con.PrintErrorf("Failed to create file: %s\n", err)
		return
	}
	con.PrintInfof("Adding new wordlist '%s' (uncompressed: %s)\n",
		crackFile.Name,
		util.ByteCountBinary(crackFile.UncompressedSize),
	)
	addCrackFile(wordlist, crackFile, con)
}

// CrackRulesAddCmd - add a rules file.
func CrackRulesAddCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	name, _ := cmd.Flags().GetString("name")

	var localPath string
	if len(args) > 0 {
		localPath = args[0]
	}
	if localPath == "" {
		con.PrintErrorf("No path specified, see --help\n")
		return
	}
	rulesStat, err := localCrackFileInfo(localPath)
	if err != nil {
		con.PrintErrorf("Invalid rules file: %s\n", err)
		return
	}
	if name == "" {
		name = rulesStat.Name()
	}
	rules, err := os.Open(localPath)
	if err != nil {
		con.PrintErrorf("Failed to open file: %s\n", err)
		return
	}
	defer rules.Close()

	crackFile, err := con.Rpc.CrackFileCreate(context.Background(), &clientpb.CrackFile{
		Type:             clientpb.CrackFileType_RULES,
		Name:             name,
		UncompressedSize: rulesStat.Size(),
		IsCompressed:     true,
	})
	if err != nil {
		con.PrintErrorf("Failed to create file: %s\n", err)
		return
	}
	con.PrintInfof("Adding new rules file '%s' (uncompressed: %s)\n",
		crackFile.Name,
		util.ByteCountBinary(crackFile.UncompressedSize),
	)
	addCrackFile(rules, crackFile, con)
}

// CrackHcstat2AddCmd - add a hcstat2 file.
func CrackHcstat2AddCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	name, _ := cmd.Flags().GetString("name")

	var localPath string
	if len(args) > 0 {
		localPath = args[0]
	}
	if localPath == "" {
		con.PrintErrorf("No path specified, see --help\n")
		return
	}
	hcstat2Stat, err := localCrackFileInfo(localPath)
	if err != nil {
		con.PrintErrorf("Invalid hcstat2 file: %s\n", err)
		return
	}
	if name == "" {
		name = hcstat2Stat.Name()
	}
	hcstat2, err := os.Open(localPath)
	if err != nil {
		con.PrintErrorf("Failed to open file: %s\n", err)
		return
	}
	defer hcstat2.Close()

	crackFile, err := con.Rpc.CrackFileCreate(context.Background(), &clientpb.CrackFile{
		Type:             clientpb.CrackFileType_MARKOV_HCSTAT2,
		Name:             name,
		UncompressedSize: hcstat2Stat.Size(),
		IsCompressed:     true,
	})
	if err != nil {
		con.PrintErrorf("Failed to create file: %s\n", err)
		return
	}
	con.PrintInfof("Adding new markov hcstat2 file '%s' (uncompressed: %s)\n",
		crackFile.Name,
		util.ByteCountBinary(crackFile.UncompressedSize),
	)
	addCrackFile(hcstat2, crackFile, con)
}

func addCrackFile(localFile *os.File, crackFile *clientpb.CrackFile, con *console.SliverClient) {
	total, err := uploadCrackFile(localFile, crackFile, con.Rpc, func(total int64, n uint32, chunkSize int) {
		con.PrintInfof("Uploading %s (chunk %d - %s) ...",
			util.ByteCountBinary(total),
			n,
			util.ByteCountBinary(int64(chunkSize)),
		)
	})
	con.Printf(console.Clearln + "\r")
	if err != nil {
		con.PrintErrorf("Failed to upload crack file: %s\n", err)
		return
	}
	con.PrintInfof("Upload completed (compressed: %s)\n", util.ByteCountBinary(total))
}

func uploadCrackFile(localFile io.Reader, crackFile *clientpb.CrackFile, rpc rpcpb.SliverRPCClient, progress func(total int64, n uint32, chunkSize int)) (total int64, retErr error) {
	if localFile == nil {
		return 0, errors.New("local crack file reader is nil")
	}
	if crackFile == nil || crackFile.ID == "" {
		return 0, errors.New("crack file reservation is missing an ID")
	}
	if rpc == nil {
		return 0, errors.New("crack file RPC client is nil")
	}

	completed := false
	defer func() {
		if retErr == nil || completed {
			return
		}
		_, cleanupErr := rpc.CrackFileDelete(context.Background(), &clientpb.CrackFile{ID: crackFile.ID})
		if cleanupErr != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed to remove crack file reservation %s: %w", crackFile.ID, cleanupErr))
		}
	}()

	digest := sha256.New()
	wordlistReader := io.TeeReader(localFile, digest)

	chunks, chunkReaderErrors := readCrackFileChunks(wordlistReader, crackFile.ChunkSize)
	n := uint32(0)
	var uploadErr error
	for chunk := range chunks {
		total += int64(len(chunk))
		if uploadErr == nil {
			if progress != nil {
				progress(total, n, len(chunk))
			}
			_, err := rpc.CrackFileChunkUpload(context.Background(), &clientpb.CrackFileChunk{
				CrackFileID: crackFile.ID,
				N:           n,
				Data:        chunk,
			})
			if err != nil {
				uploadErr = fmt.Errorf("upload chunk %d: %w", n, err)
			}
		}
		n++
	}
	chunkReaderErr := <-chunkReaderErrors
	if chunkReaderErr != nil {
		uploadErr = errors.Join(uploadErr, fmt.Errorf("read and compress crack file: %w", chunkReaderErr))
	}
	if uploadErr != nil {
		return total, uploadErr
	}
	_, err := rpc.CrackFileComplete(context.Background(), &clientpb.CrackFile{
		ID:       crackFile.ID,
		Sha2_256: hex.EncodeToString(digest.Sum(nil)),
	})
	if err != nil {
		return total, fmt.Errorf("complete crack file upload: %w", err)
	}
	completed = true
	return total, nil
}

func readCrackFileChunks(reader io.Reader, chunkSize int64) (<-chan []byte, <-chan error) {
	chunks := make(chan []byte, 1) // Chunks are in-memory!
	errors := make(chan error, 1)
	go func() {
		defer close(errors)
		errors <- chunkReader(reader, chunkSize, chunks)
	}()
	return chunks, errors
}

//nolint:gocyclo // Compression, fixed-size emission, final remainder handling, and cleanup form one streaming state machine.
func chunkReader(wordlistReader io.Reader, chunkSize int64, chunks chan []byte) (retErr error) {
	defer close(chunks)
	if chunkSize < 1 {
		return fmt.Errorf("invalid crack file chunk size %d", chunkSize)
	}
	start := int64(0)
	tmpFile, err := os.CreateTemp("", "sliver-wordlist")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer func() {
		if err := tmpFile.Close(); retErr == nil && err != nil {
			retErr = err
		}
		_ = os.Remove(tmpPath)
	}()
	compressor, err := zstd.NewWriter(tmpFile, zstd.WithEncoderLevel(zstd.SpeedBetterCompression))
	if err != nil {
		return err
	}
	compressorClosed := false
	defer func() {
		if !compressorClosed {
			_ = compressor.Close()
		}
	}()
	readBuf := make([]byte, 32*1024*1024) // 32MB
	for {
		readN, readErr := wordlistReader.Read(readBuf)
		if readN != 0 {
			_, err := compressor.Write(readBuf[:readN])
			if err != nil {
				return err
			}
			if err := compressor.Flush(); err != nil {
				return err
			}
			tmpFileStat, err := tmpFile.Stat()
			if err != nil {
				return err
			}
			for tmpFileStat.Size()-start >= chunkSize {
				chunk, err := readChunkAt(tmpFile, start, chunkSize)
				if err != nil {
					return err
				}
				chunks <- chunk
				start += int64(len(chunk))
			}
		}
		if readErr != nil && readErr != io.EOF {
			return readErr
		}
		if readErr == io.EOF {
			break
		}
	}

	if err := compressor.Close(); err != nil {
		return err
	}
	compressorClosed = true
	tmpFileStat, err := tmpFile.Stat()
	if err != nil {
		return err
	}
	for start < tmpFileStat.Size() {
		remaining := tmpFileStat.Size() - start
		readSize := chunkSize
		if remaining < readSize {
			readSize = remaining
		}
		chunk, err := readChunkAt(tmpFile, start, readSize)
		if err != nil {
			return err
		}
		chunks <- chunk
		start += int64(len(chunk))
	}
	return nil
}

func readChunkAt(tmpFile *os.File, offset int64, chunkSize int64) ([]byte, error) {
	chunkBuf := make([]byte, chunkSize)
	n, err := tmpFile.ReadAt(chunkBuf, offset)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return chunkBuf[:n], nil
}

// CrackWordlistsRmCmd - Manage GPU cracking stations.
func CrackWordlistsRmCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var wordlistName string
	if len(args) > 0 {
		wordlistName = args[0]
	}
	if wordlistName == "" {
		con.PrintErrorf("No name specified, see --help\n")
		return
	}
	wordlists, err := con.Rpc.CrackFilesList(context.Background(), &clientpb.CrackFile{Type: clientpb.CrackFileType_WORDLIST})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	found := false
	for _, wordlist := range wordlists.Files {
		if wordlist.Name == wordlistName && wordlist.Type == clientpb.CrackFileType_WORDLIST {
			found = true
			_, err := con.Rpc.CrackFileDelete(context.Background(), wordlist)
			if err != nil {
				con.PrintErrorf("%s\n", err)
				return
			}
			break
		}
	}
	if found {
		con.PrintInfof("Removed wordlist: %s\n", wordlistName)
	} else {
		con.PrintErrorf("Wordlist not found: %s\n", wordlistName)
	}
}

// CrackRulesRmCmd - Manage GPU cracking stations.
func CrackRulesRmCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var rulesName string
	if len(args) > 0 {
		rulesName = args[0]
	}
	if rulesName == "" {
		con.PrintErrorf("No name specified, see --help\n")
		return
	}
	rules, err := con.Rpc.CrackFilesList(context.Background(), &clientpb.CrackFile{Type: clientpb.CrackFileType_RULES})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	found := false
	for _, rulesFile := range rules.Files {
		if rulesFile.Name == rulesName && rulesFile.Type == clientpb.CrackFileType_RULES {
			found = true
			_, err := con.Rpc.CrackFileDelete(context.Background(), rulesFile)
			if err != nil {
				con.PrintErrorf("%s\n", err)
				return
			}
			break
		}
	}
	if found {
		con.PrintInfof("Removed rules: %s\n", rulesName)
	} else {
		con.PrintErrorf("Rules not found: %s\n", rulesName)
	}
}

// CrackHcstat2RmCmd - remove a hcstat2 file.
func CrackHcstat2RmCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var hcstat2Name string
	if len(args) > 0 {
		hcstat2Name = args[0]
	}
	if hcstat2Name == "" {
		con.PrintErrorf("No name specified, see --help\n")
		return
	}
	hcstat2s, err := con.Rpc.CrackFilesList(context.Background(), &clientpb.CrackFile{Type: clientpb.CrackFileType_MARKOV_HCSTAT2})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	found := false
	for _, hcstat2File := range hcstat2s.Files {
		if hcstat2File.Name == hcstat2Name && hcstat2File.Type == clientpb.CrackFileType_MARKOV_HCSTAT2 {
			found = true
			_, err := con.Rpc.CrackFileDelete(context.Background(), hcstat2File)
			if err != nil {
				con.PrintErrorf("%s\n", err)
				return
			}
			break
		}
	}
	if found {
		con.PrintInfof("Removed hcstat2: %s\n", hcstat2Name)
	} else {
		con.PrintErrorf("Hcstat2 not found: %s\n", hcstat2Name)
	}
}
