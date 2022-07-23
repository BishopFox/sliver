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

echo "----------------------------------------------------------------"
echo "WARNING: Running unit tests on slow systems can take a LONG time"
echo "         Recommended to only run on 16+ CPU cores and 32Gb+ RAM"
echo "----------------------------------------------------------------"
TAGS=osusergo,netgo,cgosqlite,sqlite_omit_load_extension

## Client

# client / command / alias
if go test -tags=client,$TAGS ./client/command/alias ; then
    :
else
    exit 1
fi

# client / command / extensions
if go test -tags=client,$TAGS ./client/command/extensions ; then
    :
else
    exit 1
fi

## Util

# util 
if go test -tags=server,$TAGS ./util ; then
    :
else
    exit 1
fi

# util / encoders
if go test -tags=server,$TAGS ./util/encoders/basex ; then
    :
else
    exit 1
fi
if go test -tags=server,$TAGS ./util/encoders ; then
    :
else
    exit 1
fi
if go test -tags=client,$TAGS ./util/encoders ; then
    :
else
    exit 1
fi

## Implant

# implant / sliver / transports / dnsclient
if go test -tags=server,$TAGS ./implant/sliver/transports/dnsclient ; then
    :
else
    exit 1
fi

## Server

# server / website
if go test -tags=server,$TAGS ./server/website ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / loot
if go test -tags=server,$TAGS ./server/loot ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / certs
if go test -tags=server,$TAGS ./server/certs ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / cryptography
if go test -tags=server,$TAGS ./server/cryptography ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / cryptography / minisign
if go test -tags=server,$TAGS ./server/cryptography/minisign ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / gogo
if go test -tags=server,$TAGS ./server/gogo ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / c2
if go test -tags=server,$TAGS ./server/c2 ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / configs
if go test -tags=server,$TAGS ./server/configs ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / console
if go test -tags=server,$TAGS ./server/console ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / generate
export GOPROXY=off
if go test -tags=server,$TAGS ./server/generate -timeout 6h ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi
