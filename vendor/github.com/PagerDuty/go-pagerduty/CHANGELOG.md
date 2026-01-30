# Changelog

## [v1.8.0](https://github.com/PagerDuty/go-pagerduty/tree/v1.8.0) (2024-01-09)
### What's Changed
* Adds resolved_at and updated_at fields to Incident by @surik in https://github.com/PagerDuty/go-pagerduty/pull/482
* Adds support for auto_pause_notifications_parameters by @darrendao in https://github.com/PagerDuty/go-pagerduty/pull/490
* Fixes ResponderRequest unmarshalling of IncidentResponders by @allyjweir in https://github.com/PagerDuty/go-pagerduty/pull/493
* Adds support for license and licenses allocated based operations #480 by @gerardocorea in https://github.com/PagerDuty/go-pagerduty/pull/494
* Refactors nil *string initialization by @typeid in https://github.com/PagerDuty/go-pagerduty/pull/488
* [CSGI-1827] Add Standards support by @imjaroiswebdev in https://github.com/PagerDuty/go-pagerduty/pull/499
* Updated README.md replacing deprecated calls with corresponding new ones by @oleksiypavlenko in https://github.com/PagerDuty/go-pagerduty/pull/498
* [CSGI-1984] Add `From` field to `ListResponsePlays` method options by @imjaroiswebdev in https://github.com/PagerDuty/go-pagerduty/pull/502
* Handle invalid ChangeEvent APIErrorObject response by @icholy in https://github.com/PagerDuty/go-pagerduty/pull/479
* [CSGI-2365] Add client User Agent overwrite capability by @imjaroiswebdev in https://github.com/PagerDuty/go-pagerduty/pull/503
* Updates go to 1.19 and a few dependencies by @ChuckCrawford in https://github.com/PagerDuty/go-pagerduty/pull/504
* Bump golang.org/x/crypto from 0.16.0 to 0.17.0 by @dependabot in https://github.com/PagerDuty/go-pagerduty/pull/505

### New Contributors
* @surik made their first contribution in https://github.com/PagerDuty/go-pagerduty/pull/482
* @darrendao made their first contribution in https://github.com/PagerDuty/go-pagerduty/pull/490
* @allyjweir made their first contribution in https://github.com/PagerDuty/go-pagerduty/pull/493
* @gerardocorea made their first contribution in https://github.com/PagerDuty/go-pagerduty/pull/494
* @typeid made their first contribution in https://github.com/PagerDuty/go-pagerduty/pull/488
* @imjaroiswebdev made their first contribution in https://github.com/PagerDuty/go-pagerduty/pull/499
* @oleksiypavlenko made their first contribution in https://github.com/PagerDuty/go-pagerduty/pull/498
* @icholy made their first contribution in https://github.com/PagerDuty/go-pagerduty/pull/479
* @dependabot made their first contribution in https://github.com/PagerDuty/go-pagerduty/pull/505

**Full Changelog**: https://github.com/PagerDuty/go-pagerduty/compare/v1.7.0...v1.8.0

## [v1.7.0](https://github.com/PagerDuty/go-pagerduty/tree/v1.7.0) (2023-05-17)

### What's Changed
* Upgades Go and dependencies by @ChuckCrawford in https://github.com/PagerDuty/go-pagerduty/pull/466
* Add Incident Notification Subscribers by @caveman280 in https://github.com/PagerDuty/go-pagerduty/pull/461
* Implement Event Orchestrations API by @EronWright in https://github.com/PagerDuty/go-pagerduty/pull/450

### New Contributors
* @caveman280 made their first contribution in https://github.com/PagerDuty/go-pagerduty/pull/461
* @EronWright made their first contribution in https://github.com/PagerDuty/go-pagerduty/pull/450

**Full Changelog**: https://github.com/PagerDuty/go-pagerduty/compare/v1.6.0...v1.7.0

## [v1.6.0](https://github.com/PagerDuty/go-pagerduty/tree/v1.6.0) (2022-09-21)

