# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [4.6.1] - 2022-03-29
### Changed
* Validate domain passed to Send()

## [4.6.0] - 2021-11-17
### Changed
* Add a struct and a function to parse route forward hook payloads.
* Add missing VerifyAndReturnDomain method to the interface.

## [4.5.2] - 2021-06-28
### Changed
* Change response for VerifyDomain method from (string, error) to (DomainResponse, error)

## [4.5.1] - 2021-04-12
### Changed
* `MockServer` now locks internal resources for thread safety.

## [4.5.0] - 2021-04-09
### Changed
* `MockServer` is now an interface, `NewMockServer()` now returns the interface.

## [4.4.1] - 2021-02-23
### Changed
* Removed dependency on github.com/go-chi/chi

## [4.4.0] - 2021-02-22
### Changed
* Added `SetTrackingOptions()` to suppport htmlonly option

## [4.3.4] - 2021-02-22
### Changed
* Paging through results now clears the previous result before new fetch
* Added test for multiple attachments

## [4.3.3] - 2021-01-29
### Added
* Added UpdateDomainTrackingWebPrefix()
* Add the `Risk` to the EmailVerification response attributes

## [4.3.2] - 2021-01-19
### Added
* Added UpdateDomainDkimSelector()

## [4.3.1] - 2021-01-08
### Changed
* Remove mustache from our list of supported template engines

## [4.3.0] - 2020-10-14
### Changed
* Replaced easyjson with json-iterator when marshalling events
* Modified the mailgun.Event interface by removing the Marshaller interface from easyjson.
* Fixed failure while testing webhook via mailgun web console

## [4.2.0] - 2020-09-17
### Added
* Added ListEventsWithDomain()

## [4.1.4] - 2020-08-20
### Changes
* Added Storage to Accepted, Delivered and Failed events

## [4.1.3] - 2020-06-23
### Changes
* UpdateTemplateVersion() now including the template html in the payload request

## [4.1.2] - 2020-06-10
### Added
* Added DeleteBounceList method

## [4.1.1] - 2020-06-05
### Changed
* Nows sets initial tag when creating a new template

