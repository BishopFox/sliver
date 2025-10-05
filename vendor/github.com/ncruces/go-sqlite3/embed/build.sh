#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

ROOT=../
BINARYEN="$ROOT/tools/binaryen/bin"
WASI_SDK="$ROOT/tools/wasi-sdk/bin"

trap 'rm -f sqlite3.tmp' EXIT

"$WASI_SDK/clang" --target=wasm32-wasi -std=c23 -g0 -O2 \
	-Wall -Wextra -Wno-unused-parameter -Wno-unused-function \
	-o sqlite3.wasm "$ROOT/sqlite3/main.c" \
	-I"$ROOT/sqlite3/libc" -I"$ROOT/sqlite3" \
	-mexec-model=reactor \
	-mmutable-globals -mnontrapping-fptoint \
	-msimd128 -mbulk-memory -msign-ext \
	-mreference-types -mmultivalue \
	-mno-extended-const \
	-fno-stack-protector \
	-Wl,--stack-first \
	-Wl,--import-undefined \
	-Wl,--initial-memory=327680 \
	-D_HAVE_SQLITE_CONFIG_H \
	-DSQLITE_CUSTOM_INCLUDE=sqlite_opt.h \
	$(awk '{print "-Wl,--export="$0}' exports.txt)

"$BINARYEN/wasm-ctor-eval" -g -c _initialize sqlite3.wasm -o sqlite3.tmp
"$BINARYEN/wasm-opt" -g sqlite3.tmp -o sqlite3.wasm \
	--low-memory-unused --gufa --generate-global-effects --converge -O3 \
	--enable-mutable-globals --enable-nontrapping-float-to-int \
	--enable-simd --enable-bulk-memory --enable-sign-ext \
	--enable-reference-types --enable-multivalue \
	--strip --strip-producers