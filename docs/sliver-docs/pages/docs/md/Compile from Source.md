You'll want to compile from a MacOS or Linux machine. Compiling from native Windows is possible in theory, but none of the asset scripts are designed to run on Windows. You can cross-compile the Windows server/client binaries from a better operating system like Linux or MacOS. If you only have a Windows machine, see "Windows Builds" below (TL;DR: use WSL).

# Sliver v1.6.x

To compile from source you'll need:

- Go v1.25 or later
- `make` (on MacOS you may need to install XCode and accompanying cli tools)

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

Sliver embeds its own copy of the Go compiler and a few internal tools. The first time you run `make`, a bash script will download these assets to your local system. This means the first build will take longer than subsequent builds, especially if your internet connection is slow.

### Cross-compile to Specific Platforms

You can also specify a target platform for the `make` file:

```
$ GOOS=windows GOARCH=amd64 make
```

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

The Docker build includes mingw and Metasploit, so it can take a while to build from scratch but Docker should cache the layers effectively. Sliver will also run its unit tests as part of the build, and that takes a few minutes too.

### Windows Builds

If all you have is a Windows machine, the easiest way to build Sliver is using [WSL](https://docs.microsoft.com/en-us/windows/wsl/install-win10) and following the Linux/cross-compile instructions above. To cross-compile a native Windows binary use `make windows-amd64` and copy it to your Windows file system (i.e. `/mnt/c/Users/foo/Desktop`) and run it using a terminal that supports ANSI sequences such as the [Windows Terminal](https://github.com/microsoft/terminal).

# Developers

If you want to modify any of the `.proto` files you'll need to set up a few additional tools to regenerate the `.pb.go` files.

- Protoc libprotoc v26.1 or later
- Protoc-gen-go v1.27.1
- Protoc-gen-go-grpc v1.2.0

#### `protoc`

First install your platform's version of `protoc` v3.19.4 or later:

https://github.com/protocolbuffers/protobuf/releases/latest

Ensure that the correct `protoc` version is on your `$PATH`, you can check with a simple `protoc --version`

#### `protoc-gen-go` `protoc-gen-go-grpc`

Assuming `$GOPATH/bin` is on your `$PATH` simply run the following commands to install the appropriate versions of `protoc-gen-go` and `protoc-gen-go-grpc`:

```
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.27.1
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2.0
```

Ensure that these are both on your `$PATH` after running the commands; if not, you probably need to add `$GOPATH/bin` to your `$PATH`. To regenerate the Protobuf and gRPC files run:

```
$ make pb
```
