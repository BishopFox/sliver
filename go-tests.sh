#!/bin/bash

# Server
if go test ./server/certs ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

if go test ./server/encoders ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

if go test ./server/gogo ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

if go test ./server/c2 ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi


# Sliver
if go test ./sliver/proxy ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

if go test ./server/generate ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi