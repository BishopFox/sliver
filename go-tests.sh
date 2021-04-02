#!/bin/bash

# Sliver Implant Framework
# Copyright (C) 2019  Bishop Fox

# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.

# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.

# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.

# Variables passed as  arguments to the script, from Makefile caller.
# See ".PHONY: tests" in the Makefile for arguments passed to it.
ENV=$1
GO=$2
TAGS="$3 $4"

# All remaining are ldflags
shift 4
LDFLAGS="$1"
shift
FLAGS="$@"

echo "Testing with build command:"
echo $ENV $GO test $TAGS  $LDFLAGS \"$FLAGS\"

## Util

function testDir() {
        $GO test $1 -trimpath $TAGS,server 
        return
}

# comm= $(ENV) $(GO) test -trimpath $(TAGS),server $(LDFLAGS)

# util 
if testDir ./util ; then
# if go test ./util ; then
    :
else
    exit 1
fi

# util / encoders
if testDir ./util/encoders ; then
    :
else
    exit 1
fi

## Server

# server / website
if testDir ./server/website ; then
    :
else
    # cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / certs
if testDir ./server/certs ; then
    :
else
    # cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / cryptography
if testDir ./server/cryptography ; then
    :
else
    # cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / gogo
if testDir ./server/gogo ; then
    :
else
    # cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / c2
if testDir ./server/c2 ; then
    :
else
    # cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / generate
if testDir ./server/generate -timeout 6h ; then
    :
else
    # cat ~/.sliver/logs/sliver.log
    exit 1
fi
