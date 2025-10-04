# client/command/armory

## Overview

Implements the 'armory' command group for the Sliver client console. Handlers map Cobra invocations to armory workflows such as install, manage, parsers, and search.

## Go Files

- `armory.go` – Maintains the armory cache lifecycle, fetching package metadata, verifying signatures, and presenting bundle listings.
- `armory_test.go` *(tests)* – Placeholder for armory command tests; currently used to keep the package's test harness wired up.
- `commands.go` – Declares the Cobra command hierarchy for armory actions and binds shared flag/completion plumbing.
- `install.go` – Resolves dependencies and installs aliases or extensions from cached metadata, prompting when overwriting existing packages.
- `manage.go` – Implements management helpers for viewing, adding, verifying, and persisting armory configurations.
- `parsers.go` – Provides parsers and HTTP helpers for retrieving armory indexes and packages, including Minisign verification.
- `search.go` – Implements the regex-driven search command that filters cached armory packages.
- `update.go` – Calculates available package updates, presents options to the user, and orchestrates update installation.