## [4.1.0] - 2020-04-23
### Changed
* Added EmailVerification.reason is now a []string (Fixes #217)

## [4.0.1] - 2020-03-10
### Added
* Added SetTemplateVersion and SetTemplateRenderText methods to Message

## [4.0.0] - 2020-01-27
### Changes
* Changed `UserVariables` type from `map[string]interface{}` to `interface{}`
  to handle truncated user-variable messages in events.
### Added
* Add support for setting AMP content in messages

## [3.6.3] - 2019-12-03
### Changes
* Calls to get stats now use epoch as the time format

## [3.6.2] - 2019-11-18
### Added
* Added `AddTemplateVariable()` to make adding variables to templates 
  less confusing and error prone.

## [3.6.1] - 2019-10-24
### Added
* Added `VerifyWebhookSignature()` to mailgun interface

## [3.6.1-rc.3] - 2019-07-16
### Added
* APIBaseEU and APIBaseUS to help customers change regions
* Documented how to change regions in the README

## [3.6.1-rc.2] - 2019-07-01
### Changes
* Fix the JSON response for `GetMember()`
* Typo in format string in max number of tags error

## [3.6.0] - 2019-06-26
### Added
* Added UpdateClickTracking() to modify click tracking for a domain
* Added UpdateUnsubscribeTracking() to modify unsubscribe tracking for a domain
* Added UpdateOpenTracking() to modify open tracking for a domain

## [3.5.0] - 2019-05-21
### Added
* Added notice in README about go dep bug.
* Added endpoints for webhooks in mock server
### Changes
* Change names of some parameters on public methods to make their use clearer.
* Changed signature of `GetWebhook()` now returns []string.
* Changed signature of `ListWebhooks()` now returns map[string][]string.
* Both `GetWebhooks()` and `ListWebhooks()` now handle new and legacy webhooks properly.

## [3.4.0] - 2019-04-23
### Added
* Added `Message.SetTemplate()` to allow sending with the body of a template.
### Changes
* Changed signature of `CreateDomain()` moved password into `CreateDomainOptions`

## [3.4.0] - 2019-04-23
### Added
* Added `Message.SetTemplate()` to allow sending with the body of a template.
### Changes
* Changed signature of `CreateDomain()` moved password into `CreateDomainOptions`

## [3.3.2] - 2019-03-28
### Changes
* Uncommented DeliveryStatus.Code and change it to an integer (See #175)
* Added UserVariables to all Message events (See #176)

## [3.3.1] - 2019-03-13
### Changes
* Updated Template calls to reflect the most recent Template API changes.
* GetStoredMessage() now accepts a URL instead of an id
* Deprecated GetStoredMessageForURL()
* Deprecated GetStoredMessageRawForURL()
* Fixed GetUnsubscribed()

### Added
* Added `GetStoredAttachment()`

### Removed
* Method `DeleteStoredMessage()` mailgun API no long allows this call

## [3.3.0] - 2019-01-28
### Changes
* Changed signature of CreateDomain() Now returns JSON response
* Changed signature of GetDomain() Now returns a single DomainResponse
* Clarified installation notes for non golang module users
* Changed 'Public Key' to 'Public Validation Key' in readme
* Fixed issue with Next() for limit/skip based iterators

### Added
* Added VerifyDomain()

## [3.2.0] - 2019-01-21
### Changes
* Deprecated mg.VerifyWebhookRequest()

### Added
* Added mailgun.ParseEvent()
* Added mailgun.ParseEvents()
* Added mg.VerifyWebhookSignature()


## [3.1.0] - 2019-01-16
### Changes
* Removed context.Context from ListDomains() signature
* ListEventOptions.Begin and End are no longer pointers to time.Time

### Added
* Added mg.ReSend() to public Mailgun interface
* Added Message.SetSkipVerification()
* Added Message.SetRequireTLS()

## [3.0.0] - 2019-01-15
### Added
* Added CHANGELOG
* Added `AddDomainIP()`
* Added `ListDomainIPS()`
* Added `DeleteDomainIP()`
* Added `ListIPS()`
* Added `GetIP()`
* Added `GetDomainTracking()`
* Added `GetDomainConnection()`
* Added `UpdateDomainConnection()`
* Added `CreateExport()`
* Added `ListExports()`
* Added `GetExports()`
* Added `GetExportLink()`
* Added `CreateTemplate()`
* Added `GetTemplate()`
* Added `UpdateTemplate()`
* Added `DeleteTemplate()`
* Added `ListTemplates()`
* Added `AddTemplateVersion()`
* Added `GetTemplateVersion()`
* Added `UpdateTemplateVersion()`
* Added `DeleteTemplateVersion()`
* Added `ListTemplateVersions()`

### Changed
* Added a `mailgun.MockServer` which duplicates part of the mailgun API; suitable for testing
* `ListMailingLists()` now uses the `/pages` API and returns an iterator
* `ListMembers()` now uses the `/pages` API and returns an iterator
* Renamed public interface methods to be consistent. IE: `GetThing(), ListThing(), CreateThing()`
* Moved event objects into the `mailgun/events` package, so names like
  `MailingList` returned by API calls and `MailingList` as an event object
  don't conflict and confuse users.
* Now using context.Context for all network operations
* Test suite will run without MG_ env vars defined
* ListRoutes() now uses the iterator interface
* Added SkipNetworkTest()
* Renamed GetStatsTotals() to GetStats()
* Renamed GetUnsubscribes to ListUnsubscribes()
* Renamed Unsubscribe() to CreateUnsubscribe()
* Renamed RemoveUnsubscribe() to DeleteUnsubscribe()
* GetStats() now takes an `*opt` argument to pass optional parameters
* Modified GetUnsubscribe() to follow the API
* Now using golang modules
* ListCredentials() now returns an iterator
* ListUnsubscribes() now returns an paging iterator
* CreateDomain now accepts CreateDomainOption{}
* CreateDomain() now supports all optional parameters not just spam_action and wildcard.
* ListComplaints() now returns a page iterator
* Renamed `TagItem` to `Tag`
* ListBounces() now returns a page iterator
* API responses with CreatedAt fields are now unmarshalled into RFC2822
* DomainList() now returns an iterator
* Updated godoc documentation
* Renamed ApiBase to APIBase
* Updated copyright to 2019
* `ListEvents()` now returns a list of typed events

### Removed
* Removed more deprecated types
* Removed gobuffalo/envy dependency
* Remove mention of the CLI in the README
* Removed mailgun cli from project
* Removed GetCode() from `Bounce` struct. Verified API returns 'string' and not 'int'
* Removed deprecated methods NewMessage and NewMIMEMessage
* Removed ginkgo and gomega tests
* Removed GetStats() As the /stats endpoint is depreciated
