# client/command/certificates

## Overview

Implements the 'certificates' command group for the Sliver client console.

## Go Files

- `authorities.go` – Retrieves certificate authority metadata from the server and prints it in a table.
- `completions.go` – Provides tab completion helpers for certificate common names.
- `certificates.go` – Retrieves certificate metadata from the server, formats expiration status, and prints results to the console.
- `commands.go` – Registers the certificates command and binds the flags needed to query certificate inventories.
