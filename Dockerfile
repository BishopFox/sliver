# STAGE: base
## Compiles Sliver for use
FROM golang:1.21.4 as base

#### Base packages
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

### Install testing packages
RUN apt-get update --fix-missing && apt-get -y install \
    libxml2 libxml2-dev libxslt-dev locate curl \
    libreadline6-dev libcurl4-openssl-dev git-core \
    libssl-dev libyaml-dev openssl autoconf libtool \
    ncurses-dev bison curl xsel postgresql \
    postgresql-contrib postgresql-client libpq-dev \
    libapr1 libaprutil1 libsvn1 \
    libpcap-dev libsqlite3-dev libgmp3-dev \
    mingw-w64 binutils-mingw-w64 g++-mingw-w64 \
    nasm gcc-multilib

### Install MSF for testing
RUN curl https://raw.githubusercontent.com/rapid7/metasploit-omnibus/master/config/templates/metasploit-framework-wrappers/msfupdate.erb > msfinstall \
    && chmod 755 msfinstall \
    && ./msfinstall
RUN mkdir -p ~/.msf4/ \
    && touch ~/.msf4/initial_setup_complete \
    && su -l sliver -c 'mkdir -p ~/.msf4/ && touch ~/.msf4/initial_setup_complete'

RUN /opt/sliver-server unpack --force 

### Run unit tests
RUN /go/src/github.com/bishopfox/sliver/go-tests.sh

# STAGE: production
## Final dockerized form of Sliver
FROM golang:1.21.4 as production

### Copy compiled binary
COPY --from=base /opt/sliver-server  /opt/sliver-server

### Add sliver user
RUN groupadd -g 999 sliver && useradd -r -u 999 -g sliver sliver
RUN mkdir -p /home/sliver/ && chown -R sliver:sliver /home/sliver

### Unpack Sliver:
RUN /opt/sliver-server unpack --force 

USER sliver
WORKDIR /home/sliver/
ENTRYPOINT [ "/opt/sliver-server" ]
