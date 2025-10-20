package config

import (
	"errors"
	"os"

	"github.com/alecthomas/kong"
	"github.com/moloch--/sgn/utils"
)

var (
	Version = "?"
)

type Options struct {
	Input        string `help:"Input binary path" name:"input" short:"i"`
	Output       string `help:"Encoded output binary name" name:"out" short:"o"`
	Arch         int    `help:"Binary architecture (32/64)" name:"arch" short:"a" default:"64"`
	EncCount     int    `help:"Number of times to encode the binary (increases overall size)" name:"enc" short:"c" default:"1"`
	ObsLevel     int    `help:"Maximum number of bytes for decoder obfuscation" name:"max" short:"M" default:"50"`
	PlainDecoder bool   `help:"Do not encode the decoder stub" name:"plain"`
	AsciiPayload bool   `help:"Generates a full ASCI printable payload (may take very long time to bruteforce)" name:"ascii"`
	Safe         bool   `help:"Preserve all register values (a.k.a. no clobber)" name:"safe" short:"S"`
	BadChars     string `help:"Don't use specified bad characters given in hex format (\\x00\\x01\\x02...)" name:"badchars"`
	Verbose      bool   `help:"Verbose mode" name:"verbose" short:"v"`
	Version      kong.VersionFlag
}

func HelpPrompt(options kong.HelpOptions, ctx *kong.Context) error {
	err := kong.DefaultHelpPrinter(options, ctx)
	if err != nil {
		return err
	}
	return nil
}

func ConfigureOptions() (*Options, error) {
	args := os.Args[1:]
	// Parse arguments and check for errors
	opts := &Options{}
	parser, err := kong.New(
		opts,
		kong.Help(HelpPrompt),
		kong.UsageOnError(),
		kong.Vars{"version": Version},
		kong.ConfigureHelp(kong.HelpOptions{
			Summary: true,
		}),
	)
	if err != nil {
		return nil, err
	}
	_, err = parser.Parse(args)
	if err != nil {
		return nil, err
	}

	if opts.Input == "" {
		return nil, errors.New("input file parameter is mandatory")
	}

	if opts.Verbose {
		utils.Verbose = true
	}

	return opts, nil
}
