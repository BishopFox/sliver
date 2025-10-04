# client/command/websites

## Overview

Implements the 'websites' command group for the Sliver client console. Handlers map Cobra invocations to websites workflows such as websites ADD content, websites RM content, websites RM, and websites update content.

## Go Files

- `commands.go` – Registers website management commands for hosted phishing content.
- `websites-add-content.go` – Uploads new site content (files or strings) to the server.
- `websites-rm-content.go` – Removes specific content entries from a hosted site.
- `websites-rm.go` – Deletes entire website definitions from the server.
- `websites-update-content.go` – Replaces or updates existing hosted content assets.
- `websites.go` – Lists configured websites and prints their metadata.
