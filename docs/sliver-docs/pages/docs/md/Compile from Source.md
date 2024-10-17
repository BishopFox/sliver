You'll want to compile from a MacOS or Linux machine, compiling from native Windows in theory is possible, but none of the asset scripts are designed to run on Windows, you can cross-compile the Windows server/client binaries from a better operating system like Linux or MacOS. If you only have a Windows machine see "Windows Builds" below (TL;DR use WSL).

# Sliver v1.6.x

- Go v1.21 or later
- `make`, `sed`, `tar`, `curl`, `zip`, `cut` commands; most of these are installed by default but you may need to install `make`, `curl`, and `zip` depending on your distribution. On MacOS you may need to install XCode and accompanying cli tools.

```asciinema
{"src": "/asciinema/compile-from-source.cast", "cols": "132"}
```

### Compiling

```
$ git clone https://github.com/BishopFox/sliver.git
$ cd sliver
```

**IMPORTANT:** The Sliver Makefile requires version information from the git repository, so you must `git clone` the repository. Using GitHub's "download zip" feature may omit the `.git` directory and result in broken builds.

By default `make` will build whatever platform you're currently running on:

```
$ make
```

This will create `sliver-server` and `sliver-client` binaries.

### Docker Build

There are a few Docker targets available depending on your needs:

- `test` - Runs the unit tests
- `production` - Builds the production Docker image, including optional dependencies like Metasploit
- `production-slim` - Builds the production Docker image, but without Metasploit and other optional dependencies

From the project root directory run:

```
docker build --target production -t sliver .
```

#### Compiling Sliver on Kali Linux

```asciinema
{"src": "/asciinema/sliver-docker-production.cast", "cols": "132"}
```

The Docker build includes mingw and Metasploit, so it can take a while to build from scratch but Docker should cache the layers effectively. Sliver will also run it's unit tests as part of the build, and that takes a few minutes too.

# Developers

If you want to modify any of the `.proto` files you'll need to setup a few additional tools to regenerate the `.pb.go` files.

- Protoc libprotoc v26.1 or later
- Protoc-gen-go v1.27.1
- Protoc-gen-go-grpc v1.2.0

#### `protoc`

First install your platform's version of `protoc` v3.19.4 or later:

https://github.com/protocolbuffers/protobuf/releases/latest

Ensure that correct `protoc` version is on your `$PATH`, you can check with a simple `protoc --version`

#### `protoc-gen-go` `protoc-gen-go-grpc`

Assuming `$GOPATH/bin` is on your `$PATH` simply run the following commands to install the appropriate versions of `protoc-gen-go` and `protoc-gen-go-grpc`:

```
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.27.1
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2.0
```

Ensure that these are both on your `$PATH` after running the commands, if not you probably need to add `$GOPATH/bin` to your `$PATH`. To regenerate the Protobuf and gRPC files run:

```
$ make pb
```

# Sliver v1.5.x

- When compiling v1.5.x use Go v1.20.7 you can check your version using `go version`. You can use the [Go Version Manager](https://github.com/moovweb/gvm) to install and manage multiple versions of Go.
- `make`, `sed`, `tar`, `curl`, `zip`, `cut` commands; most of these are installed by default but you may need to install `make`, `curl`, and `zip` depending on your distribution. On MacOS you may need to install XCode and accompanying cli tools.

### Compiling

First git clone the repository:

```
$ git clone https://github.com/BishopFox/sliver.git
$ cd sliver
```

The `master` branch will contain the latest Sliver features, however only release version of Sliver are recommended for production use. It is strongly recommended to checkout the latest tagged release branch when compiling from source unless you're a developer:

```
$ git checkout tags/v1.5.42
```

Sliver embeds its own copy of the Go compiler and a few internal tools, the first time your run `make` a bash script will run to download these assets to your local system. This means the first build will take much longer than subsequent builds, especially if your internet connection is slow.

By default `make` will build whatever platform you're currently running on:

```
$ make
```

**IMPORTANT:** The Sliver Makefile requires version information from the git repository, so you must `git clone` the repository. Using GitHub's "download zip" feature may omit the `.git` directory and result in broken builds.

This will create `sliver-server` and `sliver-client` binaries.

### Cross-compile to Specific Platforms

You can also specify a target platform for the `make` file, though you may need cross-compilers (see below):

```
$ make macos
$ make macos-arm64
$ make linux
$ make linux-arm64
$ make windows-amd64
```

### Docker Build

The Docker builds are mostly designed for running unit tests, but can be useful if you want a "just do everything" build:

```
docker build -t sliver .
```

The Docker build includes mingw and Metasploit, so it can take a while to build from scratch but Docker should cache the layers effectively. Sliver will also run it's unit tests as part of the build, and that takes a few minutes too.

### Windows Builds

**NOTE:** In versions of Sliver prior to 1.6.0 you will need mingw to cross-compile Windows builds:

- Kali/Ubuntu/Debian `sudo apt install mingw-w64`
- MacOS `brew install mingw-w64`

If all you have is a Windows machine, the easiest way to build Sliver is using [WSL](https://docs.microsoft.com/en-us/windows/wsl/install-win10) and following the Linux/cross-compile instructions above. To cross-compile a native Windows binary use `make windows-amd64` and copy it to your Windows file system (i.e. `/mnt/c/Users/foo/Desktop`) and run it using a terminal that supports ANSI sequences such as the [Windows Terminal](https://github.com/microsoft/terminal).
