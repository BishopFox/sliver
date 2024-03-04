#
# For production:
#   docker build --target production -t sliver .
# Multi-arch production build:
#   export BUILDX_NO_DEFAULT_ATTESTATIONS=1 # handle unknown/unknown image push
#   docker buildx build --platform linux/amd64,linux/arm64 --target production -t sliver .
# Run sliver container:
#   docker run -it --rm -v $HOME/.sliver:/home/sliver/.sliver sliver
#
# For unit testing:
#   docker build --target test .
# Multi-arch build for unit testing:
#   export BUILDX_NO_DEFAULT_ATTESTATIONS=1 # handle unknown/unknown image push
#   docker buildx build --platform linux/amd64,linux/arm64,darwin/arm64 --target test .
#

# STAGE: base
## Compiles Sliver for use
FROM golang:1.21.4 as base

### Base packages
RUN apt-get update --fix-missing && apt-get -y install \
    git build-essential zlib1g zlib1g-dev wget zip unzip

### Add sliver user
RUN groupadd -g 999 sliver && useradd -r -u 999 -g sliver sliver
RUN mkdir -p /home/sliver/ && chown -R sliver:sliver /home/sliver

### Build sliver:
WORKDIR /go/src/github.com/bishopfox/sliver
ADD . /go/src/github.com/bishopfox/sliver/
RUN make clean-all
RUN make
RUN cp -vv sliver-server /opt/sliver-server

# STAGE: test
## Run unit tests against the compiled instance
## Use `--target test` in the docker build command to run this stage
FROM base as test

RUN apt-get update --fix-missing \
    && apt-get -y upgrade \
    && apt-get -y install \
    curl build-essential mingw-w64 binutils-mingw-w64 g++-mingw-w64 \
    && if [ "$(uname -m)" = "x86_64" ]; then apt-get -y install gcc-multilib; fi

RUN /opt/sliver-server unpack --force

### Run unit tests
RUN /go/src/github.com/bishopfox/sliver/go-tests.sh

# STAGE: production
## Final dockerized form of Sliver
FROM debian:bookworm-slim as production

### Install production packages
RUN apt-get update --fix-missing \
    && apt-get -y upgrade \
    && apt-get -y install \
    libxml2 libxml2-dev libxslt-dev locate gnupg \
    libreadline6-dev libcurl4-openssl-dev git-core \
    libssl-dev libyaml-dev openssl autoconf libtool \
    ncurses-dev bison curl xsel postgresql \
    postgresql-contrib postgresql-client libpq-dev \
    curl libapr1 libaprutil1 libsvn1 \
    libpcap-dev libsqlite3-dev libgmp3-dev \
    nasm \
    && if [ "$(uname -m)" = "x86_64" ]; then apt-get -y install gcc-multilib mingw-w64 binutils-mingw-w64 g++-mingw-w64; fi

### Install MSF for stager generation
RUN curl https://raw.githubusercontent.com/rapid7/metasploit-omnibus/master/config/templates/metasploit-framework-wrappers/msfupdate.erb > msfinstall \
    && chmod 755 msfinstall \
    && ./msfinstall \
    && mkdir -p ~/.msf4/ \
    && touch ~/.msf4/initial_setup_complete

### Cleanup unneeded packages
RUN apt-get remove -y curl gnupg \
    && apt-get autoremove -y \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

### Add sliver user
RUN groupadd -g 999 sliver \
    && useradd -r -u 999 -g sliver sliver \
    && mkdir -p /home/sliver/ \
    && chown -R sliver:sliver /home/sliver \
    && su -l sliver -c 'mkdir -p ~/.msf4/ && touch ~/.msf4/initial_setup_complete' \
    && su -l sliver -c 'mkdir -p ~/sliver/.sliver/logs' \
    && su -l sliver -c 'mkdir -p ~/sliver/.sliver-client'

### Copy compiled binary
COPY --from=base /opt/sliver-server  /opt/sliver-server

### Unpack Sliver:
USER sliver
RUN /opt/sliver-server unpack --force

WORKDIR /home/sliver/
VOLUME [ "/home/sliver/.sliver" ]
ENTRYPOINT [ "/opt/sliver-server" ]

# STAGE: production-slim (about 1Gb smaller)
### Slim production image, i.e. without MSF and assoicated libraries
### Still include GCC and MinGW for cross-platform generation
FROM debian:bookworm-slim as production-slim

### Install production packages
RUN apt-get update --fix-missing \
    && apt-get -y upgrade \
    && apt-get -y install \
    curl build-essential mingw-w64 binutils-mingw-w64 g++-mingw-w64 \
    && if [ "$(uname -m)" = "x86_64" ]; then apt-get -y install gcc-multilib; fi

### Cleanup unneeded packages
RUN apt-get autoremove -y \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

### Add sliver user
RUN groupadd -g 999 sliver \
    && useradd -r -u 999 -g sliver sliver \
    && mkdir -p /home/sliver/ \
    && chown -R sliver:sliver /home/sliver

### Copy compiled binary
COPY --from=base /opt/sliver-server  /opt/sliver-server

### Unpack Sliver:
USER sliver
RUN /opt/sliver-server unpack --force

WORKDIR /home/sliver/
VOLUME [ "/home/sliver/.sliver" ]
ENTRYPOINT [ "/opt/sliver-server" ]
