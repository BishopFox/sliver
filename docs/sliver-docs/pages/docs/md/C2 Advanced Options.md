Advanced options, as the name suggests, are for advanced users that know what they're doing. Using these options isn't the best user experience and may result in broken or unexpected implant behavior. Only use these options if you understand what they do.

Advanced options are configured per-C2 endpoint and are passed as URL encoded parameters to the C2 URL in the `generate` command. For example:

```
generate --http http://example.com?driver=wininet
```

### HTTP C2 Advanced Options

- `net-timeout` - Network timeout value, parsed by `time.ParseDuration`.
- `tls-timeout` - TLS handshake timeout value, parsed by `time.ParseDuration`.
- `poll-timeout` - Poll timeout value, parsed by `time.ParseDuration`.
- `max-errors` - Max number of HTTP errors before failing (integer parsed by `strconv.Atoi`).
- `driver` - Manually specify the HTTP driver (string). On windows this value can be set to `wininet` to use the `wininet` HTTP library for C2 communication.
- `force-http` - Set to `true` to always force the use of plaintext HTTP.
- `disable-accept-header` - Set to `true` to disable the HTTP accept request header.
- `disable-upgrade-header` - Set to `true` to disable the HTTP upgrade request header.
- `proxy` - Manually specify HTTP proxy, this value is only used with Go HTTP driver, and the format should be one that is accepted by the Go HTTP library. You must specify a URI scheme with the hostname of the proxy. For example, `?proxy=http://myproxy.corp.com:8080`
- `proxy-username` - Specify a proxy username. Only valid with the Go HTTP driver.
- `proxy-password` - Specify the proxy password. Only valid with the Go HTTP driver.
- `ask-proxy-creds` - Set to `true` to ask the user for HTTP proxy credentials. Only valid when used with the `wininet` HTTP driver.
- `host-header` - Used for domain fronting.

### DNS C2 Advanced Options

- `timeout` - Network timeout value, parsed by `time.ParseDuration`.
- `retry-wait` - Upon query failure, wait this amount of time before retrying.
- `retry-count` - Upon query failure, retry this many times before failing.
- `workers-per-resolver` - Number of worker goroutines per functional DNS resolver.
- `max-errors` - Max number of query errors before failing (integer parsed by `strconv.Atoi`).
- `force-base32` - Always use Base 32 encoding.
- `force-resolv-conf` - Force the use of a provided `resolv.conf`. Note that you'll need to URL encode the newlines/etc of the file contents into the parameter.
- `resolvers` - Force the use of specific DNS resolvers. You can supply more than one by separating them with `+`, i.e. `...?resolvers=1.1.1.1+9.9.9.9`
