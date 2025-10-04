# client/command/exec

## Overview

Implements the 'exec' command group for the Sliver client console. Handlers map Cobra invocations to exec workflows such as execute assembly, execute shellcode, execute, and migrate.

## Go Files

- `commands.go` – Registers the exec command family and binds shared flags across execution techniques.
- `execute-assembly.go` – Uploads .NET assemblies and triggers in-memory execution, handling results and loot capture.
- `execute-shellcode.go` – Executes raw shellcode via RPC, supporting interactive injection and task result printing.
- `execute.go` – Runs arbitrary commands, streams combined stdout/stderr, and persists output as loot when requested.
- `migrate.go` – Implements process migration, moving implants into new host processes and reporting completion.
- `msf-inject.go` – Coordinates Metasploit payload injection tasks and prints remote execution responses.
- `msf.go` – Launches Metasploit jobs on targets and tracks their beacon callbacks.
- `psexec.go` – Provides PsExec-style remote service deployment using generated binaries and random filenames.
- `sideload.go` – Executes unmanaged DLL exports in a remote process and summarizes task output.
- `spawndll.go` – Spawns new processes with DLL injection, capturing returned handles and statuses.
- `ssh.go` – Runs remote commands over SSH, sourcing credentials from loot and presenting command output.
