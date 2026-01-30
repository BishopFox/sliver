# Change Log
All notable changes to this project will be documented in this file.

This project adheres to [Semantic Versioning](http://semver.org/).

[2022-03-09] Version 2.6.9
--------------------------
**Library - Chore**
- [PR #110](https://github.com/sendgrid/rest/pull/110): push Datadog Release Metric upon deploy success. Thanks to [@eshanholtz](https://github.com/eshanholtz)!


[2022-02-09] Version 2.6.8
--------------------------
**Library - Chore**
- [PR #109](https://github.com/sendgrid/rest/pull/109): upgrade supported language versions. Thanks to [@childish-sambino](https://github.com/childish-sambino)!
- [PR #108](https://github.com/sendgrid/rest/pull/108): add gh release to workflow. Thanks to [@shwetha-manvinkurke](https://github.com/shwetha-manvinkurke)!


[2022-01-12] Version 2.6.7
--------------------------
**Library - Chore**
- [PR #107](https://github.com/sendgrid/rest/pull/107): update license year. Thanks to [@JenniferMah](https://github.com/JenniferMah)!


[2021-12-15] Version 2.6.6
--------------------------
**Library - Chore**
- [PR #106](https://github.com/sendgrid/rest/pull/106): migrate to gh actions. Thanks to [@beebzz](https://github.com/beebzz)!


[2021-09-22] Version 2.6.5
--------------------------
**Library - Chore**
- [PR #105](https://github.com/sendgrid/rest/pull/105): add tests against v1.16. Thanks to [@shwetha-manvinkurke](https://github.com/shwetha-manvinkurke)!


[2021-05-05] Version 2.6.4
--------------------------
**Library - Chore**
- [PR #103](https://github.com/sendgrid/rest/pull/103): follow up context.Context support. Thanks to [@johejo](https://github.com/johejo)!


[2021-03-15] Version 2.6.3
--------------------------
**Library - Fix**
- [PR #92](https://github.com/sendgrid/rest/pull/92): add SendWithContext function. Thanks to [@someone1](https://github.com/someone1)!


[2020-10-14] Version 2.6.2
--------------------------
**Library - Fix**
- [PR #101](https://github.com/sendgrid/rest/pull/101): Pass empty client instead of http.DefaultClient. Thanks to [@mateorider](https://github.com/mateorider)!


[2020-08-19] Version 2.6.1
--------------------------
**Library - Chore**
- [PR #100](https://github.com/sendgrid/rest/pull/100): update GitHub branch references to use HEAD. Thanks to [@thinkingserious](https://github.com/thinkingserious)!


[2020-02-19] Version 2.6.0
--------------------------
**Library - Feature**
- [PR #73](https://github.com/sendgrid/rest/pull/73): Dockerize sendgrid/rest. Thanks to [@graystevens](https://github.com/graystevens)!


[2020-02-05] Version 2.5.1
--------------------------
**Library - Docs**
- [PR #77](https://github.com/sendgrid/rest/pull/77): Run Grammarly on *.md files. Thanks to [@obahareth](https://github.com/obahareth)!
- [PR #86](https://github.com/sendgrid/rest/pull/86): Fixed link to bug report template. Thanks to [@alxshelepenok](https://github.com/alxshelepenok)!


[2020-01-30] Version 2.5.0
--------------------------
**Library - Docs**
- [PR #97](https://github.com/sendgrid/rest/pull/97): baseline all the templated markdown docs. Thanks to [@childish-sambino](https://github.com/childish-sambino)!
- [PR #88](https://github.com/sendgrid/rest/pull/88): add our Developer Experience Engineer career opportunity to the README. Thanks to [@mptap](https://github.com/mptap)!
- [PR #65](https://github.com/sendgrid/rest/pull/65): added "Code Review" section to CONTRIBUTING.md. Thanks to [@aleien](https://github.com/aleien)!
- [PR #80](https://github.com/sendgrid/rest/pull/80): add first timers guide for newcomers. Thanks to [@daniloff200](https://github.com/daniloff200)!
- [PR #82](https://github.com/sendgrid/rest/pull/82): update contribution guide with new workflow. Thanks to [@radlinskii](https://github.com/radlinskii)!
- [PR #62](https://github.com/sendgrid/rest/pull/62): update CONTRIBUTING.md with environment variables section. Thanks to [@thepriefy](https://github.com/thepriefy)!

**Library - Chore**
- [PR #96](https://github.com/sendgrid/rest/pull/96): prep repo for automation. Thanks to [@thinkingserious](https://github.com/thinkingserious)!
- [PR #94](https://github.com/sendgrid/rest/pull/94): add current Go version to Travis. Thanks to [@pangaunn](https://github.com/pangaunn)!
- [PR #93](https://github.com/sendgrid/rest/pull/93): add current Go versions to Travis. Thanks to [@gliptak](https://github.com/gliptak)!
- [PR #83](https://github.com/sendgrid/rest/pull/83): follow godoc deprecation standards. Thanks to [@vaskoz](https://github.com/vaskoz)!
- [PR #74](https://github.com/sendgrid/rest/pull/74): create README.md in use-cases. Thanks to [@ajloria](https://github.com/ajloria)!

**Library - Feature**
- [PR #72](https://github.com/sendgrid/rest/pull/72): do not swallow the error code. Thanks to [@Succo](https://github.com/Succo)!


[2018-04-09] Version 2.4.1
--------------------------
### Fixed
- Pull #71, Solves #70
- Fix Travis CI Build
- Special thanks to [Vasko Zdravevski](https://github.com/vaskoz) for the PR!

## [2.4.0] - 2017-4-10
### Added
- Pull #18, Solves #17
- Add RestError Struct for an error handling
- Special thanks to [Takahiro Ikeuchi](https://github.com/iktakahiro) for the PR!

## [2.3.1] - 2016-10-14
### Changed
- Pull #15, solves Issue #7
- Moved QueryParams processing into BuildRequestObject
- Special thanks to [Gábor Lipták](https://github.com/gliptak) for the PR!

## [2.3.0] - 2016-10-04
### Added
- Pull [#10] [Allow for custom Content-Types](https://github.com/sendgrid/rest/issues/10)

## [2.2.0] - 2016-07-28
### Added
- Pull [#9](https://github.com/sendgrid/rest/pull/9): Allow for setting a custom HTTP client
- [Here](rest_test.go#L127) is an example of usage
- This enables usage of the [sendgrid-go library](https://github.com/sendgrid/sendgrid-go) on [Google App Engine (GAE)](https://cloud.google.com/appengine/)
- Special thanks to [Chris Broadfoot](https://github.com/broady) and [Sridhar Venkatakrishnan](https://github.com/sridharv) for providing code and feedback!

## [2.1.0] - 2016-06-10
### Added
- Automatically add Content-Type: application/json when there is a request body

## [2.0.0] - 2016-06-03
### Changed
- Made the Request and Response variables non-redundant. e.g. request.RequestBody becomes request.Body

## [1.0.2] - 2016-04-07
### Added
- these changes are thanks to [deckarep](https://github.com/deckarep). Thanks!
- more updates to error naming convention
- more error handing on HTTP request

## [1.0.1] - 2016-04-07
### Added
- these changes are thanks to [deckarep](https://github.com/deckarep). Thanks!
- update error naming convention
- explicitly define supported HTTP verbs
- better error handling on HTTP request

## [1.0.0] - 2016-04-05
### Added
- We are live!
