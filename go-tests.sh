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


## Util

# util / encoders
if go test -tags=server ./util/encoders ; then
    :
else
    exit 1
fi
if go test -tags=client ./util/encoders ; then
    :
else
    exit 1
fi

## Server

# server / website
if go test -tags=server ./server/website ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / loot
if go test -tags=server ./server/loot ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / certs
if go test -tags=server ./server/certs ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / cryptography
if go test -tags=server ./server/cryptography ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / gogo
if go test -tags=server ./server/gogo ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / c2
if go test -tags=server ./server/c2 ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / configs
if go test -tags=server ./server/configs ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / console
if go test -tags=server ./server/console ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi

# server / generate
if go test -tags=server ./server/generate -timeout 6h ; then
    :
else
    cat ~/.sliver/logs/sliver.log
    exit 1
fi
