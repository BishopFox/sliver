FROM golang:1.12

#
# IMPORTANT: This Dockerfile is used for testing, I do not recommend deploying
#            Sliver using this container configuration! However, if you do want
#            a Docker deployment this is probably a good place to start.
#

ENV PROTOC_VER 3.7.1

# Base packages
RUN apt-get update --fix-missing && apt-get -y install \
  git build-essential zlib1g zlib1g-dev \
  libxml2 libxml2-dev libxslt-dev locate curl \
  libreadline6-dev libcurl4-openssl-dev git-core \
  libssl-dev libyaml-dev openssl autoconf libtool \
  ncurses-dev bison curl wget xsel postgresql \
  postgresql-contrib postgresql-client libpq-dev \
  libapr1 libaprutil1 libsvn1 \
  libpcap-dev libsqlite3-dev libgmp3-dev \
  zip unzip mingw-w64 binutils-mingw-w64 g++-mingw-w64 \
  nasm

#
# > User
#
RUN groupadd -g 999 sliver && useradd -r -u 999 -g sliver sliver
RUN mkdir -p /home/sliver/ && chown -R sliver:sliver /home/sliver

#
# > Metasploit
#

RUN curl https://raw.githubusercontent.com/rapid7/metasploit-omnibus/master/config/templates/metasploit-framework-wrappers/msfupdate.erb > msfinstall && \
  chmod 755 msfinstall && \
  ./msfinstall
RUN mkdir -p ~/.msf4/ && touch ~/.msf4/initial_setup_complete && \
  su -l sliver -c 'mkdir -p ~/.msf4/ && touch ~/.msf4/initial_setup_complete'

#
# > Sliver
#

# protoc
WORKDIR /tmp
RUN wget https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VER}/protoc-${PROTOC_VER}-linux-x86_64.zip \
    && unzip protoc-${PROTOC_VER}-linux-x86_64.zip \
    && cp -vv ./bin/protoc /usr/local/bin

# go get utils
RUN go get github.com/golang/protobuf/protoc-gen-go
RUN go get -u github.com/gobuffalo/packr/packr

# install dep
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

# assets
WORKDIR /go/src/github.com/bishopfox/sliver
ADD ./go-assets.sh /go/src/github.com/bishopfox/sliver/go-assets.sh
RUN ./go-assets.sh

# compile - we have to run dep after copying the code over or it bitches
ADD . /go/src/github.com/bishopfox/sliver/
RUN make static-linux && cp -vv sliver-server /opt/sliver-server

RUN /opt/sliver-server -unpack && /go/src/github.com/bishopfox/sliver/go-tests.sh
RUN make clean \
    && rm -rf /go/src/* \
    && rm -rf /root/.sliver

USER sliver
ENTRYPOINT [ "/opt/sliver-server" ]
