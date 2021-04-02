C2
===

The `c2` package contains the server-side command and control implementations. This code talks the `sliver` binary (client implementations are in `sliver/transports`). The currently supported procotols are mutual-TLS, HTTP(S), and DNS.

## mTLS - `tcp-mtls.go`

Mutual-TLS is the recommended default transport mechanism for Sliver implants, it provides robust security and throughput. mTLS does require the implant to route TCP traffic directly to the internet, which may not be desirable depending on the target environment.  mTLS connections are authenticated and encrypted using per-binary X.509 certificates that are embedded into the implant at compile-time (ECDSA). Certificates are signed using a per-server-instance ECDSA certificate authority that is generated the first time you execute the server binary. Only TLS v1.2 is supported, the only cipher suite enabled is `TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384`.

## HTTP(S) - `tcp-http.go`

Sliver makes little distinction between HTTP and HTTPS C2 communication. This is because the C2 protocol implements it's own sub-HTTP authenticated encryption scheme and does not rely upon the HTTPS connection's certificate for security or authenticity. Therefore, secure communication is possible over a HTTPS connections with valid or invalid certificates, as well as "unencrypted" HTTP. By default Sliver using [long-polling](https://en.wikipedia.org/wiki/Push_technology#Long_polling) over HTTP 1.1 to achieve near-realtime communication over HTTP(S). System proxy setting are respected when present, however if the implant fails to connect using the system proxy settings it will also attempt a direct connection.

Sliver will attempt the following HTTP(S) connections per C2 domain:
 * HTTPS via proxy
 * HTTPS without proxy
 * HTTP via proxy
 * HTTP without proxy

## DNS - `udp-dns.go`

DNS C2 is the slowest protocol but can offer various envasion properties. However, the current implementation is optimized for speed and stability, _not for stealth_. A stealthier version of the DNS implementation is planned for future versions of Sliver.