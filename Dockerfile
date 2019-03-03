FROM golang:1.12

RUN apt-get update && \
    apt-get -y install wget git zip unzip build-essential autoconf libtool

# protoc
WORKDIR /tmp
RUN git clone https://github.com/google/protobuf.git && \
    cd protobuf && \
    ./autogen.sh && \
    ./configure && \
    make && \
    make install && \
    ldconfig && \
    make clean && \
    cd .. && \
    rm -r protobuf
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
RUN ./go-tests.sh

ENTRYPOINT [ "/go/src/sliver/sliver-server" ]
