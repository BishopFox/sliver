
<!-- This is a README template used as a basis for most repositories hosted here. -->
<!-- This repository has two branches: -->
<!-- main       - Contains the README and other default files -->

<!-- Documentation Setup/Pull/Edit/Push -->
<!-- ----------------------------------------- -->

<!-- We include the Github's wiki repository as a subtree of the project's repository. -->
<!-- (Using this [link](https://gist.github.com/SKempin/b7857a6ff6bddb05717cc17a44091202)) -->
<!-- Please check the raw version of this README, contains comments with appropriate -->
<!-- commands for pushing/pulling the documentation subtree. -->

<!-- Add the initial wiki in a subtree (normally ':branch' should be 'main'): -->
<!-- git subtree add --prefix docs/ https://github.com/:user/:repo.wiki.git :branch --squash -->

<!-- Pull latest changes in the wiki -->
<!-- `git subtree pull --prefix docs/ https://github.com/:user/:repo.git master --squash` -->

<!-- Push your changes to the wiki -->
<!-- `git subtree push --prefix docs/ https://github.com/:user/:repo.git :branch` -->

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

-----

## CLI examples (users)

-----

## API examples (developers)

-----

## Documentation

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

