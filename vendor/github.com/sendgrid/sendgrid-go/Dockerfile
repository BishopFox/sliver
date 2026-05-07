FROM golang:1.21.11

ENV GO111MODULE 'off'

COPY prism/prism/nginx/cert.crt /usr/local/share/ca-certificates/cert.crt
RUN update-ca-certificates

WORKDIR /go/src/github.com/sendgrid/sendgrid-go
COPY . .

RUN make install

# Use the last version of testify that works for older go versions, and then
# re-install to update dependencies.
RUN (cd /go/src/github.com/stretchr/testify && git checkout v1.6.0)
RUN make install
