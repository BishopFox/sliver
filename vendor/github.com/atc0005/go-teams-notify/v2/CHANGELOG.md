# Changelog

## Overview

All notable changes to this project will be documented in this file.

The format is based on [Keep a
Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Please [open an issue](https://github.com/atc0005/go-teams-notify/issues) for any
deviations that you spot; I'm still learning!.

## Types of changes

The following types of changes will be recorded in this file:

- `Added` for new features.
- `Changed` for changes in existing functionality.
- `Deprecated` for soon-to-be removed features.
- `Removed` for now removed features.
- `Fixed` for any bug fixes.
- `Security` in case of vulnerabilities.

## [Unreleased]

- placeholder

## [v2.14.0] - 2025-11-16

### Changed

- (GH-302) Go Dependency: Bump github.com/stretchr/testify from 1.9.0 to 1.10.0

### Fixed

- (GH-311) fix: adjust WorkflowURLBaseDomain for both new and old urls
  - credit: [@calindima](https://github.com/calindima)

## [v2.13.0] - 2024-09-08

### Added

- (GH-293) Add MSTeams CodeBlock element
  - credit: [@MichaelUrman](https://github.com/MichaelUrman)
- (GH-298) Update documentation for CodeBlock element

## [v2.12.0] - 2024-08-16

### Added

- (GH-291) Expose `TeamsMessage` interface to support mocking

## [v2.11.0] - 2024-08-02

### Added

- (GH-275) Add initial support for Workflow connectors

### Changed

#### Dependency Updates

- (GH-259) Go Dependency: Bump github.com/stretchr/testify from 1.8.4 to 1.9.0

#### Other

- (GH-272) Documentation refresh for O365 & Workflow connectors

### Fixed

- (GH-261) Remove inactive maligned linter
- (GH-274) Fix validation for `Action.Type` field
- (GH-283) Update CodeQL workflow to run on dev branch PRs

## [v2.10.0] - 2024-02-22

### Added

- (GH-255) Add `IsSublte` and `HorizontalAlignment` to `Element`
  - credit: [@codello](https://github.com/codello)

### Changed

#### Dependency Updates

- (GH-256) Update Dependabot PR prefixes

## [v2.9.0] - 2024-01-25

### Added

- (GH-241) Add proxy server examples
- (GH-251) Initial support for toggling visibility

### Changed

#### Dependency Updates

- (GH-238) ghaw: bump actions/checkout from 3 to 4
- (GH-248) ghaw: bump github/codeql-action from 2 to 3
- (GH-236) Update Dependabot config to monitor both branches

#### Other

- (GH-244) Update Go Doc comment formatting

## [v2.8.0] - 2023-07-21

### Added

- `Adaptive Card` format
  - (GH-205) Ability to create a Table in AdaptiveCard
- CI
  - (GH-232) Add initial automated release notes config
  - (GH-233) Add initial automated release build workflow

### Changed

- Dependencies
  - `stretchr/testify`
    - `v1.8.2` to `v1.8.4`
- CI
  - (GH-226) Add `quick` Makefile recipe (alias)
  - (GH-225) Update vuln analysis GHAW to remove on.push hook
  - (GH-230) Disable unsupported build opts in monthly workflow

### Fixed

- CI
  - (GH-229) Restore local CodeQL workflow

## [v2.7.1] - 2023-06-09

### Changed

- Dependencies
  - `github.com/stretchr/testify`
    - `v1.8.1` to `v1.8.2`
- CI
  - (GH-198) Add Go Module Validation, Dependency Updates jobs
  - (GH-200) Drop `Push Validation` workflow
  - (GH-201) Rework workflow scheduling
  - (GH-203) Remove `Push Validation` workflow status badge
  - (GH-207) Update vuln analysis GHAW to use on.push hook
- `Adaptive Card` format
  - (GH-206) Update `AdaptiveCardMaxVersion` to 1.5
  - (GH-216) Refactor `TopLevelCard.Validate`
- Other
  - (GH-212) Update `InList`, `InListIfFieldValNotEmpty` validators

### Fixed

- (GH-208) Validation of `(adaptivecard.Attachment).Content` is missing

## [v2.7.0] - 2022-12-12

### Added

- (GH-134) Allow setting user agent, fallback to project-specific default
  value
- (GH-135) Allow overriding default `http.Client`
- (GH-157) Add `Adaptive Card` message format support
  - see also discussion from GH-127, including feedback from
    [@ghokun](https://github.com/ghokun)
- (GH-169) Added YAML en(de)coding support to `MessageCard`
  - credit: [@pcanilho](https://github.com/pcanilho)

### Changed

- Dependencies
  - `github.com/stretchr/testify`
    - `v1.7.0` to `v1.8.1`
- (GH-154) Deprecate API interface, expose underlying "Teams" client
- (GH-183) Update Makefile and GitHub Actions Workflows
- (GH-190) Refactor GitHub Actions workflows to import logic

### Fixed

- (GH-166) Update `lintinstall` Makefile recipe
- (GH-184) Apply Go 1.19 specific doc comments linting fixes
- (GH-176) `./send_test.go:238:8: second argument to errors.As should not be
  *error`
- (GH-179) Wrong json key name for URL (uses uri instead)
  - credit: [@janfonas](https://github.com/janfonas)

## [v2.6.1] - 2022-02-25

### Changed

- Dependencies
  - `actions/setup-node`
    - `v2.2.0` to `v3`

- Linting
  - (GH-131) Expand linting GitHub Actions Workflow to include `oldstable`,
    `unstable` container images
  - (GH-132) Switch Docker image source from Docker Hub to GitHub Container
    Registry (GHCR)

### Fixed

- (GH-137) Missing doc comment for
  `teamsClient.AddWebhookURLValidationPatterns()`
- (GH-138) Missing doc comment for `teamsClient.ValidateWebhook()`
- (GH-141) send.go:306:15: nilness: tautological condition: non-nil != nil
  (govet)
- (GH-144) Incorrect field referenced in error message for
  `MessageCardSection.AddFact()`

## [v2.6.0] - 2021-07-09

### Added

- Features
  - Add support for PotentialActions (aka, "Actions")
    - credit: [@nmaupu](https://github.com/nmaupu)

- Documentation
  - Add separate `examples` directory containing standalone example code for
    most common use cases

### Changed

- Dependencies
  - `actions/setup-node`
    - `v2.1.5` to `v2.2.0`
    - update `node-version` value to always use latest LTS version instead of
      hard-coded version

- Linting
  - replace `golint`, `scopelint` linters, cleanup config
    - note: this specifically applies to linting performed via Makefile
      recipe, not (at present) the bulk of the CI linting checks

- Documentation
  - move examples from README to separate `examples` directory
  - Remove example from doc.go file, direct reader to main README
  - Update project status
    - remove history as it is likely no longer relevant (original
      project is discontinued at this point)
    - remove future (for the same reason)
  - Add explicit "Supported Releases" section to help make clear that
    the v1 series is no longer maintained
  - Remove explicit "used by" details, rely on dynamic listing provided
    by pkg.go.dev instead
  - Minor polish

## [v2.5.0] - 2021-04-08

### Added

- Features
  - Validation of webhook URLs using custom validation patterns
    - credit: [@nmaupu](https://github.com/nmaupu)
  - Validation of `MessageCard` type using a custom validation function (to
      override default validation behavior)
    - credit: [@nmaupu](https://github.com/nmaupu)

- Documentation
  - Add list of projects using this library
  - Update features list to include functionality added to this fork
    - Configurable validation of webhook URLs
    - Configurable validation of `MessageCard` type
    - Configurable timeouts
    - Configurable retry support

### Changed

- Dependencies
  - `actions/setup-node`
    - `v2.1.4` to `v2.1.5`

### Fixed

- Documentation
  - Misc typos
  - Grammatical tweaks
  - Attempt to clarify project status
    - i.e., not mothballed, just slow cadence

## [v2.4.2] - 2021-01-28

### Changed

- Apply regex pattern match for webhook URL validation instead of fixed
  strings in order to support matching private org webhook URL subdomains

### Fixed

- Updating an exiting webhook connector in Microsoft Teams switches the URL to
  unsupported `https://*.webhook.office.com/webhookb2/` format
- `SendWithRetry` method does not honor setting to disable webhook URL prefix
  validation
- Support for disabling webhook URL validation limited to just disabling
  validation of prefixes

## [v2.4.1] - 2021-01-28

### Changed

- (GH-59) Webhook URL API endpoint response validation now requires a `1` text
  string as the response body

### Fixed

- (GH-59) Microsoft Teams Webhook Connector "200 OK" status insufficient
  indication of success

## [v2.4.0] - 2021-01-28

### Added

- Add (optional) support for disabling webhook URL prefix validation
  - credit: [@odise](https://github.com/odise)

### Changed

- Documentation
  - Refresh "basic" example
  - Add example for disabling webhook URL prefix validation
  - Update "about this project" coverage
  - Swap GoDoc badge for pkg.go.dev badge

- Tests
  - Extend test coverage
  - Verbose test output by default (Makefile, GitHub Actions Workflow)

- Dependencies
  - `actions/setup-node`
    - `v2.1.1` to `v2.1.4`
  - `actions/checkout`
    - `v2.3.2` to `v2.3.4`
  - `stretchr/testify`
    - `v1.6.1` to `v1.7.0`

### Fixed

- minor linting error for commented code
- Tests fail to assert that any errors which occur are expected, only the
  types

## [v2.3.0] - 2020-08-29

### Added

- Add package-level logging for formatting functions
  - as with other package-level logging, this is disabled by default

- API
  - add `SendWithRetry` method based on the `teams.SendMessage` function from
    the `atc0005/send2teams` project
    - actively working to move relevant content from that project to this one

### Fixed

- YYYY-MM-DD date formatting of changelog version entries

## [v2.2.0] - 2020-08-28

### Added

- Add package-level logger
- Extend API to allow request cancellation via context
- Add formatting functions useful for text conversion
  - Convert Windows/Mac/Linux EOL to Markdown break statements
    - used to provide equivalent Teams-compatible formatting
  - Format text as code snippet
    - this inserts leading and trailing ` character to provide Markdown string
      formatting
  - Format text as code block
    - this inserts three leading and trailing ` characters to provide Markdown
      code block formatting
  - *`Try`* variants of code formatting functions
    - return formatted string if no errors, otherwise return the original
      string

### Changed

- Expose API response strings containing potential error messages
- README
  - Explicitly note that this fork is now standalone until such time that the
    upstream project resumes development/maintenance efforts

### Fixed

- CHANGELOG section link in previous release
- Invalid `RoundTripper` implementation used in `TestTeamsClientSend` test
  function
  - see `GH-46` and `GH-47`; thank you `@davecheney` for the fix!

## [v2.1.1] - 2020-08-25

### Added

- README
  - Add badges for GitHub Actions Workflows
  - Add release badge for latest project release
- Add CHANGELOG file
- Add GoDoc package-level documentation
- Extend webhook validation error handling
- Add Docker-based GitHub Actions Workflows
- Enable Dependabot updates
- Add Markdownlint config file

### Changed

- README
  - Replace badge for latest tag with latest release
  - Update GoDoc badge to reference this fork
  - Update license badge to reference this fork
  - Add new sections common to other projects that I maintain
    - table of contents
    - overview
    - changelog
    - references
    - features
- Vendor dependencies
- Update license to add @atc0005 (new) in addition to @dasrick (existing)
- Update go.mod to replace upstream with this fork
- Rename golangci-lint config file to match officially supported name
- Remove files no longer used by this fork
  - Travis CI configuration
  - editorconfig file (and settings)
- Add license header to source files
  - combined copyright statement for existing files
  - single copyright statement for new files

### Fixed

- Add missing Facts assignment in MessageCardSection
- scopelint: Fix improper range loop var reference
- Fix misc linting issues with README
- Test failure from previous upstream pull request submissions
  - `Object expected to be of type *url.Error, but was *errors.errorString`
- Misc linting issues with primary and test files

## [v2.1.0] - 2020-04-08

### Added

- `MessageCard` type includes additional fields
  - `Type` and `Context` fields provide required JSON payload
    fields
    - preset to required static values via updated
      `NewMessageCard()` constructor
  - `Summary`
    - required if `Text` field is not set, optional otherwise
  - `Sections` slice
    - `MessageCardSection` type

- Additional nested types
  - `MessageCardSection`
  - `MessageCardSectionFact`
  - `MessageCardSectionImage`

- Additional methods for `MessageCard` and nested types
  - `MessageCard.AddSection()`
  - `MessageCardSection.AddFact()`
  - `MessageCardSection.AddFactFromKeyValue()`
  - `MessageCardSection.AddImage()`
  - `MessageCardSection.AddHeroImageStr()`
  - `MessageCardSection.AddHeroImage()`

- Additional factory functions
  - `NewMessageCardSection()`
  - `NewMessageCardSectionFact()`
  - `NewMessageCardSectionImage()`

- `IsValidMessageCard()` added to check for minimum required
    field values.
  - This function has the potential to be extended
    later with additional validation steps.

- Wrapper `IsValidInput()` added to handle all validation
  needs from one location.
  - the intent was to both solve a CI error and provide
    a location to easily extend validation checks in
    the future (if needed)

### Changed

- `MessageCard` type includes additional fields
- `NewMessageCard` factory function sets fields needed for
   required JSON payload fields
  - `Type`
  - `Context`

- `teamsClient.Send()` method updated to apply `MessageCard` struct
  validation alongside existing webhook URL validation

- `isValidWebhookURL()` exported as `IsValidWebhookURL()` so that client
  code can use the validation functionality instead of repeating the
  code
  - e.g., flag value validation for "fail early" behavior

### Known Issues

- No support in this set of changes for `potentialAction` types
  - `ViewAction`
  - `OpenUri`
  - `HttpPOST`
  - `ActionCard`
  - `InvokeAddInCommand`
    - Outlook specific based on what I read; likely not included
      in a future release due to non-Teams specific usage

## [v2.0.0] - 2020-03-29

### Breaking

- `NewClient()` will NOT return multiple values
- remove provided mock

### Changed

- switch dependency/package management tool to from `dep` to `go mod`
- switch from `golint` to `golangci-lint`
- add more golang versions to pass via travis-ci

## [v1.3.1] - 2020-03-29

### Fixed

- fix redundant error logging
- fix redundant comment

## [v1.3.0] - 2020-03-26

### Changed

- feature: allow multiple valid webhook URL FQDNs (thx @atc0005)

## [v1.2.0] - 2019-11-08

### Added

- add mock

### Changed

- update deps
- `gosimple` (shorten two conditions)

## [v1.1.1] - 2019-05-02

### Changed

- rename client interface into API
- update deps

### Fixed

- fix typo in README

## [v1.1.0] - 2019-04-30

### Added

- add missing tests
- append documentation

### Changed

- add/change to client/interface

## [v1.0.0] - 2019-04-29

### Added

- add initial functionality of sending messages to MS Teams channel

[Unreleased]: https://github.com/atc0005/go-teams-notify/compare/v2.14.0...HEAD
[v2.14.0]: https://github.com/atc0005/go-teams-notify/releases/tag/v2.14.0
[v2.13.0]: https://github.com/atc0005/go-teams-notify/releases/tag/v2.13.0
[v2.12.0]: https://github.com/atc0005/go-teams-notify/releases/tag/v2.12.0
[v2.11.0]: https://github.com/atc0005/go-teams-notify/releases/tag/v2.11.0
[v2.10.0]: https://github.com/atc0005/go-teams-notify/releases/tag/v2.10.0
[v2.9.0]: https://github.com/atc0005/go-teams-notify/releases/tag/v2.9.0
[v2.8.0]: https://github.com/atc0005/go-teams-notify/releases/tag/v2.8.0
[v2.7.1]: https://github.com/atc0005/go-teams-notify/releases/tag/v2.7.1
[v2.7.0]: https://github.com/atc0005/go-teams-notify/releases/tag/v2.7.0
[v2.6.1]: https://github.com/atc0005/go-teams-notify/releases/tag/v2.6.1
[v2.6.0]: https://github.com/atc0005/go-teams-notify/releases/tag/v2.6.0
[v2.5.0]: https://github.com/atc0005/go-teams-notify/releases/tag/v2.5.0
[v2.4.2]: https://github.com/atc0005/go-teams-notify/releases/tag/v2.4.2
[v2.4.1]: https://github.com/atc0005/go-teams-notify/releases/tag/v2.4.1
[v2.4.0]: https://github.com/atc0005/go-teams-notify/releases/tag/v2.4.0
[v2.3.0]: https://github.com/atc0005/go-teams-notify/releases/tag/v2.3.0
[v2.2.0]: https://github.com/atc0005/go-teams-notify/releases/tag/v2.2.0
[v2.1.1]: https://github.com/atc0005/go-teams-notify/releases/tag/v2.1.1
[v2.1.0]: https://github.com/atc0005/go-teams-notify/releases/tag/v2.1.0
[v2.0.0]: https://github.com/atc0005/go-teams-notify/releases/tag/v2.0.0
[v1.3.1]: https://github.com/atc0005/go-teams-notify/releases/tag/v1.3.1
[v1.3.0]: https://github.com/atc0005/go-teams-notify/releases/tag/v1.3.0
[v1.2.0]: https://github.com/atc0005/go-teams-notify/releases/tag/v1.2.0
[v1.1.1]: https://github.com/atc0005/go-teams-notify/releases/tag/v1.1.1
[v1.1.0]: https://github.com/atc0005/go-teams-notify/releases/tag/v1.1.0
[v1.0.0]: https://github.com/atc0005/go-teams-notify/releases/tag/v1.0.0
