// Copyright 2021 Adam Chalkley
//
// https://github.com/atc0005/go-teams-notify
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

/*
Package goteamsnotify is used to send messages to a Microsoft Teams channel.

# Project Home

See our GitHub repo (https://github.com/atc0005/go-teams-notify) for the
latest code, to file an issue or submit improvements for review and potential
inclusion into the project.

# Purpose

Send messages to a Microsoft Teams channel.

# Features

  - Submit messages to Microsoft Teams consisting of one or more sections,
    Facts (key/value pairs), Actions or images (hosted externally)
  - Support for MessageCard and Adaptive Card messages
  - Support for Actions, allowing users to take quick actions within Microsoft
    Teams
  - Support for user mentions
  - Configurable validation
  - Configurable timeouts
  - Configurable retry support
  - Support for overriding the default http.Client
  - Support for overriding the default project-specific user agent

# Usage

See our main README for supported settings and examples.
*/
package goteamsnotify
