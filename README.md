Sliver
======



## Building From Scratch

You'll want to compile from a MacOS or Linux machine, compiling from Windows should work but none of the scripts are designed to run on Windows (you can compile the Windows binaries from MacOS or Linux).

Requirements:
* Go v1.11 or later
* Make, sed, tar, wget, zip

Build thin server (for developement)

```
$ ./deps.sh
$ ./go-assets.sh
$ make
```

Statically compile and bundle server with all dependencies and assets:

```
$ make static-macos
$ make static-linux
$ make static-windows
```
