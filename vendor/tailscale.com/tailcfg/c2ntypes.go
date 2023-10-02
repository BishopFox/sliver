// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

// c2n (control-to-node) API types.

package tailcfg

// C2NSSHUsernamesRequest is the request for the /ssh/usernames.
// A GET request without a request body is equivalent to the zero value of this type.
// Otherwise, a POST request with a JSON-encoded request body is expected.
type C2NSSHUsernamesRequest struct {
	// Exclude optionally specifies usernames to exclude
	// from the response.
	Exclude map[string]bool `json:",omitempty"`

	// Max is the maximum number of usernames to return.
	// If zero, a default limit is used.
	Max int `json:",omitempty"`
}

// C2NSSHUsernamesResponse is the response (from node to control) from the
// /ssh/usernames handler.
//
// It returns username auto-complete suggestions for a user to SSH to this node.
// It's only shown to people who already have SSH access to the node. If this
// returns multiple usernames, only the usernames that would have access per the
// tailnet's ACLs are shown to the user so as to not leak the existence of
// usernames.
type C2NSSHUsernamesResponse struct {
	// Usernames is the list of usernames to suggest. If the machine has many
	// users, this list may be truncated. If getting the list of usernames might
	// be too slow or unavailable, this list might be empty. This is effectively
	// just a best effort set of hints.
	Usernames []string
}

// C2NUpdateResponse is the response (from node to control) from the /update
// handler. It tells control the status of its request for the node to update
// its Tailscale installation.
type C2NUpdateResponse struct {
	// Err is the error message, if any.
	Err string `json:",omitempty"`

	// Enabled indicates whether the user has opted in to updates triggered from
	// control.
	Enabled bool

	// Supported indicates whether remote updates are supported on this
	// OS/platform.
	Supported bool

	// Started indicates whether the update has started.
	Started bool
}
