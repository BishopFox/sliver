
<div align="center">
  <br> <h1> Team </h1>

  <p>  Transform any Go program into a client of itself, remotely or locally.  </p>
  <p>  Use, manage teamservers and clients with code, with their CLI, or both.  </p>
</div>


<!-- Badges -->
<!-- Assuming the majority of them being written in Go, most of the badges below -->
<!-- Replace the repo name: :%s/reeflective\/template/reeflective\/repo/g -->

<p align="center">
  <a href="https://github.com/reeflective/team/actions/workflows/go.yml">
    <img src="https://github.com/reeflective/team/actions/workflows/go.yml/badge.svg?branch=main"
      alt="Github Actions (workflows)" />
  </a>

  <a href="https://github.com/reeflective/team">
    <img src="https://img.shields.io/github/go-mod/go-version/reeflective/team.svg"
      alt="Go module version" />
  </a>

  <a href="https://pkg.go.dev/github.com/reeflective/team">
    <img src="https://img.shields.io/badge/godoc-reference-blue.svg"
      alt="GoDoc reference" />
  </a>

  <a href="https://goreportcard.com/report/github.com/reeflective/team">
    <img src="https://goreportcard.com/badge/github.com/reeflective/team"
      alt="Go Report Card" />
  </a>

  <a href="https://codecov.io/gh/reeflective/team">
    <img src="https://codecov.io/gh/reeflective/team/branch/main/graph/badge.svg"
      alt="codecov" />
  </a>

  <a href="https://opensource.org/licenses/BSD-3-Clause">
    <img src="https://img.shields.io/badge/License-BSD_3--Clause-blue.svg"
      alt="License: BSD-3" />
  </a>
</p>


-----
## Summary

The client-server paradigm is an ubiquitous concept in computer science. Equally large and common is
the problem of building software that _collaborates_ easily with other peer programs. Although
writing collaborative software seems to be the daily task of many engineers around the world,
succeedingly and easily doing so in big programs as well as in smaller ones is not more easily done
than said. Difficulty still increases -and keeping in mind that humans use software and not the
inverse- when programs must enhance the capacity of humans to collaborate while not restricting the
number of ways they can do so, for small tasks as well as for complex ones.

The `reeflective/team` library provides a small toolset for arbitrary programs (and especially those
controlled in more or less interactive ways) to collaborate together by acting as clients and
servers of each others, as part of a team. Teams being made of players (humans _and_ their tools),
the library focuses on offering a toolset for "human teaming": that is, treating software tools that
are either _teamclients_ or _teamservers_ of others, within a defined -generally restricted- team of
users, which shall generally be strictly and securely authenticated.

The project originates from the refactoring of a security-oriented tool that used this approach to
clearly segregate client and server binary code (the former's not needing most of the latter's).
Besides, the large exposure of the said-tool to the CLI prompted the author of the
`reeflective/team` library to rethink how the notion of "collaborative programs" could be approached
and explored from different viewpoints: distinguishing between the tools' developers, and their
users. After having to reuse this core code for other projects, the idea appeared to extract the
relevant parts and to restructure and repackage them behind coherent interfaces (API and CLI).


-----
## Components & Terms

The result consists in 2 Go packages (`client` and `server`) for programs needing to act as:
- A **Team client**: a program, or one of its components, that needs to rely on a "remote" program peer
  to serve some functionality that is available to a team of users' tools. The program acting as a
  _teamclient_ may do so for things as simple as sending a message to the team, or as complicated as a
  compiler backend with which multiple client programs can send data to process and build.
- A **Team server**: The remote, server-side counterpart of the software teamclient. Again, the
  teamserver can be doing anything, from simply notifying users' teamclient connections to all the team
  all the way to handling very complex and resource-hungry tasks that can only be ran on a server host.

Throughout this library and its documentation, various words are repeatedly employed:
- _teamclient_ refers to either the client-specific toolset provided by this library
  (`team/client.Client` core type) or the software making use of this teamclient code.
- _teamserver_ refers to either the server-specific toolset provided to make a program serve its
  functionality remotely, or to the tools embedding this code in order to do so.
