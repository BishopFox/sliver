Sliver can be compiled on Linux, macOS, and native Windows. Windows users can also build through the Windows Subsystem for Linux (WSL).

To compile from source you'll need:

- Go v1.27.1 or later
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

Sliver embeds its own copy of the Go compiler and a few internal tools. The first time you run `make`, a Go asset tool will download these assets to your local system. This means the first build will take longer than subsequent builds, especially if your internet connection is slow.

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

The production Docker image includes Metasploit, so it can take a while to build from scratch, but Docker caches the layers. Unit tests run only when building the separate `test` target.

### Windows Builds

For a native Windows build, install Go, Git, and GNU Make, then run `make windows-amd64` from the repository root. Alternatively, use [WSL](https://docs.microsoft.com/en-us/windows/wsl/install-win10) and follow the Linux/cross-compile instructions above. Run the resulting binaries in a terminal that supports ANSI sequences, such as [Windows Terminal](https://github.com/microsoft/terminal).

# Developers

If you want to modify any of the `.proto` files you'll need to set up a few additional tools to regenerate the `.pb.go` files.

The checked-in bindings record these tool versions:

- `protoc` v7.36.2 (as reported in the generated headers)
- `protoc-gen-go` v1.36.11
- `protoc-gen-go-grpc` v1.6.1

#### `protoc`

Install a compatible `protoc` release for your platform:

https://github.com/protocolbuffers/protobuf/releases/latest

Ensure that the correct `protoc` version is on your `$PATH`, you can check with a simple `protoc --version`

#### `protoc-gen-go` `protoc-gen-go-grpc`

Assuming `$GOPATH/bin` is on your `$PATH` simply run the following commands to install the appropriate versions of `protoc-gen-go` and `protoc-gen-go-grpc`:

```
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.1
```

Ensure that these are both on your `$PATH` after running the commands; if not, you probably need to add `$GOPATH/bin` to your `$PATH`. To regenerate the Protobuf and gRPC files run:

```
$ make pb
```
