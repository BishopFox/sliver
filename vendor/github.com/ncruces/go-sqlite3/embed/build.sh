#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

ROOT=../
BINARYEN="$ROOT/tools/binaryen-version_113/bin"
WASI_SDK="$ROOT/tools/wasi-sdk-20.0/bin"

"$WASI_SDK/clang" --target=wasm32-wasi -flto -g0 -O2 \
	-o sqlite3.wasm "$ROOT/sqlite3/main.c" \
	-I"$ROOT/sqlite3" \
	-mexec-model=reactor \
	-mmutable-globals \
	-mbulk-memory -mreference-types \
	-mnontrapping-fptoint -msign-ext \
	-Wl,--initial-memory=327680 \
	-Wl,--stack-first \
	-Wl,--import-undefined \
	$(awk '{print "-Wl,--export="$0}' exports.txt)

trap 'rm -f sqlite3.tmp' EXIT
"$BINARYEN/wasm-ctor-eval" -g -c _initialize sqlite3.wasm -o sqlite3.tmp
"$BINARYEN/wasm-opt" -g -O2 sqlite3.tmp -o sqlite3.wasm \
	--enable-multivalue --enable-mutable-globals \
	--enable-bulk-memory --enable-reference-types \
	--enable-nontrapping-float-to-int --enable-sign-ext