- _team tool/s_ might be used to refer to programs using either or all of the library components at
  large.

-----
## Principles, Constraints & Features

The library rests on several principles, constraints and ideas to fulfill its intended purpose:
- The library's sole aim is to **make most programs able to collaborate together** under the
  paradigm of team clients and team servers, and to do so while ensuring performance, coherence,
  ease of use and security of all processes and workflows involved. This, under the _separate
  viewpoints_ of tool development, enhancement and usage.
- Ensure a **working-by-default toolset**, assuming that the time spent on any tool's configuration
  is inversely proportional to its usage. Emphasis on this aspect should apply equally well to team
  tools' users and developers.
- Ensure the **full, secure and reliable authentication of all team clients and servers'
  interactions**, by using certificate-based communication encryption and user authentication, _aka_
  "zero-trust" model. Related and equally important, ensure the various team toolset interfaces
  provide for easy and secure usage of their host tools.
- **Accomodate for the needs of developers to use more specific components**, at times or at points,
  while not hampering on the working-by-default aspects of the team client/server toolset. Examples
  include replacing parts or all of the transport, RPC, loggers, database and filesystem
  backends.
- To that effect, the library offers **different interfaces to its functionality**: an API (Go code)
  provides developers a working-by-default, simple and powerful way to instruct their software how 
  to collaborate with peers, and a CLI, for users to operate their team tools, manage their related 
  team configurations with ease, with a featured command-line tree to embed anywhere.
- Ensure that team client/server functionality can be **easily integrated in automated workflows**: 
  this is done by offering clear code/execution paths and behaviors, for both users and developers,
  and by providing commands and functions to ease deployment of said tools.

-----
## CLI (Users)

The following extracts assume a program binary named `teamserver`, which is simply the root command
of the server-side team code. In this case therefore, the binary program only purpose its to be a
teamserver, with no application-specific logic, (and is therefore quite useless on its own):
```
$ teamserver
Manage the application server-side teamserver and users

Usage:
  teamserver [command]

teamserver control
  client      Client-only teamserver commands (import configs, show users, etc)
  close       Close a listener and remove it from persistent ones if it's one
  daemon      Start the teamserver in daemon mode (blocking)
  listen      Start a teamserver listener (non-blocking)
  status      Show the status of the teamserver (listeners, configurations, health...)
  systemd     Print a systemd unit file for the application teamserver, with options

user management
  delete      Remove a user from the teamserver, and revoke all its current tokens
  export      Export a Certificate Authority file containing the teamserver users
  import      Import a certificate Authority file containing teamserver users
  user        Create a user for this teamserver and generate its client configuration file
```

In this example, this program comes with a client-only binary counterpart, `teamclient`. The latter 
does not include any team server-specific code, and has therefore a much smaller command set:
```
$ teamclient
Client-only teamserver commands (import configs, show users, etc)

Usage:
  teamclient [command]

Available Commands:
  import      Import a teamserver client configuration file for teamserver
  users       Display a table of teamserver users and their status
  version     Print teamserver client version
```

With these example binaries at hand, below are some examples of workflows.
Starting with the `teamserver` binary (which might be under access/control of a team admin):
``` bash
# 1 - Generate a user for a local teamserver, and import users from a file.
teamserver user --name Michael --host localhost
teamserver import ~/.other_app/teamserver/certs/other_app_user-ca-cert.teamserver.pem

# 2 - Start some teamserver listeners, then start the teamserver daemon (blocking).
# Use the application-defined default port in the first call, and instruct the server
# to start the listeners automatically when used in daemon mode with --persistent.
teamserver listen --host localhost --persistent 
teamserver listen --host 172.10.0.10 --port 32333 --persistent
teamserver status                                                   # Prints the saved listeners, configured loggers, databases, etc.
teamserver daemon --host localhost --port 31337                     # Blocking: serves all persistent listeners and a main one at localhost:31337

# 3 - Export and enable a systemd service configuration for the teamserver.
teamserver systemd                                                  # Use default host, port and listener stacks. 
teamserver systemd --host localhost --binpath /path/to/teamserver   # Specify binary path.
teamserver systemd --user --save ~/teamserver.service               # Print to file instead of stdout.

# 4 - Import the "remote" administrator configuration for (1), and use it.
teamserver client import ~/Michael_localhost.teamclient.cfg 
teamserver client version                                   # Print the client and the server version information.
teamserver client users                                     # Print all users registered to the teamserver and their status.

# 5 - Quality of life
teamserver _carapace <shell> # Source detailed the completion engine for the teamserver.
```

