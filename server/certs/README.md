certs
======

X.509 certificate generation and management code. We use four seperate certificate chains (4 CAs):

 * `SliverCA` - Used to encrypt and authenticated client-side C2 channels between the server and the Slivers. Uses both ECC and RSA certificates depending on the use case.
 * `OperatorCA` (was `ClientCA`) - Used to sign certs that authenticate and encrypt the mutual TLS connection between the operator and the server.
 * `ServerCA` - Used to secure server-side C2, the ServerCA public key is embedded into the sliver binaries.
 * `HTTPSCA` - Used to generate self-signed HTTPS certificates (that are not used to encrypt C2 data)

Certificates are all stored CA-specific Badger databases managed by the `db` package. The key is the common name of the certificate and the value is a JSON object (i.e. `CertificateKeyPair`) that contains the key type (RSA or ECC), certifcate, and private key.

### ACME

The package can also interact with Let's Encrypt (ACME) services to generate certificates that are trusted in the browser (alternative to `HTTPSCA`). These certificates are used with the HTTPS servers/listeners, but not used to encrypt any C2.
