FROM golang:1.12

#
# IMPORTANT: This Dockerfile is used for testing, I do not recommend deploying
#            Sliver using this container configuration! However, if you do want
#            a Docker deployment this is probably a good place to start.
#

ENV PROTOC_VER 3.7.0
ENV RUBY_VER 2.6.2

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

#
# > Metasploit
#

WORKDIR /opt
RUN git clone --progress --verbose --depth 1 https://github.com/rapid7/metasploit-framework.git msf
WORKDIR /opt/msf

# RVM
RUN gpg --no-tty --keyserver hkp://keys.gnupg.net --recv-keys 409B6B1796C275462A1703113804BB82D39DC0E3 7D2BAF1CF37B13E2069D6956105BD0E739499BDB
RUN curl -L https://get.rvm.io | bash -s stable 
RUN /bin/bash -l -c "rvm requirements"
RUN /bin/bash -l -c "rvm install ${RUBY_VER}"
RUN /bin/bash -l -c "rvm use ${RUBY_VER} --default"
RUN /bin/bash -l -c "source /usr/local/rvm/scripts/rvm"
RUN /bin/bash -l -c "gem install bundler"
RUN /bin/bash -l -c "source /usr/local/rvm/scripts/rvm && which bundle"
RUN /bin/bash -l -c "which bundle"

# Get dependencies
RUN /bin/bash -l -c "BUNDLEJOBS=$(expr $(cat /proc/cpuinfo | grep vendor_id | wc -l) - 1)"
RUN /bin/bash -l -c "bundle config --global jobs $BUNDLEJOBS"
RUN /bin/bash -l -c "bundle install"

# Symlink tools to $PATH
RUN for i in `ls /opt/msf/tools/*/*`; do ln -s $i /usr/local/bin/; done
RUN ln -s /opt/msf/msf* /usr/local/bin


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

RUN mkdir -p /home/sliver/ && chown -R sliver:sliver /home/sliver
USER sliver
ENTRYPOINT [ "/opt/sliver-server" ]
