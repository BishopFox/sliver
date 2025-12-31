**NOTE:** Any platform can cross-compile a standalone executable to any other platform out of the box; you only need cross-compilers when using `--format shared`.

Sliver can tell you which platforms it can likely target based on the server's platform and available cross-compilers by running the `generate info` command in the console.

Sliver supports [External Builders](/docs?name=External+Builders), which can be used to easily cross-compile implants.

## From Linux to MacOS/Windows

Sliver embeds a Zig cross-compiler out of the box, so Windows shared library and shellcode implants from Linux do not require mingw-w64.

To compile MacOS shared library implants from Linux, we recommend using https://github.com/tpoechtrager/osxcross by default Sliver will look in `/opt/osxcross` but you can override this via [environment variables](/docs?name=Environment+Variables). If you do not have a MacOS-based machine you can use GitHub Actions' MacOS instances to build OSXCross.

**NOTE:** Sliver expects the root of the osxcross git repo to be located at `/opt/osxcross` and the actual binaries in `/opt/osxcross/target/bin`.

An example deployment is shown below, you have to procure the `MacOSX123.sdk.tar.xz` yourself due to license restrictions (see the OSXCross GitHub for more details). It is recommended to use the llvm-based version (i.e. the `2.0-llvm-based` branch) of osxcross:

For a MacOS machine, if you don't have one handy I recommend using a MacOS GitHub Actions runner:

```shell
git clone https://github.com/tpoechtrager/osxcross.git
cd osxcross
./tools/gen_sdk_package.sh
```

From your Linux server, commands may differ slightly depending on your Linux distribution, just ask ChatGPT for help if you need it:

```shell
sudo apt-get install -y clang cmake git patch python3 libssl-dev lzma-dev libxml2-dev xz-utils bzip2 cpio libbz2-dev zlib1g-dev llvm-dev uuid-dev
git clone --depth 1 -b 2.0-llvm-based https://github.com/tpoechtrager/osxcross.git
cd osxcross
curl -o ./tarballs/MacOSX123.sdk.tar.xz 'https://example.com/MacOSX123.sdk.tar.xz'

TARGET_DIR=/opt/osxcross ENABLE_ARCHS="arm64 x86_64" ./build.sh
```

Sliver automatically looks in the default paths for these cross-compilers; once installed simply use the `generate` command with the desired `--os` and `--arch`, check `~/.sliver/logs/sliver.log` for build errors. You can override any cross-compiler location via the appropriate [environment variables](/docs?name=Environment+Variables).

## From MacOS to Linux/Windows

Sliver embeds a Zig cross-compiler out of the box, so Windows DLLs and Linux shared objects do not require mingw-w64 or musl-cross.

However, we're not aware of any good options to target 32-bit Linux from MacOS. Sliver automatically looks in the default paths for these cross-compilers; once installed simply use the `generate` command with the desired `--os` and `--arch`, check `~/.sliver/logs/sliver.log` for build errors. You can override any cross-compiler location via the appropriate [environment variables](/docs?name=Environment+Variables).

## From Windows to MacOS/Linux

Good luck.
