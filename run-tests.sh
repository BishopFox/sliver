#!/bin/bash

if ! [ -x "$(command -v docker)" ]; then
  echo 'Error: docker is not installed.' >&2
  exit 1
fi

echo -e "\n[*] Go grab some coffee, this takes a few minutes ...\n"
docker build -t sliver-tests . && docker run --rm  -it sliver-tests:latest $@
