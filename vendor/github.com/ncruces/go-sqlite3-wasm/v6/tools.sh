#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" ]]; then
  WASI_SDK="x86_64-windows"
  BINARYEN="x86_64-windows"
elif [[ "$OSTYPE" == "linux"* ]]; then
  if [[ "$(uname -m)" == "x86_64" ]]; then
    WASI_SDK="x86_64-linux"
    BINARYEN="x86_64-linux"
  else
    WASI_SDK="arm64-linux"
    BINARYEN="aarch64-linux"
  fi
elif [[ "$OSTYPE" == "darwin"* ]]; then
  if [[ "$(uname -m)" == "x86_64" ]]; then
    WASI_SDK="x86_64-macos"
    BINARYEN="x86_64-macos"
  else
    WASI_SDK="arm64-macos"
    BINARYEN="arm64-macos"
  fi
fi

WASI_SDK="https://github.com/WebAssembly/wasi-sdk/releases/download/wasi-sdk-34/wasi-sdk-34.0-$WASI_SDK.tar.gz"
BINARYEN="https://github.com/WebAssembly/binaryen/releases/download/version_132/binaryen-version_132-$BINARYEN.tar.gz"

# Download tools
rm -rf "tools/"
mkdir -p "tools/"
curl -#L "$WASI_SDK" | tar xzC "tools/" &
curl -#L "$BINARYEN" | tar xzC "tools/" &
wait

mv "tools/wasi-sdk"* "tools/wasi-sdk"
mv "tools/binaryen"* "tools/binaryen"
