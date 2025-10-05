# client/command/privilege

## Overview

Implements the 'privilege' command group for the Sliver client console. Handlers map Cobra invocations to privilege workflows such as getprivs, getsystem, impersonate, and make token.

## Go Files

- `commands.go` – Registers privilege escalation commands and binds platform restrictions.
- `getprivs.go` – Queries and prints token privileges for the current user context.
- `getsystem.go` – Attempts to obtain SYSTEM privileges via built-in escalation techniques.
- `impersonate.go` – Handles impersonation of other logon sessions using available tokens.
- `make-token.go` – Creates new logon tokens with supplied credentials for the implant.
- `rev2self.go` – Reverts impersonation back to the implant's original security context.
- `runas.go` – Runs commands under alternate credentials or integrity levels through runas-like workflows.
