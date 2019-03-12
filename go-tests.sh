#!/bin/bash

# Server
go test ./server/certs
go test ./server/encoders
go test ./server/gogo
go test ./server/c2

# Sliver
go test ./sliver/proxy

# Server & Sliver 
go test ./server/generate
