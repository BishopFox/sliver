# server/loot

## Overview

Operator loot storage and retrieval helpers managed by the server. Handles filesystem organization, metadata, and access control. Key routines cover backend within the loot subsystem.

## Go Files

- `backend.go` – Defines loot storage backends and filesystem layout.
- `loot.go` – Implements loot CRUD operations and metadata handling.
- `loot_test.go` *(tests)* – Tests loot backend behaviors.
