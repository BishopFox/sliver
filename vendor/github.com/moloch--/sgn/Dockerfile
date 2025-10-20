FROM golang:latest as builder

RUN apt-get update && apt-get -y install \
    build-essential \    
    cmake \
    g++-multilib \
    gcc-multilib \
    git \
    libcapstone-dev \
    python3 \
    time
WORKDIR /root/
RUN git clone https://github.com/EgeBalci/keystone
RUN mkdir keystone/build
WORKDIR /root/keystone/build

RUN ../make-lib.sh
RUN cmake -DCMAKE_BUILD_TYPE=Release -DBUILD_SHARED_LIBS=OFF -DLLVM_TARGETS_TO_BUILD="AArch64;X86" -G "Unix Makefiles" ..
RUN make -j8
RUN make install && ldconfig

WORKDIR /root
RUN git clone https://github.com/egebalci/sgn
WORKDIR /root/sgn
RUN go build -o /root/bin/sgn -ldflags '-w -s -extldflags -static' -trimpath main.go

FROM scratch
COPY --from=builder /root/bin/sgn /sgn
ENTRYPOINT ["/sgn"]
