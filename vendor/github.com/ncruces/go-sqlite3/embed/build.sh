#!/usr/bin/env bash
set -eo pipefail

cd -P -- "$(dirname -- "$0")"

# build SQLite
zig cc --target=wasm32-wasi -flto -g0 -O2 \
  -o sqlite3.wasm ../sqlite3/main.c \
	-I../sqlite3/ \
	-mmutable-globals \
	-mbulk-memory -mreference-types \
	-mnontrapping-fptoint -msign-ext \
	-mexec-model=reactor \
	-D_HAVE_SQLITE_CONFIG_H \
	$(awk '{print "-Wl,--export="$0}' exports.txt)