Continuing the `teamclient` binary (which is available to all users' tool in the team):
```bash
# Example 1 - Import a remote teamserver configuration file given by a team administrator.
teamclient import ~/Michael_localhost.teamclient.cfg

# Example 2 - Query the server for its information.
teamclient users
teamclient version
```

-----
## API (Developers)

The teamclient and teamserver APIs are designed with several things in mind as well:
- While users are free to use their tools teamclients/servers within the bounds of the provided
  command-line interface tree (`teamserver` and `teamclient` commands), the developers using the 
  library have access to a slightly larger API, especially with regards to "selection strategies"
  (grossly, the way tools' teamclients choose their remote teamservers before connecting to them).
  This is equivalent of saying that tools developers should have identified 70% of all different
  scenarios/valid operation mode for their tools, and program their teamclients accounting for this,
  but let the users decide of the remaining 30% when using the tools teamclient/server CLI commands.
- The library makes it easy to embed a teamclient or a teamserver in existing codebases, or easy to 
  include it in the ones who will need it in the future. In any case, importing and using a default
  teamclient/teamserver should fit into a couple of function calls at most.
- To provide a documented code base, with a concise naming and programming model which allows equally
  well to use default teamclient backends or to partially/fully reimplement different layers.

Below is the simplest, shortest example of the above's `teamserver` binary `main()` function:
```go
// Generate a teamserver, without any specific transport/RPC backend.
// Such backends are only needed when the teamserver serves remote clients.
teamserver, err := server.New("teamserver")

// Generate a tree of server-side commands: this tree also has client-only
// commands as a subcommand "client" of the "teamserver" command root here.
serverCmds := commands.Generate(teamserver, teamserver.Self())

// Run the teamserver CLI.
serverCmds.Execute()
```

Another slightly more complex example, involving a gRPC transport/RPC backend:
```go
// The examples directory has a default teamserver listener backend.
gTeamserver := grpc.NewListener()

// Create a new teamserver, register the gRPC backend with it.
// All gRPC teamclients will be able to connect to our teamserver.
teamserver, err := server.New("teamserver", server.WithListener(gTeamserver))

// Since our teamserver offers its functionality through a gRPC layer,
// our teamclients must have the corresponding client-side RPC client backend.
// Create an in-memory gRPC teamclient backend for the server to serve itself.
gTeamclient := grpc.NewClientFrom(gTeamserver)

// Create a new teamclient, registering the gRPC backend to it.
teamclient := teamserver.Self(client.WithDialer(gTeamclient))

// Generate the commands for the teamserver.
serverCmds := commands.Generate(teamserver, teamclient)

// Run any of the commands.
serverCmds.Execute()
```

Some additional and preliminary/example notes about the codebase:
- All errors returned by the API are always logged before return (with configured log behavior).
- Interactions with the filesystem restrained until they need to happen.
- The default database is a pure Go file-based sqlite db, which can be configured to run in memory.
- Unless absolutely needed or specified otherwise, return all critical errors instead of log
  fatal/panicking (exception made of the certificate infrastructure which absolutely needs to work
  for security reasons).
- Exception made of the `teamserver daemon` command related `server.ServeDaemon` function, all API
  functions and interface methods are non-blocking. Mentions of this are found throughout the
  code documentation when needed.
- Loggers offered by the teamclient/server cores are never nil, and will log to both stdout (above
  warning level) and to default files (above info level) if no custom logger is passed to them.
  If such a custom logger is given, team clients/servers won't log to stdout or their default files.

Please see the [example](https://github.com/reeflective/team/tree/main/example) directory for all client/server entrypoint examples.

-----
## Documentation

- Go code documentation is available at the [Godoc website](https://pkg.go.dev/github.com/reeflective/team).
- Client and server documentation can be found in the [directories section](https://pkg.go.dev/github.com/reeflective/team#section-directories) of the Go documentation.
- The `example/` subdirectories also include documentation for their own code, and should provide
a good introduction to this library usage. 

-----
## Differences with the Hashicorp Go plugin system

At first glance, different and not much related to our current topic is the equally large problem of
dynamic code loading and execution for arbitrary programs. In the spectrum of major programming
languages, various approaches have been taken to tackle the dynamic linking, loading and execution
problem, with interpreted languages offering the most common solutioning approach to this.

The Go language (and many other compiled languages that do not encourage dynamic linking for that
matter) has to deal with the problem through other means, the first of which simply being the
adoption of different architectural designs in the first place (eg. "microservices"). Another path
has been the "plugin system" for emulating the dynamic workflows of interpreted languages, of which
the most widely used attempt being the [Hashicorp plugin
system](https://github.com/hashicorp/go-plugin), which entirely rests on an (g)RPC backend.

Consequently, differences and similarities can be resumed as follows:
- The **Hashicorp plugins only support "remote" plugins** in that each plugin must be a different
  binary. Although those plugins seem to be executed "in-memory", they are not. On the contrary,
  the `reeflective/team` clients and servers can (should, and will) be used both in memory and
  remotely (here remotely means as a distinct subprocess: actual network location is irrelevant).
- The purpose of the `reeflective/team` library is **not** to emulate dynamic code execution behavior.
  Rather, its intent is to make programs that should or might be better used as servers to several
  clients to act as such easily and securely in many different scenarios.
- The **Hashicorp plugins are by essence restrained to an API problem**, and while the `team` library
  is equally (but not mandatorily or exclusively) about interactive usage of arbitrary programs.
- **The Hashicorp plugin relies mandatorily (since it's built on) a gRPC transport backend**. While
  gRPC is a very sensible choice for many reasons (and is therefore used for the default example
  backend in `example/transports/`), the `team` library does not force library users to use a given
  transport/RPC backend, nor even to use one. Again, this would be beyond the library scope, but
  what is in scope is the capacity of this library to interface with or use different transports.
- Finally, the Hashicorp plugins are not aware of any concept of users as they are considered by
  the team library, although both use certificate-based connections. However, `team` promotes and
  makes easy to use mutually authenticated (Mutual TLS) connections (see the default gRPC example 
  backend). Related to this, teamservers integrate loggers and a database to store working data.

-----
## Status

The Command-Line and Application-Programming Interfaces of this library are unlikely to change
much in the future, and should be considered mostly stable. These might grow a little bit, but
will not shrink, as they been already designed to be as minimal as they could be.

In particular, `client.Options` and `server.Options` APIs might grow, so that new features/behaviors
can be integrated without the need for the teamclients and teamservers types APIs to change.

The section **Possible Enhancements** below includes 9 points, which should grossly be equal
to 9 minor releases (`0.1.0`, `0.2.0`, `0.3.0`, etc...), ending up in `v1.0.0`.

- Please open a PR or an issue if you face any bug, it will be promptly resolved.
- New features and/or PRs are welcome if they are likely to be useful to most users.

-----
## Possible enhancements

The list below is not an indication on the roadmap of this repository, but should be viewed as
things the author of this library would be very glad to merge contributions for, or get ideas. 
This teamserver library aims to remain small, with a precise behavior and role.
Overall, contributions and ideas should revolve around strenghening its core/transport code
or around enhancing its interoperability with as much Go code/programs as possible.

- [ ] Use viper for configs.
- [ ] Use afero filesystem.
- [ ] Add support for encrypted sqlite by default.
- [ ] Encrypt in-memory channels, or add option for it.
- [ ] Simpler/different listener/dialer backend interfaces, if it appears needed.
- [ ] Abstract away the client-side authentication, for pluggable auth/credential models.
- [ ] Replace logrus entirely and restructure behind a single package used by both client/server.
- [ ] Review/refine/strenghen the dialer/listener init/close/start process, if it appears needed.
- [ ] `teamclient update` downloads latest version of the server binary + method to `team.Client` for it.

