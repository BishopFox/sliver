#!/bin/bash

## Server
# server / db
if go test ./server/db ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / certs
if go test ./server/certs ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / encoders
if go test ./server/encoders ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / gogo
if go test ./server/gogo ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / c2
if go test ./server/c2 ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / generate
if go test ./server/generate -timeout 6h ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi


## Sliver
# sliver / proxy
if go test ./sliver/proxy ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi