// Copyright 2022 Adam Chalkley
//
// https://github.com/atc0005/go-teams-notify
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

/*
Package validator provides logic to assist with validation tasks. The logic is
designed so that each subsequent validation step short-circuits after the
first validation failure; only the first validation failure is reported.

Credit to Fabrizio Milo for sharing the original implementation:

- https://stackoverflow.com/a/23960293/903870
- https://github.com/Mistobaan
*/
package validator