[Full Changelog](https://github.com/PagerDuty/go-pagerduty/compare/v1.5.0...1.6.0)
[Milestone](https://github.com/PagerDuty/go-pagerduty/milestone/7)

### A Quick Word from Us
Hello there! We just wanted to drop in and address the elephant in the room. In a [previous release](https://github.com/PagerDuty/go-pagerduty/releases/tag/v1.5.0) we made a promise to maintain API compatibility for future v1 releases. This release contains a few changes that may require minor updates to the way your code interacts with API request and response objects. These changes are necessary in order to make these APIs work properly for as many of you as possible.

One of our goals with this client library is to provide a working client that properly interacts with the [PagerDuty Public API](https://developer.pagerduty.com/api-reference/). While we remain committed to not breaking API compatibility for reasons such as "making the library [better | easier to use | etc.]"; we believe it is in all of our best interest to make this client library work.

As always, we welcome your feedback on this decision via a GitHub issue.

### Highlights
* Returns custom error type `EventsAPIV2Error` for errors that occur with Events API (V2) calls by @theckman in https://github.com/PagerDuty/go-pagerduty/pull/419
* List notifications API now works when using `includes` by @jaimegago in https://github.com/PagerDuty/go-pagerduty/pull/439
* Improves support for macOS builds by @mjlshen in https://github.com/PagerDuty/go-pagerduty/pull/434
* Fix potential panic when debug request captures are enabled by @theckman in https://github.com/PagerDuty/go-pagerduty/pull/443
* Fixes memory leak by @attilakunelwood in https://github.com/PagerDuty/go-pagerduty/pull/453
* Prevent leak on error responses by @ChuckCrawford in https://github.com/PagerDuty/go-pagerduty/pull/454

### Breaking Changes
* Fix unmarshaling error with AlertGroupingParameters timeout by @mjlshen in https://github.com/PagerDuty/go-pagerduty/pull/459
* Fix parsing bug for AlertGroupingParameters by @mjlshen in https://github.com/PagerDuty/go-pagerduty/pull/448
* Fix ResponderRequest regression by @ehlerorngard in https://github.com/PagerDuty/go-pagerduty/pull/452

### New Contributors
* @jaimegago made their first contribution in https://github.com/PagerDuty/go-pagerduty/pull/439
* @mjlshen made their first contribution in https://github.com/PagerDuty/go-pagerduty/pull/434
* @attilakunelwood made their first contribution in https://github.com/PagerDuty/go-pagerduty/pull/453
* @ChuckCrawford made their first contribution in https://github.com/PagerDuty/go-pagerduty/pull/454
* @ehlerorngard made their first contribution in https://github.com/PagerDuty/go-pagerduty/pull/452

## [v1.5.1](https://github.com/PagerDuty/go-pagerduty/tree/v1.5.1) (2022-04-24) - Bug Fixes

[Milestone](https://github.com/PagerDuty/go-pagerduty/milestone/8)
[Full Changelog](https://github.com/PagerDuty/go-pagerduty/compare/v1.5.0...v1.5.1)

**Highlights**

- Fix panic that would occur when debug request capturing is enabled, and request has a nil HTTP body (GET request)
- Fix options for ListNotifications, where URL parameters weren't being set correctly.

**Closed issues**

None

**Merged pull requests**

- Backport fix for panic when debug request captures are enabled [\#444](https://github.com/PagerDuty/go-pagerduty/pull/444) ([theckman](https://github.com/theckman))
- Backport: Fix list notifications options [\#445](https://github.com/PagerDuty/go-pagerduty/pull/445) ([jaimegago](https://github.com/jaimegago), backported by [theckman](https://github.com/theckman))

## [v1.5.0](https://github.com/PagerDuty/go-pagerduty/tree/v1.5.0) (2022-01-22) - BREAKING CHANGES

[Milestone](https://github.com/PagerDuty/go-pagerduty/milestone/2)
[Full Changelog](https://github.com/PagerDuty/go-pagerduty/compare/v1.4.3...v1.5.0)

**NOTICE**

This release is special, and intentionally includes breaking API changes without
bumping the module's major version. We apologize for any inconveience this
causes, but we felt this approach was better than incurring the cost of
releasing v2 today. Specifically, we believed it was best to break the API, so
that you became aware of features you depended on that were not working as you
expected. We welcome your feedback on this decision via a GitHub issue.

These changes largely fix API issues that would have made it impossible for the
this module to be used correctly. Most often this is due to incorrect or invalid
struct fields, others were a lack of required inputs to specific actions.

We commit henceforth to maintaining API compatibility for future v1 releases.

**Highlights**

- Add support for handling signature verification of V3 Webhook requests.
- Update `APIError` type to provide more helpful error strings.
- Add support for API debugging, by allowing capture of the request/response from the API.
- Added various missing fields to different structs.
- Add support for response plays, fetching audit records, and setting up email filters.

**Breaking changes**

- Fix the ResponderRequest input/output structures [\#328](https://github.com/PagerDuty/go-pagerduty/pull/328) ([CerealBoy](https://github.com/CerealBoy))
- Fix overlapping struct fields in Incident, Service, and User types [\#332](https://github.com/PagerDuty/go-pagerduty/pull/332) ([theckman](https://github.com/theckman))
- Remove *http.Response return from different API methods. [\#357](https://github.com/PagerDuty/go-pagerduty/pull/357) [\#358](https://github.com/PagerDuty/go-pagerduty/pull/358) [\#359](https://github.com/PagerDuty/go-pagerduty/pull/359) [\#360](https://github.com/PagerDuty/go-pagerduty/pull/360) [\#361](https://github.com/PagerDuty/go-pagerduty/pull/361) ([theckman](https://github.com/theckman))
- Add missing required From parameter to ManageIncidentAlerts [\#380](https://github.com/PagerDuty/go-pagerduty/pull/380) ([theckman](https://github.com/theckman))
- Fix mismatches between REST API and struct definitions [\#396](https://github.com/PagerDuty/go-pagerduty/pull/396) [\#414](https://github.com/PagerDuty/go-pagerduty/pull/414) ([theckman](https://github.com/theckman))
- Update pagination query params to conform to API spec [\#405](https://github.com/PagerDuty/go-pagerduty/pull/405) ([theckman](https://github.com/theckman)

**Closed issues**

- incident.ID or incident.Id [\#218](https://github.com/PagerDuty/go-pagerduty/issues/218) ([mblaschke](https://github.com/mblaschke))
- Improper unmarshalling [\#232](https://github.com/PagerDuty/go-pagerduty/issues/232) ([Erog38](https://github.com/Erog38))
- Update initialisms / acronyms to be capitalized [\#268](https://github.com/PagerDuty/go-pagerduty/issues/268) ([theckman](https://github.com/theckman))
- Add IncidentDetails.Title field and mark IncidentDetails.Description as deprecated [\#277](https://github.com/PagerDuty/go-pagerduty/issues/277) ([theckman](https://github.com/theckman))
- Remove *http.Response returns from API methods [\#305](https://github.com/PagerDuty/go-pagerduty/issues/305) ([theckman](https://github.com/theckman))
- Add ability to create or update integrations with email filter rules [\#315](https://github.com/PagerDuty/go-pagerduty/issues/315) ([gerardocorea](https://github.com/gerardocorea))
- EscalationRule struct should accept a slice of APIReference rather then APIObject for Targets [\#316](https://github.com/PagerDuty/go-pagerduty/issues/316) ([gerardocorea](https://github.com/gerardocorea))
- User slice fields missing omitempty [\#343](https://github.com/PagerDuty/go-pagerduty/issues/343) ([theckman](https://github.com/theckman))
- Should all fields in Service type be omitempty [\#348](https://github.com/PagerDuty/go-pagerduty/issues/348) ([callumj](https://github.com/callumj))
- ManageIncidentsOptions doesn't take EscalationLevel [\#364](https://github.com/PagerDuty/go-pagerduty/issues/364) ([sim1s](https://github.com/sim1s))
- Setting conference information in new incident [\#373](https://github.com/PagerDuty/go-pagerduty/issues/373) ([bparlidoordash](https://github.com/bparlidoordash))
- Validate that all structure formats and method signatures work with PagerDuty API [\#389](https://github.com/PagerDuty/go-pagerduty/issues/389) ([theckman](https://github.com/theckman))
- When creating an incident, consumers shouldn't need to set the Type field [\#390](https://github.com/PagerDuty/go-pagerduty/issues/390) ([theckman](https://github.com/theckman))
- Reduce code duplication in analytics.go [\#393](https://github.com/PagerDuty/go-pagerduty/issues/393) ([theckman](https://github.com/theckman))
- Add support for fetching Audit Records [\#394](https://github.com/PagerDuty/go-pagerduty/issues/394) ([theckman](https://github.com/theckman))

**Merged pull requests**

- Add assignees to log entry [\#237](https://github.com/PagerDuty/go-pagerduty/pull/237) ([tautvydass](https://github.com/tautvydass))
- Add support for better API debugging; start v1.5.0 development [\#325](https://github.com/PagerDuty/go-pagerduty/pull/325) ([theckman](https://github.com/theckman))
- Fix the ResponderRequest input/output structures [\#328](https://github.com/PagerDuty/go-pagerduty/pull/328) ([CerealBoy](https://github.com/CerealBoy))
- Fix overlapping struct fields & last golint errors [\#332](https://github.com/PagerDuty/go-pagerduty/pull/332) ([theckman](https://github.com/theckman))
- Add comment indicating IncidentDetails.Description is deprecated [\#333](https://github.com/PagerDuty/go-pagerduty/pull/333) ([theckman](https://github.com/theckman))
- Update APIError.Error() to provide more helpful error messages [\#334](https://github.com/PagerDuty/go-pagerduty/pull/334) ([theckman](https://github.com/theckman))
- Add comment above IncidentDetails.Alerts field explaining behaviors [\#335](https://github.com/PagerDuty/go-pagerduty/pull/335) ([theckman](https://github.com/theckman))
- Correct formatting of deprecation notices. [\#340](https://github.com/PagerDuty/go-pagerduty/pull/340) ([dsymonds](https://github.com/dsymonds))
- Fix `pd schedule override create`. [\#341](https://github.com/PagerDuty/go-pagerduty/pull/341) ([dsymonds](https://github.com/dsymonds))
- allow setting suppress to false [\#345](https://github.com/PagerDuty/go-pagerduty/pull/345) ([cluarkhpe](https://github.com/cluarkhpe))
- Swap two transposed words in the README file [\#350](https://github.com/PagerDuty/go-pagerduty/pull/350) ([theckman](https://github.com/theckman))
- Fixing link to PD API Reference [\#356](https://github.com/PagerDuty/go-pagerduty/pull/356) ([stmcallister](https://github.com/stmcallister))
- Remove returned *http.Response from incident-related methods [\#357](https://github.com/PagerDuty/go-pagerduty/pull/357) ([theckman](https://github.com/theckman))
- Remove returned *http.Response from business svc related methods [\#358](https://github.com/PagerDuty/go-pagerduty/pull/358) ([theckman](https://github.com/theckman))
- Remove returned *http.Response from svc dependency related methods [\#359](https://github.com/PagerDuty/go-pagerduty/pull/359) ([theckman](https://github.com/theckman))
- Remove returned *http.Response from tag-related methods [\#360](https://github.com/PagerDuty/go-pagerduty/pull/360) ([theckman](https://github.com/theckman))
- Remove returned *http.Response from ruleset-related methods [\#361](https://github.com/PagerDuty/go-pagerduty/pull/361) ([theckman](https://github.com/theckman))
- implement missing maintenance-window subcommands [\#363](https://github.com/PagerDuty/go-pagerduty/pull/363) ([Hsn723](https://github.com/Hsn723))
- Add json field incidents_responders to Incident struct [\#365](https://github.com/PagerDuty/go-pagerduty/pull/365) ([sostakas](https://github.com/sostakas))
- Adding escalation level to ManageIncidentOptions [\#366](https://github.com/PagerDuty/go-pagerduty/pull/366) ([sim1s](https://github.com/sim1s))
- Add v3 webhook signature verification [\#370](https://github.com/PagerDuty/go-pagerduty/pull/370) ([theckman](https://github.com/theckman))
- Fix test after merging #332 (2f47dfc62321b) [\#371](https://github.com/PagerDuty/go-pagerduty/pull/371) ([theckman](https://github.com/theckman))
- Add title to ManageIncidentOptions [\#372](https://github.com/PagerDuty/go-pagerduty/pull/372) ([d33d33](https://github.com/d33d33))
- Add Service and User to LogEntry [\#377](https://github.com/PagerDuty/go-pagerduty/pull/377) ([theckman](https://github.com/theckman))
- Add missing required parameeter to ManageIncidentAlerts [\#380](https://github.com/PagerDuty/go-pagerduty/pull/380) ([theckman](https://github.com/theckman))
- Handle unexpected type changes in PagerDuty REST API error responses [\#382](https://github.com/PagerDuty/go-pagerduty/pull/382) ([theckman](https://github.com/theckman))
- Add omitempty JSON tag to User slice fields [\#383](https://github.com/PagerDuty/go-pagerduty/pull/383) ([theckman](https://github.com/theckman))
- Add omitempty JSON tag to specific Service fields [\#384](https://github.com/PagerDuty/go-pagerduty/pull/384)([theckman](https://github.com/theckman))
- Add support for adding email filters for Generic Email Integrations [\#385](https://github.com/PagerDuty/go-pagerduty/pull/385) ([theckman](https://github.com/theckman))
- Support adding conference bridge when creating or managing incidents [\#391](https://github.com/PagerDuty/go-pagerduty/pull/391) ([theckman](https://github.com/theckman))
- Mark Type struct field deprecated, for incident creation + management [\#392](https://github.com/PagerDuty/go-pagerduty/pull/392) ([theckman](https://github.com/theckman))
- Fix some mismatches between REST API and struct definitions [\#396](https://github.com/PagerDuty/go-pagerduty/pull/396) ([theckman](https://github.com/theckman))
- refactor: Reduce code duplication in analytics.go [\#397](https://github.com/PagerDuty/go-pagerduty/pull/397) ([t-junjie](https://github.com/t-junjie))
- Add support for escalation_policy.on_call_handoff_notifications field [\#401](https://github.com/PagerDuty/go-pagerduty/pull/401) ([zonorti](https://github.com/zonorti))
- Missing incident fields [\#402](https://github.com/PagerDuty/go-pagerduty/pull/402) ([zonorti](https://github.com/zonorti))
- Add extension enable [\#403](https://github.com/PagerDuty/go-pagerduty/pull/403) ([zonorti](https://github.com/zonorti))
- Add support for response_plays [\#404](https://github.com/PagerDuty/go-pagerduty/pull/404) ([petetanton](https://github.com/petetanton))
- Update pagination query to conform to API spec [\#405](https://github.com/PagerDuty/go-pagerduty/pull/405) ([theckman](https://github.com/theckman))
- add createStatusUpdate [\#406](https://github.com/PagerDuty/go-pagerduty/pull/406) ([kkawamura](https://github.com/kkawamura))
- feat: Add support for fetching Audit Records [\#408](https://github.com/PagerDuty/go-pagerduty/pull/408) ([t-junjie](https://github.com/t-junjie))
- Fix linter issues, update ResponsePlays API before v1.5.0 release [\#410](https://github.com/PagerDuty/go-pagerduty/pull/410) ([theckman](https://github.com/theckman))
- Second batch of fixes for API incompatibilities [\#414](https://github.com/PagerDuty/go-pagerduty/pull/414) ([theckman](https://github.com/theckman))
- Find a way to gracefully avoid one breaking change in #405 [\#416](https://github.com/PagerDuty/go-pagerduty/pull/416) ([theckman](https://github.com/theckman))
- Fix linter issues introduced by final PR merges [\#417](https://github.com/PagerDuty/go-pagerduty/pull/417) ([theckman](https://github.com/theckman))

## [v1.4.3](https://github.com/PagerDuty/go-pagerduty/tree/v1.4.3) (2021-11-13)

[Milestone](https://github.com/PagerDuty/go-pagerduty/milestone/6)
[Full Changelog](https://github.com/PagerDuty/go-pagerduty/compare/v1.4.2...v1.4.3)

**Highlights**
- Mitigate PagerDuty REST API bug that would result in a JSON parsing failure when reading an error response from the API. Prior to `v1.4.0` our error parsing logic was not impacted by the bug.

**Merged pull requests**
- Handle unexpected type changes in PagerDuty REST API error responses [\#382](https://github.com/PagerDuty/go-pagerduty/pull/382) [backported via [\#386](https://github.com/PagerDuty/go-pagerduty/pull/386)] ([theckman](https://github.com/theckman))

**Closed issues**
- APIError unmarshaling broken in 1.4 [\#339](https://github.com/PagerDuty/go-pagerduty/pull/339) ([dsymonds](https://github.com/dsymonds))

## [v1.4.2](https://github.com/PagerDuty/go-pagerduty/tree/v1.4.2) (2021-08-30)

[Milestone](https://github.com/PagerDuty/go-pagerduty/milestone/5)
[Full Changelog](https://github.com/PagerDuty/go-pagerduty/compare/v1.4.1...v1.4.2)

**Highlights**
- Fix bug that prevented CreateService() from working when SupportHours and ScheduledActions were unset.

**Merged pull requests**
- Add omitempty on Service.SupportHours and Service.ScheduledActions [\#352](https://github.com/PagerDuty/go-pagerduty/pull/352) ([theckman](https://github.com/theckman))

**Closed issues**
- Backward incompatability - CreateService - existing setup works in 1.3.0 but not in 1.4.x [\#346](https://github.com/PagerDuty/go-pagerduty/pull/346) ([onikroo](https://github.com/onikroo))

## [v1.4.1](https://github.com/PagerDuty/go-pagerduty/tree/v1.4.1) (2021-05-13)

[Milestone](https://github.com/PagerDuty/go-pagerduty/milestone/4)
[Full Changelog](https://github.com/PagerDuty/go-pagerduty/compare/v1.4.0...v1.4.1)

**Highlights**
- Fix bugs that impacted pagination of both tags and business services

**Merged pull requests**

- Fix query params on tags [\#329](https://github.com/PagerDuty/go-pagerduty/pull/329) ([jfmyers9](https://github.com/jfmyers9))
- Fix pagination within ListBusinessServices + Paginated [\#330](https://github.com/PagerDuty/go-pagerduty/pull/330) ([theckman](https://github.com/theckman))

## [v1.4.0](https://github.com/PagerDuty/go-pagerduty/tree/v1.4.0) (2021-04-23)

[Milestone](https://github.com/PagerDuty/go-pagerduty/milestone/3)
[Full Changelog](https://github.com/PagerDuty/go-pagerduty/compare/v1.3.0...v1.4.0)

**Highlights**

- Add support for passing a `context.Context` to all methods in package (in non-breaking way)
- Add new `APIError` type, that allows for [richer inspection of errors returned from API](https://github.com/PagerDuty/go-pagerduty/tree/v1.4.0#api-error-responses).
- Add support for the V2 Events API, Analytics, and Change Events.
- Miscellaneous bug fixes, including some that may result in incorrect API request/response handling.

**Closed issues**

- CreateEventWithHTTPClient nil pointer dereference [\#274](https://github.com/PagerDuty/go-pagerduty/issues/274) ([blockpane](https://github.com/blockpane))
- ManageIncidents - support set resolution [\#243](https://github.com/PagerDuty/go-pagerduty/issues/243) ([hagaishapira](https://github.com/hagaishapira))
- Exposing context.Context in exported API without requiring major version bump [\#267](https://github.com/PagerDuty/go-pagerduty/issues/267) ([theckman](https://github.com/theckman/))
- Fix any linter issues and add missing GoDoc comments [\#317](https://github.com/PagerDuty/go-pagerduty/issues/317) ([theckman](https://github.com/theckman))

**Merged pull requests**

- Fix rulset rule not respecting position "0" [\#236](https://github.com/PagerDuty/go-pagerduty/pull/236) ([zane-deg](https://github.com/zane-deg))
- Event v2 client [\#241](https://github.com/PagerDuty/go-pagerduty/pull/241) ([goatherder](https://github.com/goatherder))
- Add Support For Change Events [\#246](https://github.com/PagerDuty/go-pagerduty/pull/246) ([Sjeanpierre](https://github.com/Sjeanpierre))
- Correct namespacing of Change Event resource [\#248](https://github.com/PagerDuty/go-pagerduty/pull/248) ([Sjeanpierre](https://github.com/Sjeanpierre))
- add tag and tag_test [\#252](https://github.com/PagerDuty/go-pagerduty/pull/252) ([stmcallister](https://github.com/stmcallister))
- allow creating services with no scheduled actions [\#234](https://github.com/PagerDuty/go-pagerduty/pull/234) ([cluarkhpe](https://github.com/cluarkhpe))
- business_service: clean b.ID before calling UPDATE [\#253](https://github.com/PagerDuty/go-pagerduty/pull/253) ([GiedriusS](https://github.com/GiedriusS))
- service_dependency: fix (dis-)associate operations [\#254](https://github.com/PagerDuty/go-pagerduty/pull/254) ([GiedriusS](https://github.com/GiedriusS))
- ruleset: remove omitempty from Route [\#256](https://github.com/PagerDuty/go-pagerduty/pull/256) ([GiedriusS](https://github.com/GiedriusS))
- add service.alert_grouping_params field [\#257](https://github.com/PagerDuty/go-pagerduty/pull/257) ([stmcallister](https://github.com/stmcallister))
- fix creating services that don't use support hours [\#255](https://github.com/PagerDuty/go-pagerduty/pull/255) ([cluarkhpe](https://github.com/cluarkhpe))
- Provide a method for ferryign API errors back to the caller [#\265](https://github.com/PagerDuty/go-pagerduty/pull/265) ([theckman](https://github.com/theckman))
- Prepare internals for exposing context.Context in exported API [#\266](https://github.com/PagerDuty/go-pagerduty/pull/266) ([theckman](https://github.com/theckman))
- Update APIError struct to use new NullAPIErrorObject type for safety [#\272](https://github.com/PagerDuty/go-pagerduty/pull/272) ([theckman](https://github.com/theckman))
- Update internal HTTP methods to not take pointer to map as argument [#\269](https://github.com/PagerDuty/go-pagerduty/pull/269) ([theckman](https://github.com/theckman))
- Fix tags on log entry options struct [\#275](https://github.com/PagerDuty/go-pagerduty/pull/275) ([evnsio](https://github.com/evnsio))
- Add alerts to IncidentDetails struct [\#276](https://github.com/PagerDuty/go-pagerduty/pull/276) ([StupidScience](https://github.com/StupidScience))
- Fix logentry.channel json marshaling/unmarshaling [\#264](https://github.com/PagerDuty/go-pagerduty/pull/264) ([StupidScience](https://github.com/StupidScience))
- Fixed CreateRuleSet docs typo [\#281](https://github.com/PagerDuty/go-pagerduty/pull/281) ([neufeldtech](https://github.com/neufeldtech))
- Omitting EndpointURL from Extension if empty [\#282](https://github.com/PagerDuty/go-pagerduty/pull/282) ([au-akash](https://github.com/au-akash))
- add ListServicesPaginated to services api (and tests) [\#260](https://github.com/PagerDuty/go-pagerduty/pull/260) ([c6h12o6](https://github.com/c6h12o6))
- return nil instead of resp on error [\#278](https://github.com/PagerDuty/go-pagerduty/pull/278) (fixes [\#274](https://github.com/PagerDuty/go-pagerduty/issues/274)) ([blockpane](https://github.com/blockpane))
- Analytics [\#261](https://github.com/PagerDuty/go-pagerduty/pull/261) ([melchiormoulin](https://github.com/melchiormoulin))
- Add context.Context support to * (fixes [\#267](https://github.com/PagerDuty/go-pagerduty/issues/267)) ([theckman](https://github.com/theckman/))
  - [\#283](https://github.com/PagerDuty/go-pagerduty/pull/283), [\#284](https://github.com/PagerDuty/go-pagerduty/pull/284), [\#285](https://github.com/PagerDuty/go-pagerduty/pull/285), [\#286](https://github.com/PagerDuty/go-pagerduty/pull/286), [\#287](https://github.com/PagerDuty/go-pagerduty/pull/287), [\#288](https://github.com/PagerDuty/go-pagerduty/pull/288)
  - [\#289](https://github.com/PagerDuty/go-pagerduty/pull/289), [\#290](https://github.com/PagerDuty/go-pagerduty/pull/290), [\#291](https://github.com/PagerDuty/go-pagerduty/pull/291), [\#292](https://github.com/PagerDuty/go-pagerduty/pull/292), [\#293](https://github.com/PagerDuty/go-pagerduty/pull/293), [\#294](https://github.com/PagerDuty/go-pagerduty/pull/294)
  - [\#297](https://github.com/PagerDuty/go-pagerduty/pull/297), [\#298](https://github.com/PagerDuty/go-pagerduty/pull/298), [\#299](https://github.com/PagerDuty/go-pagerduty/pull/299), [\#300](https://github.com/PagerDuty/go-pagerduty/pull/300), [\#301](https://github.com/PagerDuty/go-pagerduty/pull/301), [\#303](https://github.com/PagerDuty/go-pagerduty/pull/303)
  - [\#306](https://github.com/PagerDuty/go-pagerduty/pull/306), [\#307](https://github.com/PagerDuty/go-pagerduty/pull/307), [\#308](https://github.com/PagerDuty/go-pagerduty/pull/308), [\#309](https://github.com/PagerDuty/go-pagerduty/pull/309), [\#322](https://github.com/PagerDuty/go-pagerduty/pull/322)
- Add support for Service Event Rules [\#304](https://github.com/PagerDuty/go-pagerduty/pull/304) ([mrzacarias](https://github.com/mrzacarias))
- teams and services analytics endpoints [\#312](https://github.com/PagerDuty/go-pagerduty/pull/312) ([newbootz](https://github.com/newbootz))
- Add Resolution string to ManageIncidentsOptions struct [\#313](https://github.com/PagerDuty/go-pagerduty/pull/313) (fixes [\#243](https://github.com/PagerDuty/go-pagerduty/issues/243)) ([theckman](https://github.com/theckman/))
- Fix invalid JSON struct tag, other linter issues [\#319](https://github.com/PagerDuty/go-pagerduty/pull/319) (fixes [\#317](https://github.com/PagerDuty/go-pagerduty/issues/317)) ([theckman](https://github.com/theckman))
- Add ability to set Escalation Policy when managing incidents [\#323](https://github.com/PagerDuty/go-pagerduty/pull/323) ([theckman](https://github.com/theckman)) superseded: [\#273](https://github.com/PagerDuty/go-pagerduty/pull/273) ([evnsio](https://github.com/evnsio))


## [v1.3.0](https://github.com/PagerDuty/go-pagerduty/tree/v1.3.0) (2020-09-08)

[Full Changelog](https://github.com/PagerDuty/go-pagerduty/compare/v1.2.0...v1.3.0)

**Closed issues:**

- `ListIncidents` pagination [\#238](https://github.com/PagerDuty/go-pagerduty/issues/238)

**Merged pull requests:**

- Fix ruleset rule not respecting position "0" [\#236](https://github.com/PagerDuty/go-pagerduty/pull/236) ([zane-deg](https://github.com/zane-deg))
- Adding Get Incident Alert and Manage Incident Alert endpoints [\#231](https://github.com/PagerDuty/go-pagerduty/pull/231) ([stmcallister](https://github.com/stmcallister))
- Add FirstTriggerLogEntry and CommonLogEntryField fields and json tags [\#230](https://github.com/PagerDuty/go-pagerduty/pull/230) ([afarbos](https://github.com/afarbos))
- adding business_service and service_dependency [\#228](https://github.com/PagerDuty/go-pagerduty/pull/228) ([stmcallister](https://github.com/stmcallister))
- update changelog for v1.2.0 [\#227](https://github.com/PagerDuty/go-pagerduty/pull/227) ([stmcallister](https://github.com/stmcallister))


## [v1.2.0](https://github.com/PagerDuty/go-pagerduty/tree/v1.2.0) (2020-06-04)

[Full Changelog](https://github.com/PagerDuty/go-pagerduty/compare/v1.1.2...v1.2.0)

**Closed issues:**

- Allowing custom API endpoint in NewClient config [\#198](https://github.com/PagerDuty/go-pagerduty/issues/198)
- service: SupportHours creation not supported [\#188](https://github.com/PagerDuty/go-pagerduty/issues/188)
- The "Channel" field doesn't expose all possible data fields [\#153](https://github.com/PagerDuty/go-pagerduty/issues/153)

**Merged pull requests:**

- Adding Rulesets and Ruleset Rules [\#226](https://github.com/PagerDuty/go-pagerduty/pull/226) ([stmcallister](https://github.com/stmcallister))
- Fix UpdateService [\#220](https://github.com/PagerDuty/go-pagerduty/pull/220) ([n-apalm](https://github.com/n-apalm))
- Add support for modifying an incident status and assignees [\#219](https://github.com/PagerDuty/go-pagerduty/pull/219) ([raidancampbell](https://github.com/raidancampbell))
- This should be requester\_id according to pagerduty docs [\#217](https://github.com/PagerDuty/go-pagerduty/pull/217) ([michael-bud](https://github.com/michael-bud))
- adding since and until to incident logentry options [\#216](https://github.com/PagerDuty/go-pagerduty/pull/216) ([stmcallister](https://github.com/stmcallister))
- User notification rules [\#215](https://github.com/PagerDuty/go-pagerduty/pull/215) ([heimweh](https://github.com/heimweh))
- List incident alerts [\#214](https://github.com/PagerDuty/go-pagerduty/pull/214) ([kilianw](https://github.com/kilianw))
- Bump golang to v1.14 [\#212](https://github.com/PagerDuty/go-pagerduty/pull/212) ([chenrui333](https://github.com/chenrui333))
- adding NewClientWithAPIEndpoint function [\#210](https://github.com/PagerDuty/go-pagerduty/pull/210) ([stmcallister](https://github.com/stmcallister))
- Webhook conforms to v2 struct [\#209](https://github.com/PagerDuty/go-pagerduty/pull/209) ([nbutton23](https://github.com/nbutton23))
- Add Teams to Schedule [\#208](https://github.com/PagerDuty/go-pagerduty/pull/208) ([miekg](https://github.com/miekg))
- adding Raw to LogEntry.Channel object [\#207](https://github.com/PagerDuty/go-pagerduty/pull/207) ([stmcallister](https://github.com/stmcallister))
- Updating the Version constant to v1.1.2 [\#206](https://github.com/PagerDuty/go-pagerduty/pull/206) ([stmcallister](https://github.com/stmcallister))
- updating changelog to v1.1.2 [\#205](https://github.com/PagerDuty/go-pagerduty/pull/205) ([stmcallister](https://github.com/stmcallister))
- Adding OAuth token support [\#203](https://github.com/PagerDuty/go-pagerduty/pull/203) ([chrisforrette](https://github.com/chrisforrette))

## [v1.1.2](https://github.com/PagerDuty/go-pagerduty/tree/v1.1.2) (2020-02-21)

[Full Changelog](https://github.com/PagerDuty/go-pagerduty/compare/1.1.1...v1.1.2)

**Closed issues:**

- EventV2Response doesn't match API response [\#186](https://github.com/PagerDuty/go-pagerduty/issues/186)
- List escalation policy with current on call members using include `current\_oncall` [\#181](https://github.com/PagerDuty/go-pagerduty/issues/181)
- Create service extension \(like slack extension\) over API [\#149](https://github.com/PagerDuty/go-pagerduty/issues/149)
- Mock Client? [\#148](https://github.com/PagerDuty/go-pagerduty/issues/148)
- Make a release? [\#146](https://github.com/PagerDuty/go-pagerduty/issues/146)
- Priority field should be optional according to API spec [\#135](https://github.com/PagerDuty/go-pagerduty/issues/135)
- Missing Services Extensions available over API [\#129](https://github.com/PagerDuty/go-pagerduty/issues/129)
- Missing ContactMethod operations [\#125](https://github.com/PagerDuty/go-pagerduty/issues/125)
- Add A CODEOWNERS file for easier review requests. [\#124](https://github.com/PagerDuty/go-pagerduty/issues/124)
- missing severity in create\_event.json object? [\#100](https://github.com/PagerDuty/go-pagerduty/issues/100)
- Assignment struct has no json conversion [\#92](https://github.com/PagerDuty/go-pagerduty/issues/92)
- Publish CLI binaries as releases [\#81](https://github.com/PagerDuty/go-pagerduty/issues/81)
- Package test coverage is lacking [\#70](https://github.com/PagerDuty/go-pagerduty/issues/70)
- Create releases with built binaries [\#50](https://github.com/PagerDuty/go-pagerduty/issues/50)

**Merged pull requests:**

- fixing eventV2Response to match API [\#204](https://github.com/PagerDuty/go-pagerduty/pull/204) ([stmcallister](https://github.com/stmcallister))
- Remove duplicate license link in README [\#202](https://github.com/PagerDuty/go-pagerduty/pull/202) ([ahornace](https://github.com/ahornace))
- Adding GetCurrentUser method [\#199](https://github.com/PagerDuty/go-pagerduty/pull/199) ([chrisforrette](https://github.com/chrisforrette))
- Adding User-Agent Headers [\#197](https://github.com/PagerDuty/go-pagerduty/pull/197) ([stmcallister](https://github.com/stmcallister))
- Implement the Incident endpoint for ResponderRequest [\#196](https://github.com/PagerDuty/go-pagerduty/pull/196) ([CerealBoy](https://github.com/CerealBoy))
- updating changelog with 1.1.1 [\#195](https://github.com/PagerDuty/go-pagerduty/pull/195) ([stmcallister](https://github.com/stmcallister))
- List team members, single page or all \(with helper for auto-pagination\) [\#192](https://github.com/PagerDuty/go-pagerduty/pull/192) ([mwhite-ibm](https://github.com/mwhite-ibm))

## [1.1.1](https://github.com/PagerDuty/go-pagerduty/tree/1.1.1) (2020-02-05)

[Full Changelog](https://github.com/PagerDuty/go-pagerduty/compare/1.1.0...1.1.1)

**Merged pull requests:**

- Community Contributions -- 05 Feb 2020 [\#194](https://github.com/PagerDuty/go-pagerduty/pull/194) ([stmcallister](https://github.com/stmcallister))
- Add support for extensions/extension schemas [\#193](https://github.com/PagerDuty/go-pagerduty/pull/193) ([heimweh](https://github.com/heimweh))
- Added AlertGrouping and AlertGroupingTimeout to Service [\#189](https://github.com/PagerDuty/go-pagerduty/pull/189) ([toneill818](https://github.com/toneill818))
- Adds oncall to escalation policy [\#183](https://github.com/PagerDuty/go-pagerduty/pull/183) ([ewilde](https://github.com/ewilde))
- Add ContactMethods operations [\#169](https://github.com/PagerDuty/go-pagerduty/pull/169) ([timlittle](https://github.com/timlittle))
- return http code with errors [\#134](https://github.com/PagerDuty/go-pagerduty/pull/134) ([yomashExpel](https://github.com/yomashExpel))

## [1.1.0](https://github.com/PagerDuty/go-pagerduty/tree/1.1.0) (2020-02-03)

[Full Changelog](https://github.com/PagerDuty/go-pagerduty/compare/1.0.4...1.1.0)

**Closed issues:**

- listOverrides result in JSON unmarshall failure [\#180](https://github.com/PagerDuty/go-pagerduty/issues/180)
- How to create incident via command pd? [\#171](https://github.com/PagerDuty/go-pagerduty/issues/171)
- Poorly documented, library code broken, please step it up.  [\#170](https://github.com/PagerDuty/go-pagerduty/issues/170)
- failed to create an incident [\#165](https://github.com/PagerDuty/go-pagerduty/issues/165)
- I need create incident function, Can we release the latest master? [\#163](https://github.com/PagerDuty/go-pagerduty/issues/163)
- Update logrus imports to github.com/sirupsen/logrus [\#160](https://github.com/PagerDuty/go-pagerduty/issues/160)
- build error cannot find package [\#144](https://github.com/PagerDuty/go-pagerduty/issues/144)
- Missing ListIncidentAlerts [\#141](https://github.com/PagerDuty/go-pagerduty/issues/141)
- ListIncidentsOptions Example [\#139](https://github.com/PagerDuty/go-pagerduty/issues/139)
- Support for V2 Event Management in the CLI [\#136](https://github.com/PagerDuty/go-pagerduty/issues/136)
- Custom connection settings [\#110](https://github.com/PagerDuty/go-pagerduty/issues/110)
- Missing the "From" parameter in Create note on an incident function [\#107](https://github.com/PagerDuty/go-pagerduty/issues/107)
- Support V2 events [\#83](https://github.com/PagerDuty/go-pagerduty/issues/83)
- Support Event Transformer Code? [\#67](https://github.com/PagerDuty/go-pagerduty/issues/67)
- Fix help flag behavior [\#18](https://github.com/PagerDuty/go-pagerduty/issues/18)

**Merged pull requests:**

- Tests [\#190](https://github.com/PagerDuty/go-pagerduty/pull/190) ([stmcallister](https://github.com/stmcallister))
- Modify ListOverrides and add ListOverridesResponse [\#185](https://github.com/PagerDuty/go-pagerduty/pull/185) ([dstevensio](https://github.com/dstevensio))
- client: allow overriding the api endpoint [\#173](https://github.com/PagerDuty/go-pagerduty/pull/173) ([heimweh](https://github.com/heimweh))
- Change makefiles and readme [\#172](https://github.com/PagerDuty/go-pagerduty/pull/172) ([dineshba](https://github.com/dineshba))
- Use Go modules [\#168](https://github.com/PagerDuty/go-pagerduty/pull/168) ([nbutton23](https://github.com/nbutton23))
- escalation\_policy: support clearing teams from an existing escalation policy [\#167](https://github.com/PagerDuty/go-pagerduty/pull/167) ([heimweh](https://github.com/heimweh))
- Correct JSON payload format for CreateIncident call [\#166](https://github.com/PagerDuty/go-pagerduty/pull/166) ([joepurdy](https://github.com/joepurdy))
- Use pointer to Priority so don't send an empty priority for incident [\#164](https://github.com/PagerDuty/go-pagerduty/pull/164) ([atomicules](https://github.com/atomicules))
- Update README.md [\#158](https://github.com/PagerDuty/go-pagerduty/pull/158) ([jonhyman](https://github.com/jonhyman))
- Fixed typo [\#155](https://github.com/PagerDuty/go-pagerduty/pull/155) ([uthark](https://github.com/uthark))
- Support Links in V2Event [\#154](https://github.com/PagerDuty/go-pagerduty/pull/154) ([alindeman](https://github.com/alindeman))
- Add supported fields to POST /incident request. [\#151](https://github.com/PagerDuty/go-pagerduty/pull/151) ([archydragon](https://github.com/archydragon))
- Consolidated community contributions [\#150](https://github.com/PagerDuty/go-pagerduty/pull/150) ([dobs](https://github.com/dobs))
- Incident alerts [\#143](https://github.com/PagerDuty/go-pagerduty/pull/143) ([soullivaneuh](https://github.com/soullivaneuh))
- Incident alerts [\#142](https://github.com/PagerDuty/go-pagerduty/pull/142) ([soullivaneuh](https://github.com/soullivaneuh))
- Remove useless else from README [\#140](https://github.com/PagerDuty/go-pagerduty/pull/140) ([soullivaneuh](https://github.com/soullivaneuh))
- Add V2 Event Management to the CLI [\#138](https://github.com/PagerDuty/go-pagerduty/pull/138) ([philnielsen](https://github.com/philnielsen))
- Fix incident struct to behave as API expects [\#137](https://github.com/PagerDuty/go-pagerduty/pull/137) ([DennyLoko](https://github.com/DennyLoko))
- Add ListContactMethods and GetContactMethod [\#132](https://github.com/PagerDuty/go-pagerduty/pull/132) ([amencarini](https://github.com/amencarini))
- Adding fields for incident id and priority [\#131](https://github.com/PagerDuty/go-pagerduty/pull/131) ([davidgibbons](https://github.com/davidgibbons))
- Add src to cd $GOPATH instruction [\#130](https://github.com/PagerDuty/go-pagerduty/pull/130) ([ryanhall07](https://github.com/ryanhall07))
- CreateIncident takes CreateIncidentOptions param [\#127](https://github.com/PagerDuty/go-pagerduty/pull/127) ([wdhnl](https://github.com/wdhnl))
- update CreateMaintenanceWindow [\#126](https://github.com/PagerDuty/go-pagerduty/pull/126) ([wdhnl](https://github.com/wdhnl))
- Add instructions for actually being able to install the CLI [\#123](https://github.com/PagerDuty/go-pagerduty/pull/123) ([whithajess](https://github.com/whithajess))
- Create Incident, List Priorities, and headers in POST method support [\#122](https://github.com/PagerDuty/go-pagerduty/pull/122) ([ldelossa](https://github.com/ldelossa))
- Allow package consumers to provide their own HTTP client [\#121](https://github.com/PagerDuty/go-pagerduty/pull/121) ([theckman](https://github.com/theckman))
- Add MergeIncidents \(using Incident\) [\#114](https://github.com/PagerDuty/go-pagerduty/pull/114) ([atomicules](https://github.com/atomicules))
- Updated incident.go to reflect current api documentation [\#113](https://github.com/PagerDuty/go-pagerduty/pull/113) ([averstappen](https://github.com/averstappen))
- Try out CircleCI. [\#109](https://github.com/PagerDuty/go-pagerduty/pull/109) ([felicianotech](https://github.com/felicianotech))
- Added From parameter to createNote function [\#108](https://github.com/PagerDuty/go-pagerduty/pull/108) ([Nnoromuche](https://github.com/Nnoromuche))
- Get a list of alerts for a given incident id. [\#106](https://github.com/PagerDuty/go-pagerduty/pull/106) ([pushkar-engagio](https://github.com/pushkar-engagio))
- Add oncall details [\#104](https://github.com/PagerDuty/go-pagerduty/pull/104) ([luqasn](https://github.com/luqasn))
- Add support for v2 ManageEvent api call [\#103](https://github.com/PagerDuty/go-pagerduty/pull/103) ([luqasn](https://github.com/luqasn))
- Reformatted Apache 2.0 license [\#99](https://github.com/PagerDuty/go-pagerduty/pull/99) ([joshdk](https://github.com/joshdk))
- Add ability to list a user's contact methods [\#97](https://github.com/PagerDuty/go-pagerduty/pull/97) ([facundoagriel](https://github.com/facundoagriel))
- Added json fields to structs [\#93](https://github.com/PagerDuty/go-pagerduty/pull/93) ([bradleyrobinson](https://github.com/bradleyrobinson))
- Get user's contact methods [\#91](https://github.com/PagerDuty/go-pagerduty/pull/91) ([wvdeutekom](https://github.com/wvdeutekom))
- Fixed spelling, entires-\>entries [\#78](https://github.com/PagerDuty/go-pagerduty/pull/78) ([lowesoftware](https://github.com/lowesoftware))
- Updating incident.go [\#75](https://github.com/PagerDuty/go-pagerduty/pull/75) ([domudall](https://github.com/domudall))
- Adding new fields to Vendor [\#74](https://github.com/PagerDuty/go-pagerduty/pull/74) ([domudall](https://github.com/domudall))
- Vendor CLI [\#73](https://github.com/PagerDuty/go-pagerduty/pull/73) ([domudall](https://github.com/domudall))
- Fixing structs within user.go [\#72](https://github.com/PagerDuty/go-pagerduty/pull/72) ([domudall](https://github.com/domudall))

## [1.0.4](https://github.com/PagerDuty/go-pagerduty/tree/1.0.4) (2018-05-28)

[Full Changelog](https://github.com/PagerDuty/go-pagerduty/compare/1.0.3...1.0.4)

## [1.0.3](https://github.com/PagerDuty/go-pagerduty/tree/1.0.3) (2018-05-28)

[Full Changelog](https://github.com/PagerDuty/go-pagerduty/compare/1.0.2...1.0.3)

## [1.0.2](https://github.com/PagerDuty/go-pagerduty/tree/1.0.2) (2018-05-28)

[Full Changelog](https://github.com/PagerDuty/go-pagerduty/compare/1.0.1...1.0.2)

**Merged pull requests:**

- Add gorleaser to release [\#118](https://github.com/PagerDuty/go-pagerduty/pull/118) ([mattstratton](https://github.com/mattstratton))

## [1.0.1](https://github.com/PagerDuty/go-pagerduty/tree/1.0.1) (2018-05-28)

[Full Changelog](https://github.com/PagerDuty/go-pagerduty/compare/1.0.0...1.0.1)

## [1.0.0](https://github.com/PagerDuty/go-pagerduty/tree/1.0.0) (2018-05-28)

[Full Changelog](https://github.com/PagerDuty/go-pagerduty/compare/d080263da74613ba3ac237baaf09f5f169b00d72...1.0.0)

**Fixed bugs:**

- Escalation Policy's repeat\_enabled Is Ignored [\#57](https://github.com/PagerDuty/go-pagerduty/issues/57)
- Problems running freshly built pd utility [\#39](https://github.com/PagerDuty/go-pagerduty/issues/39)
- Manage Incident gives error [\#32](https://github.com/PagerDuty/go-pagerduty/issues/32)
- Added missing slash to delete integration method url [\#59](https://github.com/PagerDuty/go-pagerduty/pull/59) ([reybard](https://github.com/reybard))

**Closed issues:**

- Trouble creating an integration [\#102](https://github.com/PagerDuty/go-pagerduty/issues/102)
- Client does not trigger events [\#101](https://github.com/PagerDuty/go-pagerduty/issues/101)
- Paging help [\#94](https://github.com/PagerDuty/go-pagerduty/issues/94)
- Help with incident creation API [\#89](https://github.com/PagerDuty/go-pagerduty/issues/89)
- Memory leak because of response body is not closed [\#66](https://github.com/PagerDuty/go-pagerduty/issues/66)
- Since and Until don't work for log\_entries [\#61](https://github.com/PagerDuty/go-pagerduty/issues/61)
- service: auto\_resolve\_timeout & acknowledgement\_timeout cannot be set to null [\#51](https://github.com/PagerDuty/go-pagerduty/issues/51)
- Possible to create new service and integration together [\#42](https://github.com/PagerDuty/go-pagerduty/issues/42)
- Documentation does not match code [\#16](https://github.com/PagerDuty/go-pagerduty/issues/16)
- Typo in repo description [\#15](https://github.com/PagerDuty/go-pagerduty/issues/15)
- Webhook decoder [\#14](https://github.com/PagerDuty/go-pagerduty/issues/14)
- incident\_key for create\_event [\#13](https://github.com/PagerDuty/go-pagerduty/issues/13)

**Merged pull requests:**

- Fix pagination for ListOnCalls [\#90](https://github.com/PagerDuty/go-pagerduty/pull/90) ([IainCole](https://github.com/IainCole))
- Revert "Fix inconsistency with some REST Options objects passed by reference …" [\#88](https://github.com/PagerDuty/go-pagerduty/pull/88) ([mimato](https://github.com/mimato))
- Adding travis config, fixup Makefile [\#87](https://github.com/PagerDuty/go-pagerduty/pull/87) ([mimato](https://github.com/mimato))
- Fixed invalid JSON descriptor for FirstTriggerLogEntry [\#86](https://github.com/PagerDuty/go-pagerduty/pull/86) ([mwisniewski0](https://github.com/mwisniewski0))
- \[incidents\] fix entries typo in a few places [\#85](https://github.com/PagerDuty/go-pagerduty/pull/85) ([joeyparsons](https://github.com/joeyparsons))
- Fix inconsistency with some REST Options objects passed by reference … [\#79](https://github.com/PagerDuty/go-pagerduty/pull/79) ([lowesoftware](https://github.com/lowesoftware))
- Explicit JSON reference to schedules [\#77](https://github.com/PagerDuty/go-pagerduty/pull/77) ([domudall](https://github.com/domudall))
- Adding AlertCreation to Service struct [\#76](https://github.com/PagerDuty/go-pagerduty/pull/76) ([domudall](https://github.com/domudall))
- Add support for escalation rules [\#71](https://github.com/PagerDuty/go-pagerduty/pull/71) ([heimweh](https://github.com/heimweh))
- Fix maintenance window JSON [\#69](https://github.com/PagerDuty/go-pagerduty/pull/69) ([domudall](https://github.com/domudall))
- Fixing Maintenance typo [\#68](https://github.com/PagerDuty/go-pagerduty/pull/68) ([domudall](https://github.com/domudall))
- Update event.go - fix a memory leak [\#65](https://github.com/PagerDuty/go-pagerduty/pull/65) ([AngelRefael](https://github.com/AngelRefael))
- Add query to vendor [\#64](https://github.com/PagerDuty/go-pagerduty/pull/64) ([heimweh](https://github.com/heimweh))
- Fix JSON decode \(errorObject\) [\#63](https://github.com/PagerDuty/go-pagerduty/pull/63) ([heimweh](https://github.com/heimweh))
- fix since and until by adding them to url scheme [\#60](https://github.com/PagerDuty/go-pagerduty/pull/60) ([ethansommer](https://github.com/ethansommer))
- fix webhook struct member name [\#58](https://github.com/PagerDuty/go-pagerduty/pull/58) ([pgray](https://github.com/pgray))
- Incident - Add status field to incident [\#56](https://github.com/PagerDuty/go-pagerduty/pull/56) ([heimweh](https://github.com/heimweh))
- enable fetch log entries via incident api [\#55](https://github.com/PagerDuty/go-pagerduty/pull/55) ([flyinprogrammer](https://github.com/flyinprogrammer))
- Allow service timeouts to be disabled [\#53](https://github.com/PagerDuty/go-pagerduty/pull/53) ([heimweh](https://github.com/heimweh))
- Schedule restriction - Add support for start\_day\_of\_week [\#52](https://github.com/PagerDuty/go-pagerduty/pull/52) ([heimweh](https://github.com/heimweh))
- Add vendor support [\#49](https://github.com/PagerDuty/go-pagerduty/pull/49) ([heimweh](https://github.com/heimweh))
- Add schedules listing [\#46](https://github.com/PagerDuty/go-pagerduty/pull/46) ([Marc-Morata-Fite](https://github.com/Marc-Morata-Fite))
- dont declare main twice in examples [\#45](https://github.com/PagerDuty/go-pagerduty/pull/45) ([ranjib](https://github.com/ranjib))
- add service show [\#44](https://github.com/PagerDuty/go-pagerduty/pull/44) ([cmluciano](https://github.com/cmluciano))
- \(feat\)implement integration creation [\#43](https://github.com/PagerDuty/go-pagerduty/pull/43) ([ranjib](https://github.com/ranjib))
- \(chore\)add create event example [\#41](https://github.com/PagerDuty/go-pagerduty/pull/41) ([ranjib](https://github.com/ranjib))
- \(bug\)Add test. fix version issue [\#40](https://github.com/PagerDuty/go-pagerduty/pull/40) ([ranjib](https://github.com/ranjib))
- Remove subdomain argument from escalation\_policy example. [\#38](https://github.com/PagerDuty/go-pagerduty/pull/38) ([cmluciano](https://github.com/cmluciano))
- Skip JSON encoding if no payload was given [\#37](https://github.com/PagerDuty/go-pagerduty/pull/37) ([heimweh](https://github.com/heimweh))
- \(feat\)add ability API and CLI [\#36](https://github.com/PagerDuty/go-pagerduty/pull/36) ([ranjib](https://github.com/ranjib))
- Make updates to Escalation Policies work [\#35](https://github.com/PagerDuty/go-pagerduty/pull/35) ([heimweh](https://github.com/heimweh))
- Fix misspelling in User struct and add JSON tags [\#34](https://github.com/PagerDuty/go-pagerduty/pull/34) ([heimweh](https://github.com/heimweh))
- \(bug\)allow passing headers in http do call. fix manage incident call [\#33](https://github.com/PagerDuty/go-pagerduty/pull/33) ([ranjib](https://github.com/ranjib))
- \(chore\)get rid of logrus from all core structs except CLI entries. fix schedule override command [\#31](https://github.com/PagerDuty/go-pagerduty/pull/31) ([ranjib](https://github.com/ranjib))
- \(bug\)rename override struct [\#30](https://github.com/PagerDuty/go-pagerduty/pull/30) ([ranjib](https://github.com/ranjib))
- \(bug\)implement schedule override [\#29](https://github.com/PagerDuty/go-pagerduty/pull/29) ([ranjib](https://github.com/ranjib))
- fix misspelling in trigger\_summary\_data's JSON key. [\#28](https://github.com/PagerDuty/go-pagerduty/pull/28) ([tomwans](https://github.com/tomwans))
- Correctly set meta flag for incident list [\#26](https://github.com/PagerDuty/go-pagerduty/pull/26) ([afirth](https://github.com/afirth))
- Add \*.swp to gitignore [\#25](https://github.com/PagerDuty/go-pagerduty/pull/25) ([afirth](https://github.com/afirth))
- Support the /oncalls endpoint in the CLI [\#24](https://github.com/PagerDuty/go-pagerduty/pull/24) ([afirth](https://github.com/afirth))
- Refactor to work correctly with V2 API [\#23](https://github.com/PagerDuty/go-pagerduty/pull/23) ([dthagard](https://github.com/dthagard))
- \(feat\)Add webhook decoding capability [\#22](https://github.com/PagerDuty/go-pagerduty/pull/22) ([ranjib](https://github.com/ranjib))
- \(chore\)Decode event API response.  [\#21](https://github.com/PagerDuty/go-pagerduty/pull/21) ([ranjib](https://github.com/ranjib))
- \(bug\)add incident\_key field in event api client [\#20](https://github.com/PagerDuty/go-pagerduty/pull/20) ([ranjib](https://github.com/ranjib))
- \(chore\)nuke sub domain, v2 api does not need one [\#19](https://github.com/PagerDuty/go-pagerduty/pull/19) ([ranjib](https://github.com/ranjib))
- Implement list users CLI [\#17](https://github.com/PagerDuty/go-pagerduty/pull/17) ([ranjib](https://github.com/ranjib))
- Add team\_ids\[\] query string arg [\#12](https://github.com/PagerDuty/go-pagerduty/pull/12) ([marklap](https://github.com/marklap))
- Incidents fix [\#11](https://github.com/PagerDuty/go-pagerduty/pull/11) ([jareksm](https://github.com/jareksm))
- Added APIListObject to Option types to allow setting offset and [\#10](https://github.com/PagerDuty/go-pagerduty/pull/10) ([jareksm](https://github.com/jareksm))
- fix typo [\#9](https://github.com/PagerDuty/go-pagerduty/pull/9) ([sjansen](https://github.com/sjansen))
- implement incident list cli. event posting api [\#8](https://github.com/PagerDuty/go-pagerduty/pull/8) ([ranjib](https://github.com/ranjib))
- CLI for create escalation policy, maintainenance window , schedule ov… [\#7](https://github.com/PagerDuty/go-pagerduty/pull/7) ([ranjib](https://github.com/ranjib))
- \(feat\)implement create service cli [\#6](https://github.com/PagerDuty/go-pagerduty/pull/6) ([ranjib](https://github.com/ranjib))
- \(feat\)list service cli [\#5](https://github.com/PagerDuty/go-pagerduty/pull/5) ([ranjib](https://github.com/ranjib))
- \(feat\)implement addon update/delete [\#4](https://github.com/PagerDuty/go-pagerduty/pull/4) ([ranjib](https://github.com/ranjib))
- \(feat\)Show addon cli [\#3](https://github.com/PagerDuty/go-pagerduty/pull/3) ([ranjib](https://github.com/ranjib))
- \(feat\) addon list api. create cli [\#2](https://github.com/PagerDuty/go-pagerduty/pull/2) ([ranjib](https://github.com/ranjib))
- \(chore\) list addon [\#1](https://github.com/PagerDuty/go-pagerduty/pull/1) ([ranjib](https://github.com/ranjib))



\* *This Changelog was automatically generated by [github_changelog_generator](https://github.com/github-changelog-generator/github-changelog-generator)*
