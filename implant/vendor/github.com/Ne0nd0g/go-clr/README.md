# go-clr
[![GoDoc](https://godoc.org/github.com/ropnop/go-clr?status.svg)](https://godoc.org/github.com/ropnop/go-clr)

This is my PoC code for hosting the CLR in a Go process and using it to execute a DLL from disk or an assembly from memory.

It's written in pure Go by just wrapping the needed syscalls and making use of a lot of unsafe.Pointers to 
load structs from memory.

For more info and references, see [this blog post](https://blog.ropnop.com/hosting-clr-in-golang/).

This was was a fun project and proof of concept, but the code is definitely not "production ready". It makes heavy use 
of `unsafe` and it's probably very unstable. I don't plan on supporting it much moving forward,
but I wanted to share the code and knowledge to enable others to either contribute, or fork and make their own awesome tools.

## Installation and Usage
`go-clr` is intended to be used as a package in other scripts. Install it with:
```bash
go get github.com/ropnop/go-clr
```

Take a look at the [examples](./examples) folder for some examples on how to leverage it. The package exposes all the structs and methods
necessary to customize, but it also includes two "magic" functions to execute .NET from Go: `ExecuteDLLFromDisk` and
`ExecuteByteArray`. Here's a quick example of using both:

```go
package main

import (
	clr "github.com/ropnop/go-clr"
	"log"
	"fmt"
	"io/ioutil"
	"runtime"
)

func main() {
	fmt.Println("[+] Loading DLL from Disk")
	ret, err := clr.ExecuteDLLFromDisk(
		"TestDLL.dll",
		"TestDLL.HelloWorld",
		"SayHello",
		"foobar")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("[+] DLL Return Code: %d\n", ret)

	
	fmt.Println("[+] Executing EXE from memory")
	exebytes, err := ioutil.ReadFile("helloworld.exe")
	if err != nil {
		log.Fatal(err)
	}
	runtime.KeepAlive(exebytes)

	ret2, err := clr.ExecuteByteArray(exebytes)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("[+] EXE Return Code: %d\n", ret2)
}
``` 

The other 2 examples show the same technique but without the magic functions.

### License
This project is licensed under the [Do What the Fuck You Want to Public License](http://www.wtfpl.net/). I deliberately
chose this "joke" license because I really don't think anyone should be using this for anything serious, and I know
some organizations forbid this license from being used in products (which is a good thing).