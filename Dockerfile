FROM golang:1.12

ENV PROTOC_VER 3.6.1

# os packages
RUN apt-get update && \
    apt-get -y install wget git zip unzip build-essential

# protoc
WORKDIR /tmp
RUN wget https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VER}/protoc-${PROTOC_VER}-linux-x86_64.zip \
    && unzip protoc-${PROTOC_VER}-linux-x86_64.zip \
    && cp -vv ./bin/protoc /usr/local/bin

# go utils
RUN go get github.com/golang/protobuf/protoc-gen-go
RUN go get -u github.com/gobuffalo/packr/packr

# dep
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

# assets
WORKDIR /go/src/sliver
ADD ./go-assets.sh /go/src/sliver/go-assets.sh
RUN ./go-assets.sh

# compile - we have to run dep after copying the code over or it bitches
ADD . /go/src/sliver/
RUN dep ensure
RUN make static-linux

RUN /go/src/sliver/sliver-server -unpack

ENTRYPOINT [ "/go/src/sliver/go-tests.sh" ]
