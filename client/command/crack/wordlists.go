package crack

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/util"
	"github.com/desertbit/grumble"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/klauspost/compress/zstd"
)

// CrackWordlistsCmd - Manage GPU cracking stations
func CrackWordlistsCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	wordlists, err := con.Rpc.CrackFilesList(context.Background(), &clientpb.CrackFile{Type: clientpb.CrackFileType_WORDLIST})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if len(wordlists.Files) == 0 {
		con.PrintInfof("No wordlists, add some using `crack wordlists add`\n")
	} else {
		PrintWordlists(wordlists, con)
	}
}

func PrintWordlists(crackFiles *clientpb.CrackFiles, con *console.SliverConsoleClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{"Name", "Size"})
	for _, file := range crackFiles.Files {
		tw.AppendRow(table.Row{file.Name, util.ByteCountBinary(file.UncompressedSize)})
	}
	con.Printf("%s\n", tw.Render())
	con.Println()
	con.Printf("Disk quota %02.2f%% - %s of %s\n",
		(float64(crackFiles.CurrentDiskUsage)/float64(crackFiles.MaxDiskUsage))*100,
		util.ByteCountBinary(crackFiles.CurrentDiskUsage),
		util.ByteCountBinary(crackFiles.MaxDiskUsage),
	)
}

// CrackWordlistsAddCmd - Manage GPU cracking stations
func CrackWordlistsAddCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	name := ctx.Flags.String("name")
	localPath := ctx.Args.String("path")
	if localPath == "" {
		con.PrintErrorf("No path specified, see --help\n")
		return
	}
	wordlistStat, err := os.Stat(localPath)
	if os.IsNotExist(err) || wordlistStat.IsDir() {
		con.PrintErrorf("File does not exist: %s\n", localPath)
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

	digest := sha256.New()
	wordlistReader := io.TeeReader(wordlist, digest)

	chunks := make(chan []byte, 1) // Chunks are in-memory!

	var chunkReaderErr error
	go func() {
		chunkReaderErr = chunkReader(wordlistReader, crackFile.ChunkSize, chunks)
	}()
	n := uint32(0)
	total := int64(0)
	errors := []error{}
	for chunk := range chunks {
		total += int64(len(chunk))
		con.PrintInfof("Uploading %s (chunk %d - %s) ...",
			util.ByteCountBinary(total),
			n,
			util.ByteCountBinary(int64(len(chunk))),
		)
		_, err = con.Rpc.CrackFileChunkUpload(context.Background(), &clientpb.CrackFileChunk{
			CrackFileID: crackFile.ID,
			N:           n,
			Data:        chunk,
		})
		n++
		if err != nil {
			errors = append(errors, err)
			continue
		}
	}
	con.Printf(console.Clearln + "\r")
	if chunkReaderErr != nil {
		con.PrintErrorf("Failed to read file: %s\n", err)
		return
	}
	if len(errors) > 0 {
		for _, err := range errors {
			con.PrintErrorf("Failed to upload chunk: %s\n", err)
		}
		return
	}
	_, err = con.Rpc.CrackFileComplete(context.Background(), &clientpb.CrackFile{
		ID:       crackFile.ID,
		Sha2_256: hex.EncodeToString(digest.Sum(nil)),
	})
	if err != nil {
		con.PrintErrorf("Failed to complete file upload: %s\n", err)
		return
	}
	con.PrintInfof("Upload completed (compressed: %s)\n", util.ByteCountBinary(total))
}

func chunkReader(wordlistReader io.Reader, chunkSize int64, chunks chan []byte) error {
	defer close(chunks)
	start := int64(0)
	tmpFile, err := os.CreateTemp("", "sliver-wordlist")
	if err != nil {
		return err
	}
	defer tmpFile.Close()
	compressor, _ := zstd.NewWriter(tmpFile, zstd.WithEncoderLevel(zstd.SpeedBetterCompression))
	for {
		readBuf := make([]byte, 32*1024*1024) // 32MB
		readN, readErr := wordlistReader.Read(readBuf)
		if readErr != nil && readErr != io.EOF {
			return err
		}
		if readN != 0 {
			_, err := compressor.Write(readBuf[:readN])
			if err != nil {
				return err
			}
			compressor.Flush()
			tmpFileStat, err := os.Stat(tmpFile.Name())
			if err != nil {
				return err
			}
			if tmpFileStat.Size()-start >= chunkSize {
				stop := start + chunkSize
				chunk, err := readChunkAt(tmpFile, start, chunkSize)
				if err != nil {
					return err
				}
				chunks <- chunk
				start = stop
			}
		}
		if readErr == io.EOF {
			chunk, err := readChunkAt(tmpFile, start, chunkSize)
			if err != nil {
				return err
			}
			chunks <- chunk
			return nil
		}
	}
}

func readChunkAt(tmpFile *os.File, offset int64, chunkSize int64) ([]byte, error) {
	chunkBuf := make([]byte, chunkSize)
	n, err := tmpFile.ReadAt(chunkBuf, offset)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return chunkBuf[:n], nil
}

// CrackWordlistsRmCmd - Manage GPU cracking stations
func CrackWordlistsRmCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	wordlistName := ctx.Args.String("name")
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
