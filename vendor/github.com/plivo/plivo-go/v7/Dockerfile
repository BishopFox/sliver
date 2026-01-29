FROM golang:1.17-alpine

WORKDIR /usr/src/app
RUN apk update && apk add git vim bash gcc musl-dev make

ENV PATH $PATH:/usr/local/go/bin

# Copy setup script
COPY setup_sdk.sh /usr/src/app/
RUN chmod a+x /usr/src/app/setup_sdk.sh

ENTRYPOINT [ "/usr/src/app/setup_sdk.sh" ]
