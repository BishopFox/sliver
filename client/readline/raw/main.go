package main

import (
	"fmt"
	"os"

	"github.com/lmorg/readline"
)

func main() {
	readline.MakeRaw(int(os.Stdin.Fd()))

	for {
		b := make([]byte, 1024)
		i, err := os.Stdin.Read(b)
		if err != nil {
			panic(err)
		}

		fmt.Println(b[:i])
	}
}
