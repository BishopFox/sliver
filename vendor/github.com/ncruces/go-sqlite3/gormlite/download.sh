#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

curl -#OL "https://github.com/go-gorm/sqlite/raw/master/ddlmod.go"
curl -#OL "https://github.com/go-gorm/sqlite/raw/master/ddlmod_test.go"
curl -#OL "https://github.com/go-gorm/sqlite/raw/master/error_translator.go"
curl -#OL "https://github.com/go-gorm/sqlite/raw/master/migrator.go"
curl -#OL "https://github.com/go-gorm/sqlite/raw/master/sqlite.go"
curl -#OL "https://github.com/go-gorm/sqlite/raw/master/sqlite_test.go"