# client/assets

## Overview

Manages embedded client asset manifests such as operator profiles, settings, and armory catalogs. Handles serialization, bundling, and lookup of static client metadata. Core logic addresses aliases, armories, C2 profiles, and extensions within the assets package.

## Go Files

- `aliases.go` – Provides helpers for discovering, installing, and reading alias manifests shipped with the client.
- `armories.go` – Loads configured armory definitions and persists operator customizations.
- `assets.go` – Central entry point that exposes asset directories, lazy initialization, and filesystem utilities.
- `c2profiles.go` – Embeds default HTTP C2 profile templates and exposes lookup utilities.
- `config.go` – Tracks client configuration paths and manages persistence of runtime options.
- `extensions.go` – Handles embedded extension manifests and supports lookup by command name.
- `settings.go` – Stores default console settings and manages serialization to disk.
