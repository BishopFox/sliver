FROM golang:1

ENV PROTOC_VER 3.7.1

RUN go get -u -v \
    github.com/mdempsky/gocode \
    github.com/uudashr/gopkgs/cmd/gopkgs \
    github.com/ramya-rao-a/go-outline \
    github.com/acroca/go-symbols \
    golang.org/x/tools/cmd/guru \
    golang.org/x/tools/cmd/gorename \
    github.com/rogpeppe/godef \
    github.com/zmb3/gogetdoc \
    github.com/sqs/goreturns \
    golang.org/x/tools/cmd/goimports \
    golang.org/x/lint/golint \
    github.com/alecthomas/gometalinter \
    honnef.co/go/tools/... \
    github.com/golangci/golangci-lint/cmd/golangci-lint \
    github.com/mgechev/revive \
    github.com/golang/protobuf/protoc-gen-go \
    github.com/gobuffalo/packr/packr \
    github.com/derekparker/delve/cmd/dlv 2>&1

# gocode-gomod
RUN go get -x -d github.com/stamblerre/gocode \
    && go build -o gocode-gomod github.com/stamblerre/gocode \
    && mv gocode-gomod $GOPATH/bin/

# Install git, process tools, lsb-release (common in install instructions for CLIs)
RUN apt-get update && apt-get -y install git zsh procps lsb-release build-essential zlib1g zlib1g-dev \
    libxml2 libxml2-dev libxslt-dev locate curl \
    libreadline6-dev libcurl4-openssl-dev git-core \
    libssl-dev libyaml-dev openssl autoconf libtool \
    ncurses-dev bison curl wget xsel postgresql \
    postgresql-contrib postgresql-client libpq-dev \
    libapr1 libaprutil1 libsvn1 \
    libpcap-dev libsqlite3-dev libgmp3-dev \
    zip unzip mingw-w64 binutils-mingw-w64 g++-mingw-w64 \
    nasm

# Clean up
RUN apt-get autoremove -y \
    && apt-get clean -y \
    && rm -rf /var/lib/apt/lists/*

# Metasploit
RUN curl https://raw.githubusercontent.com/rapid7/metasploit-omnibus/master/config/templates/metasploit-framework-wrappers/msfupdate.erb > msfinstall && \
    chmod 755 msfinstall && \
    ./msfinstall
RUN mkdir -p ~/.msf4/ && touch ~/.msf4/initial_setup_complete && \
    mkdir -p ~/.msf4/ && touch ~/.msf4/initial_setup_complete

# protoc
WORKDIR /tmp
RUN wget https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VER}/protoc-${PROTOC_VER}-linux-x86_64.zip \
    && unzip protoc-${PROTOC_VER}-linux-x86_64.zip \
    && cp -vv ./bin/protoc /usr/local/bin

# install dep
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
RUN mkdir -p $GOPATH/src/github.com/bishopfox/sliver
WORKDIR $GOPATH/src/github.com/bishopfox/sliver

# Set the default shell to bash instead of sh
ENV SHELL /bin/bash