# Sliver / gobfuscate

This directory contains a highly modified version of gobfuscate, it's been modified to work with the Sliver build process.

It also contains a modified version of the `https://github.com/golang/tools/refactor/rename` tool, it has been modified to log messages instead of writing them stdout, ignore "DO NOT EDIT" tags, and a few other tweaks.

## gobfuscate

When you compile a Go binary, it contains a lot of information about your source code: field names, strings, package paths, etc. If you want to ship a binary without leaking this kind of information, what are you to do?

With gobfuscate, you can compile a Go binary from obfuscated source code. This makes a lot of information difficult or impossible to decipher from the binary.

## What it does

Currently, gobfuscate manipulates package names, global variable and function names, type names, method names, and strings.

### Package name obfuscation

When gobfuscate builds your program, it constructs a copy of a subset of your GOPATH. It then refactors this GOPATH by encrypting package names and paths. As a result, a package like "github.com/unixpickle/deleteme" becomes something like "jiikegpkifenppiphdhi/igijfdokiaecdkihheha/jhiofoppieegdaif". This helps get rid of things like Github usernames from the executable.

**Limitation:** currently, packages which use CGO cannot be renamed. I suspect this is due to a bug in Go's refactoring API.

### Global names

Gobfuscate encrypts the names of global vars, consts, and funcs. It also encrypts the names of any newly-defined types.

Due to restrictions in the refactoring API, this does not work for packages which contain assembly files or use CGO. It also does not work for names which appear multiple times because of build constraints.

### Struct methods

Gobfuscate encrypts the names of most struct methods. However, it does not rename methods whose names match methods of any imported interfaces. This is mostly due to internal constraints from the refactoring engine. Theoretically, most interfaces could be obfuscated as well (except for those in the standard library).

Due to restrictions in the refactoring API, this does not work for packages which contain assembly files or use CGO. It also does not work for names which appear multiple times because of build constraints.

### Strings

Strings are obfuscated by replacing them with functions. A string will be turned into an expression like the following:

```go
(func() string {
	mask := []byte{33, 15, 199}
	maskedStr := []byte{73, 106, 190}
	res := make([]byte, 3)
	for i, m := range mask {
		res[i] = m ^ maskedStr[i]
	}
	return string(res)
}())
```

Since `const` declarations cannot include function calls, gobfuscate tries to change any `const` strings into `var`s. It works for declarations like any of the following:

```
const MyStr = "hello"
const MyStr1 = MyStr + "yoyo"
const MyStr2 = MyStr + (MyStr1 + "hello1")

const (
  MyStr3 = "hey there"
  MyStr4 = MyStr1 + "yo"
)
```

However, it does not work for mixed const/int blocks:

```
const (
  MyStr = "hey there"
  MyNum = 3
)
```

## License

This is under a BSD 2-clause license. See [LICENSE](LICENSE).
