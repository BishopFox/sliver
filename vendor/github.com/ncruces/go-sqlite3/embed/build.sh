#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

ROOT=../
BINARYEN="$ROOT/tools/binaryen-version_117/bin"
WASI_SDK="$ROOT/tools/wasi-sdk-22.0/bin"

"$WASI_SDK/clang" --target=wasm32-wasi -std=c17 -flto -g0 -O2 \
	-Wall -Wextra -Wno-unused-parameter \
	-o sqlite3.wasm "$ROOT/sqlite3/main.c" \
	-I"$ROOT/sqlite3" \
	-mexec-model=reactor \
	-msimd128 -mmutable-globals \
	-mbulk-memory -mreference-types \
	-mnontrapping-fptoint -msign-ext \
	-fno-stack-protector -fno-stack-clash-protection \
	-Wl,--initial-memory=327680 \
	-Wl,--stack-first \
	-Wl,--import-undefined \
	-D_HAVE_SQLITE_CONFIG_H \
	$(awk '{print "-Wl,--export="$0}' exports.txt)

trap 'rm -f sqlite3.tmp' EXIT
"$BINARYEN/wasm-ctor-eval" -g -c _initialize sqlite3.wasm -o sqlite3.tmp
"$BINARYEN/wasm-opt" -g --strip --strip-producers -c -O3 \
	sqlite3.tmp -o sqlite3.wasm \
	--enable-simd --enable-mutable-globals --enable-multivalue \
	--enable-bulk-memory --enable-reference-types \
	--enable-nontrapping-float-to-int --enable-sign-ext