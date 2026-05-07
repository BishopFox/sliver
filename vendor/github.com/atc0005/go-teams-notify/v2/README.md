<!-- omit in toc -->
# goteamsnotify

A package to send messages to a Microsoft Teams channel.

[![Latest release][githubtag-image]][githubtag-url]
[![Go Reference][goref-image]][goref-url]
[![License][license-image]][license-url]
[![go.mod Go version](https://img.shields.io/github/go-mod/go-version/atc0005/go-teams-notify)](https://github.com/atc0005/go-teams-notify)
[![Lint and Build](https://github.com/atc0005/go-teams-notify/actions/workflows/lint-and-build.yml/badge.svg)](https://github.com/atc0005/go-teams-notify/actions/workflows/lint-and-build.yml)
[![Project Analysis](https://github.com/atc0005/go-teams-notify/actions/workflows/project-analysis.yml/badge.svg)](https://github.com/atc0005/go-teams-notify/actions/workflows/project-analysis.yml)

<!-- omit in toc -->
## Table of contents

- [Project home](#project-home)
- [Overview](#overview)
- [Features](#features)
- [Project Status](#project-status)
- [Supported Releases](#supported-releases)
  - [Plans: v2](#plans-v2)
  - [Plans: v3](#plans-v3)
- [Changelog](#changelog)
- [Usage](#usage)
  - [Add this project as a dependency](#add-this-project-as-a-dependency)
  - [Setup a connection to Microsoft Teams](#setup-a-connection-to-microsoft-teams)
    - [Overview](#overview-1)
    - [Workflow connectors](#workflow-connectors)
      - [Workflow webhook URL format](#workflow-webhook-url-format)
      - [How to create a Workflow connector webhook URL](#how-to-create-a-workflow-connector-webhook-url)
        - [Using Teams client Workflows context option](#using-teams-client-workflows-context-option)
        - [Using Teams client app](#using-teams-client-app)
        - [Using Power Automate web UI](#using-power-automate-web-ui)
    - [O365 connectors](#o365-connectors)
      - [O365 webhook URL format](#o365-webhook-url-format)
      - [How to create an O365 connector webhook URL](#how-to-create-an-o365-connector-webhook-url)
  - [Examples](#examples)
    - [Basic](#basic)
    - [Specify proxy server](#specify-proxy-server)
    - [User Mention](#user-mention)
    - [CodeBlock](#codeblock)
    - [Tables](#tables)
    - [Set custom user agent](#set-custom-user-agent)
    - [Add an Action](#add-an-action)
    - [Toggle visibility](#toggle-visibility)
    - [Disable webhook URL prefix validation](#disable-webhook-url-prefix-validation)
    - [Enable custom patterns' validation](#enable-custom-patterns-validation)
- [Used by](#used-by)
- [References](#references)

## Project home

See [our GitHub repo](https://github.com/atc0005/go-teams-notify) for the
latest code, to file an issue or submit improvements for review and potential
inclusion into the project.

## Overview

The `goteamsnotify` package (aka, `go-teams-notify`) allows sending messages
to a Microsoft Teams channel. These messages can be composed of
[🚫 deprecated][o365-connector-retirement-announcement] legacy
[`MessageCard`][msgcard-ref] or [`Adaptive Card`][adaptivecard-ref] card
formats.

Simple messages can be created by specifying only a title and a text body.
More complex messages may be composed of multiple sections ([🚫
deprecated][o365-connector-retirement-announcement] `MessageCard`) or
containers (`Adaptive Card`), key/value pairs (aka, `Facts`) and externally
hosted images. See the [Features](#features) list for more information.

**NOTE**: `Adaptive Card` support is currently limited. The goal is to expand
this support in future releases to include additional features supported by
Microsoft Teams.

## Features

- Submit simple or complex messages to Microsoft Teams
  - simple messages consist of only a title and a text body (one or more
    strings)
  - complex messages may consist of multiple sections ([🚫
    deprecated][o365-connector-retirement-announcement] `MessageCard`),
    containers (`Adaptive Card`) key/value pairs (aka, `Facts`) and externally
    hosted images
- Support for Actions, allowing users to take quick actions within Microsoft
  Teams
  - [🚫 deprecated][o365-connector-retirement-announcement] [`MessageCard` `Actions`][msgcard-ref-actions]
  - [`Adaptive Card` `Actions`][adaptivecard-ref-actions]
- Support for [user mentions][adaptivecard-user-mentions] (`Adaptive
  Card` format)
- Configurable validation of webhook URLs
  - enabled by default, attempts to match most common known webhook URL
    patterns
  - option to disable validation entirely
  - option to use custom validation patterns
- Configurable validation of [🚫
  deprecated][o365-connector-retirement-announcement] `MessageCard` type
  - default assertion that bare-minimum required fields are present
  - support for providing a custom validation function to override default
    validation behavior
- Configurable validation of `Adaptive Card` type
  - default assertion that bare-minimum required fields are present
  - support for providing a custom validation function to override default
    validation behavior
- Configurable timeouts
- Configurable retry support

## Project Status

In short:

- The upstream project is no longer being actively developed or maintained.
- This fork is now a standalone project, accepting contributions, bug reports
  and feature requests.
  - see [Supported Releases](#supported-releases) for details
- Others have also taken an interest in [maintaining their own
  forks](https://github.com/atc0005/go-teams-notify/network/members) of the
  original project. See those forks for other ideas/changes that you may find
  useful.

For more details, see the
[Releases](https://github.com/atc0005/go-teams-notify/releases) section or our
[Changelog](https://github.com/atc0005/go-teams-notify/blob/master/CHANGELOG.md).

## Supported Releases

| Series   | Example          | Status                                  |
| -------- | ---------------- | --------------------------------------- |
| `v1.x.x` | `v1.3.1`         | Not Supported (EOL)                     |
| `v2.x.x` | `v2.6.0`         | Supported (until approximately 2026-01) |
| `v3.x.x` | `v3.0.0-alpha.1` | Planning (target 2026-01)               |
| `v4.x.x` | `v4.0.0-alpha.1` | TBD                                     |

### Plans: v2

| Task                                                         | Start Date / Version | Status   |
| ------------------------------------------------------------ | -------------------- | -------- |
| support the v2 branch with bugfixes and minor changes        | 2020-03-29 (v2.0.0)  | Ongoing  |
| add support & documentation for Power Automate workflow URLs | v2.11.0-alpha.1      | Complete |

### Plans: v3

Early January 2026:

- Microsoft [drops support for O365
  connectors][o365-connector-retirement-announcement] in December 2025
- we release a v3 branch
  - drop support for the [🚫
deprecated][o365-connector-retirement-announcement] O365 connectors
  - drop support for the [🚫
deprecated][o365-connector-retirement-announcement] `MessageCard`) format
- we drop support for the v2 branch
  - the focus would be on maintaining the v3 branch with bugfixes and minor
    changes

> [!NOTE]
>
> While the plan for the upcoming v3 series includes dropping support for the
[🚫 deprecated][o365-connector-retirement-announcement] `MessageCard` format
and O365 connectors, the focus would not be on refactoring the overall code
structure; many of the rough edges currently present in the API would remain
in the v3 series and await a more focused cleanup effort made in preparation
for a future v4 series.

## Changelog

See the [`CHANGELOG.md`](CHANGELOG.md) file for the changes associated with
each release of this application. Changes that have been merged to `master`,
but not yet an official release may also be noted in the file under the
`Unreleased` section. A helpful link to the Git commit history since the last
official release is also provided for further review.

## Usage

### Add this project as a dependency

See the [Examples](#examples) section for more details.

### Setup a connection to Microsoft Teams

#### Overview

> [!WARNING]
>
> Microsoft announced July 3rd, 2024 that Office 365 (O365) connectors within
Microsoft Teams would be [retired in 3
months][o365-connector-retirement-announcement] and replaced by Power Automate
workflows (or just "Workflows" for short).

Quoting from the microsoft365dev blog:

> We will gradually roll out this change in waves:
>
> - Wave 1 - effective August 15th, 2024: All new Connector creation will be
>   blocked within all clouds
> - Wave 2 - effective October 1st, 2024: All connectors within all clouds
>   will stop working

[Microsoft later changed some of the
details][o365-connector-retirement-announcement] regarding the retirement
timeline of O365 connectors:

> Update 07/23/2024: We understand and appreciate the feedback that customers
> have shared with us regarding the timeline provided for the migration from
> Office 365 connectors. We have extended the retirement timeline through
> December 2025 to provide ample time to migrate to another solution such as
> Power Automate, an app within Microsoft Teams, or Microsoft Graph. Please
> see below for more information about the extension:
>
> - All existing connectors within all clouds will continue to work until
>   December 2025, however using connectors beyond December 31, 2024 will
>   require additional action.
>   - Connector owners will be required to update the respective URL to post
>     by December 31st, 2024. At least 90 days prior to the December 31, 2024
>     deadline, we will send further guidance about making this URL update. If
>     the URL is not updated by December 31, 2024 the connector will stop
>     working. This is due to further service hardening updates being
>     implemented for Office 365 connectors in alignment with Microsoft’s
>     [Secure Future
>     Initiative](https://blogs.microsoft.com/blog/2024/05/03/prioritizing-security-above-all-else/)
> - Starting August 15th, 2024 all new creations should be created using the
>   Workflows app in Microsoft Teams

Since O365 connectors will likely persist in many environments until the very
end of the deprecation period this project will [continue to support
them](#supported-releases) until then alongside Power Automate workflows.

#### Workflow connectors

##### Workflow webhook URL format

Valid Power Automate Workflow URLs used to submit messages to Microsoft Teams
use this format:

- `https://*.logic.azure.com:443/workflows/GUID_HERE/triggers/manual/paths/invoke?api-version=YYYY-MM-DD&sp=%2Ftriggers%2Fmanual%2Frun&sv=1.0&sig=SIGNATURE_HERE`

Example URL from the LinkedIn [Bring Microsoft Teams incoming webhook security to
the next level with Azure Logic App][linkedin-teams-webhook-security-article]
article:

- `https://webhook-jenkins.azure-api.net/manual/paths/invoke?api-version=2016-10-01&sp=%2Ftriggers%2Fmanual%2Frun&sv=1.0&sig=f2QjZY50uoRnX6PIpyPT3xk`

##### How to create a Workflow connector webhook URL

> [!TIP]
>
> Use a dedicated "service" account not tied to a specific team member to help
ensure that the Workflow connector is long lived.

The [initial O365 retirement blog
post][o365-connector-retirement-announcement] provides a list of templates
which guide you through the process of creating a Power Automate Workflow
webhook URL.

###### Using Teams client Workflows context option

1. Navigate to a channel or chat
1. Select the ellipsis on the channel or chat
1. Select `Workflows`
1. Type `when a webhook request`
1. Select the appropriate template
   - `Post to a channel when a webhook request is received`
   - `Post to a chat when a webhook request is received`
1. Verify that `Microsoft Teams` is successfully enabled
1. Select `Next`
1. Select an appropriate value from the `Microsoft Teams Team` drop-down list.
1. Select an appropriate `Microsoft Teams Channel` drop-down list.
1. Select `Create flow`
1. Copy the new workflow URL
1. Select `Done`

###### Using Teams client app

1. Open `Workflows` application in teams
1. Select `Create` across the top of the UI
1. Choose `Notifications` at the left
1. Select `Post to a channel when a webhook request is received`
1. Verify that `Microsoft Teams` is successfully enabled
1. Select `Next`
1. Select an appropriate value from the `Microsoft Teams Team` drop-down list.
1. Select an appropriate `Microsoft Teams Channel` drop-down list.
1. Select `Create flow`
1. Copy the new workflow URL
1. Select `Done`

###### Using Power Automate web UI

[This][workflow-channel-post-from-webhook-request] template walks you through
the steps of creating a new Workflow using the
<https://make.powerautomate.com/> web UI:

1. Select or create a new connection (e.g., <user@example.com>) to Microsoft
   Teams
1. Select `Create`
1. Select an appropriate value from the `Microsoft Teams Team` drop-down list.
1. Select an appropriate `Microsoft Teams Channel` drop-down list.
1. Select `Create`
1. If prompted, read the info message (e.g., "Your flow is ready to go") and
   dismiss it.
1. Select `Edit` from the menu across the top
   - alternatively, select `My flows` from the side menu, then select `Edit`
     from the "More commands" ellipsis
1. Select `When a Teams webhook request is received` (e.g., left click)
1. Copy the `HTTP POST URL` value
   - this is your *private* custom Workflow connector URL
   - by default anyone can `POST` a request to this Workflow connector URL
     - while this access setting can be changed it will prevent this library
       from being used to submit webhook requests

#### O365 connectors

##### O365 webhook URL format

> [!WARNING]
>
> O365 connector webhook URLs are deprecated and [scheduled to be
retired][o365-connector-retirement-announcement] on 2024-10-01.

Valid (***deprecated***) O365 webhook URLs for Microsoft Teams use one of several
(confirmed) FQDNs patterns:

- `outlook.office.com`
- `outlook.office365.com`
- `*.webhook.office.com`
  - e.g., `example.webhook.office.com`

Using an O365 webhook URL with any of these FQDN patterns appears to give
identical results.

Here are complete, equivalent example webhook URLs from Microsoft's
documentation using the FQDNs above:

- <https://outlook.office.com/webhook/a1269812-6d10-44b1-abc5-b84f93580ba0@9e7b80c7-d1eb-4b52-8582-76f921e416d9/IncomingWebhook/3fdd6767bae44ac58e5995547d66a4e4/f332c8d9-3397-4ac5-957b-b8e3fc465a8c>
- <https://outlook.office365.com/webhook/a1269812-6d10-44b1-abc5-b84f93580ba0@9e7b80c7-d1eb-4b52-8582-76f921e416d9/IncomingWebhook/3fdd6767bae44ac58e5995547d66a4e4/f332c8d9-3397-4ac5-957b-b8e3fc465a8c>
- <https://example.webhook.office.com/webhookb2/a1269812-6d10-44b1-abc5-b84f93580ba0@9e7b80c7-d1eb-4b52-8582-76f921e416d9/IncomingWebhook/3fdd6767bae44ac58e5995547d66a4e4/f332c8d9-3397-4ac5-957b-b8e3fc465a8c>
  - note the `webhookb2` sub-URI specific to this FQDN pattern

All of these patterns when provided to this library should pass the default
validation applied. See the example further down for the option of disabling
webhook URL validation entirely.

##### How to create an O365 connector webhook URL

> [!WARNING]
>
> O365 connector webhook URLs are deprecated and [scheduled to be
retired][o365-connector-retirement-announcement] on 2024-10-01.

1. Open Microsoft Teams
1. Navigate to the channel where you wish to receive incoming messages from
   this application
1. Select `⋯` next to the channel name and then choose Connectors.
1. Scroll through the list of Connectors to Incoming Webhook, and choose Add.
1. Enter a name for the webhook, upload an image to associate with data from
   the webhook, and choose Create.
1. Copy the webhook URL to the clipboard and save it. You'll need the webhook
   URL for sending information to Microsoft Teams.
   - NOTE: While you can create another easily enough, you should treat this
     webhook URL as sensitive information as anyone with this unique URL is
     able to send messages (without authentication) into the associated
     channel.
1. Choose Done.

Credit:
[docs.microsoft.com](https://docs.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/connectors-using#setting-up-a-custom-incoming-webhook),
[gist comment from
shadabacc3934](https://gist.github.com/chusiang/895f6406fbf9285c58ad0a3ace13d025#gistcomment-3562501)

### Examples

#### Basic

This is an example of a simple client application which uses this library.

- `Adaptive Card`
  - File: [basic](./examples/adaptivecard/basic/main.go)
- [🚫 deprecated][o365-connector-retirement-announcement] `MessageCard`
  - File: [basic](./examples/messagecard/basic/main.go)

#### Specify proxy server

This is an example of a simple client application which uses this library to
route a generated message through a specified proxy server.

- `Adaptive Card`
  - File: [basic](./examples/adaptivecard/proxy/main.go)
- [🚫 deprecated][o365-connector-retirement-announcement] `MessageCard`
  - File: [basic](./examples/messagecard/proxy/main.go)

#### User Mention

These examples illustrates the use of one or more user mentions. This feature
is not available in the legacy [🚫
deprecated][o365-connector-retirement-announcement] `MessageCard` card format.

- File: [user-mention-single](./examples/adaptivecard/user-mention-single/main.go)
- File: [user-mention-multiple](./examples/adaptivecard/user-mention-multiple/main.go)
- File: [user-mention-verbose](./examples/adaptivecard/user-mention-verbose/main.go)
  - this example does not necessarily reflect an optimal implementation

#### CodeBlock

This example illustrates the use of a [`CodeBlock`][adaptivecard-codeblock].
This feature is not available in the legacy [🚫
deprecated][o365-connector-retirement-announcement] `MessageCard` card format.

- File: [codeblock](./examples/adaptivecard/codeblock/main.go)

#### Tables

These examples illustrates the use of a [`Table`][adaptivecard-table]. This
feature is not available in the legacy [🚫
deprecated][o365-connector-retirement-announcement] `MessageCard` card format.

- File: [table-manually-created](./examples/adaptivecard/table-manually-created/main.go)
- File: [table-unordered-grid](./examples/adaptivecard/table-unordered-grid/main.go)
- File: [table-with-headers](./examples/adaptivecard/table-with-headers/main.go)

#### Set custom user agent

This example illustrates setting a custom user agent.

- `Adaptive Card`
  - File: [custom-user-agent](./examples/adaptivecard/custom-user-agent/main.go)
- [🚫 deprecated][o365-connector-retirement-announcement] `MessageCard`
  - File: [custom-user-agent](./examples/messagecard/custom-user-agent/main.go)

#### Add an Action

This example illustrates adding an [`OpenUri`][msgcard-ref-actions] ([🚫
deprecated][o365-connector-retirement-announcement] `MessageCard`) or
[`OpenUrl`][adaptivecard-ref-actions] Action. When used, this action triggers
opening a URL in a separate browser or application.

- `Adaptive Card`
  - File: [actions](./examples/adaptivecard/actions/main.go)
- [🚫 deprecated][o365-connector-retirement-announcement] `MessageCard`
  - File: [actions](./examples/messagecard/actions/main.go)

#### Toggle visibility

These examples illustrates using
[`ToggleVisibility`][adaptivecard-ref-actions] Actions to control the
visibility of various Elements of an `Adaptive Card` message.

- File: [toggle-visibility-single-button](./examples/adaptivecard/toggle-visibility-single-button/main.go)
- File: [toggle-visibility-multiple-buttons](./examples/adaptivecard/toggle-visibility-multiple-buttons/main.go)
- File: [toggle-visibility-column-action](./examples/adaptivecard/toggle-visibility-column-action/main.go)
- File: [toggle-visibility-container-action](./examples/adaptivecard/toggle-visibility-container-action/main.go)

#### Disable webhook URL prefix validation

This example disables the validation webhook URLs, including the validation of
known prefixes so that custom/private webhook URL endpoints can be used (e.g.,
testing purposes).

- `Adaptive Card`
  - File: [disable-validation](./examples/adaptivecard/disable-validation/main.go)
- [🚫 deprecated][o365-connector-retirement-announcement] `MessageCard`
  - File: [disable-validation](./examples/messagecard/disable-validation/main.go)

#### Enable custom patterns' validation

This example demonstrates how to enable custom validation patterns for webhook
URLs.

- `Adaptive Card`
  - File: [custom-validation](./examples/adaptivecard/custom-validation/main.go)
- [🚫 deprecated][o365-connector-retirement-announcement] `MessageCard`
  - File: [custom-validation](./examples/messagecard/custom-validation/main.go)

## Used by

See the Known importers lists below for a dynamically updated list of projects
using either this library or the original project.

- [this fork](https://pkg.go.dev/github.com/atc0005/go-teams-notify/v2?tab=importedby)
- [original project](https://pkg.go.dev/github.com/dasrick/go-teams-notify/v2?tab=importedby)

## References

- [Original project](https://github.com/dasrick/go-teams-notify)
- [Forks of original project](https://github.com/atc0005/go-teams-notify/network/members)

<!--
  TODO: Refresh/replace these ref links after 2024-10-01 when O365 connectors are scheduled to be retired.
-->
- Microsoft Teams
  - Adaptive Cards
  ([de-de](https://docs.microsoft.com/de-de/outlook/actionable-messages/adaptive-card),
  [en-us](https://docs.microsoft.com/en-us/outlook/actionable-messages/adaptive-card))
  - O365 connectors
    - [Send via connectors](https://docs.microsoft.com/en-us/outlook/actionable-messages/send-via-connectors))
    - [Create Incoming Webhooks](https://learn.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/add-incoming-webhook)
  - [adaptivecards.io](https://adaptivecards.io/designer)
  - [Legacy actionable message card reference][msgcard-ref]
  - Workflow connectors
    - [Creating a workflow from a chat in Teams](https://support.microsoft.com/en-us/office/creating-a-workflow-from-a-channel-in-teams-242eb8f2-f328-45be-b81f-9817b51a5f0e)
    - [Creating a workflow from a channel in Teams](https://support.microsoft.com/en-us/office/creating-a-workflow-from-a-chat-in-teams-e3b51c4f-49de-40aa-a6e7-bcff96b99edc)

<!-- Footnotes here  -->

[o365-connector-retirement-announcement]: <https://devblogs.microsoft.com/microsoft365dev/retirement-of-office-365-connectors-within-microsoft-teams/> "Retirement of Office 365 connectors within Microsoft Teams"
[workflow-channel-post-from-webhook-request]: <https://make.preview.powerautomate.com/galleries/public/templates/d271a6f01c2545a28348d8f2cddf4c8f/post-to-a-channel-when-a-webhook-request-is-received> "Post to a channel when a webhook request is received"
[linkedin-teams-webhook-security-article]: <https://www.linkedin.com/pulse/bring-microsoft-teams-incoming-webhook-security-next-level-kinzelin> "Bring Microsoft Teams incoming webhook security to the next level with Azure Logic App"

[githubtag-image]: https://img.shields.io/github/release/atc0005/go-teams-notify.svg?style=flat
[githubtag-url]: https://github.com/atc0005/go-teams-notify

[goref-image]: https://pkg.go.dev/badge/github.com/atc0005/go-teams-notify/v2.svg
[goref-url]: https://pkg.go.dev/github.com/atc0005/go-teams-notify/v2

[license-image]: https://img.shields.io/github/license/atc0005/go-teams-notify.svg?style=flat
[license-url]: https://github.com/atc0005/go-teams-notify/blob/master/LICENSE

[msgcard-ref]: <https://docs.microsoft.com/en-us/outlook/actionable-messages/message-card-reference>
[msgcard-ref-actions]: <https://docs.microsoft.com/en-us/outlook/actionable-messages/message-card-reference#actions>

[adaptivecard-ref]: <https://adaptivecards.io/explorer>
[adaptivecard-ref-actions]: <https://docs.microsoft.com/en-us/adaptive-cards/authoring-cards/getting-started>
[adaptivecard-user-mentions]: <https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-format#mention-support-within-adaptive-cards>
[adaptivecard-table]: <https://adaptivecards.io/explorer/Table.html>

[adaptivecard-codeblock]: <https://learn.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-format?tabs=adaptive-md%2Cdesktop%2Cconnector-html#codeblock-in-adaptive-cards>

<!-- []: PLACEHOLDER "DESCRIPTION_HERE" -->
