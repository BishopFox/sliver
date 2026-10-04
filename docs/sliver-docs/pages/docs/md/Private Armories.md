[Private Armory](https://github.com/sliverarmory/private-armory) is a standalone service for hosting your own Sliver aliases, extensions, and bundles. Each Sliver client connects to it independently; installed packages are still local to that client, as described in [Armory](/docs?name=Armory).

The upstream wiki sometimes calls the project `external-armory`. The current repository is `private-armory`, and its executable is `armory-server`.

## Hosting

The service can run as a standalone executable or in a container. Follow the upstream [Setup](https://github.com/sliverarmory/private-armory/wiki/Setup) or [Docker](https://github.com/sliverarmory/private-armory/wiki/Docker) guide for deployment details.

- Configuration is stored in `<armory-root>/config.json`; restart the service after editing it.
- The supplied Compose configuration publishes port `8888` and persists `./armory-data` at `/data/armory-data`.
- Review TLS and authentication settings before deployment. The supplied `armory.env` disables TLS; HTTPS requires a certificate and private key. Verify the effective authentication settings for the version you deploy rather than assuming the example enables access control.
- Initial setup displays client connection information and a separate administration token. Give clients the client token, never the administration token or a signing private key.

See [Environment Variables](https://github.com/sliverarmory/private-armory/wiki/Environment-Variables) for configuration options. Older wiki JSON examples may differ from the current service configuration.

## Storage and Signing

The [storage providers](https://github.com/sliverarmory/private-armory/wiki/Storage-Providers) support a writable local filesystem or an S3 general-purpose bucket. Preserve the configured storage across restarts and container replacements.

Package and index signatures use Minisign. The [signing key providers](https://github.com/sliverarmory/private-armory/wiki/Package-Signing-Key-Providers) are:

| Provider | Key location |
| --- | --- |
| `local` | A private key file in the armory root. |
| `aws` | AWS Secrets Manager. |
| `vault` | HashiCorp Vault. |
| `external` | A separate signing process; the serving application holds the public key. |

With `external`, signing happens outside the service and package administration through its API is disabled. See [Standalone Signing](https://github.com/sliverarmory/private-armory/wiki/Standalone-Signing) and [Package Administration](https://github.com/sliverarmory/private-armory/wiki/Package-Administration) for the respective workflows.

Obtain the Minisign public key through a trusted channel. It establishes trust in the index and package signing metadata; it is separate from the TLS certificate. A valid signature does not establish that a package is safe to use.

## Client Configuration

Obtain the index URL, Minisign public key, and client authorization token from your armory administrator. The service's index URL ends in `/armory/index`; use the configured hostname, port, and HTTPS endpoint.

```text
sliver > armory add private --url https://armory.example.com:8888/armory/index --pubkey <MINISIGN_PUBLIC_KEY> --auth <CLIENT_TOKEN>
sliver > armory info
```

Replace the placeholders with your armory's values. `--auth` supplies the complete `Authorization` header value; Sliver does not add a `Bearer` prefix. Adding a source validates its index before saving it.

Configuration is saved to `~/.sliver-client/armories.json` by default, or under `SLIVER_CLIENT_ROOT_DIR` when set. Authorization values are stored in plaintext, and detailed `armory info <name>` output can display them; protect the file and redact credentials before sharing output.

## Managing Sources

| Command | Effect |
| --- | --- |
| `armory info` | List configured sources. |
| `armory refresh` | Reload saved source configuration and refresh cached metadata. |
| `armory disable private` / `armory enable private` | Disable or enable the private source. |
| `armory rm private` | Remove its source registration. |
| `armory save` | Save the current source configuration. |

Add and modify operations save by default. Changes made with `--no-save` are temporary and are lost on refresh or restart unless saved first. Disabling or removing a source does not uninstall packages already installed from it.
