// Package kong aims to support arbitrarily complex command-line structures with as little developer effort as possible.
//
// Here's an example:
//
//     shell rm [-f] [-r] <paths> ...
//     shell ls [<paths> ...]
//
// This can be represented by the following command-line structure:
//
//     package main
//
//     import "github.com/alecthomas/kong"
//
//     var CLI struct {
//       Rm struct {
//         Force     bool `short:"f" help:"Force removal."`
//         Recursive bool `short:"r" help:"Recursively remove files."`
//
//         Paths []string `arg help:"Paths to remove." type:"path"`
//       } `cmd help:"Remove files."`
//
//       Ls struct {
//         Paths []string `arg optional help:"Paths to list." type:"path"`
//       } `cmd help:"List paths."`
//     }
//
//     func main() {
//       kong.Parse(&CLI)
//     }
//
// See https://github.com/alecthomas/kong for details.
package kong
