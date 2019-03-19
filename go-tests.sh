#!/bin/bash

# Server
go test ./server/certs
go test ./server/encoders
go test ./server/gogo
go test ./server/c2

# Sliver
go test ./sliver/proxy


if go test ./server/generate ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi