// SPDX-License-Identifier: Apache-2.0

package models

import "net/url"

// URLHasCredentials reports whether a remote URL contains embedded
// credentials (password or token in the userinfo). SSH URLs with
// just a username (e.g. "git@github.com") are NOT flagged.
//
// Flagged: https://user:token@github.com/...
// Not flagged: ssh://git@github.com/..., git@github.com:...
func URLHasCredentials(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	// Only HTTP(S) URLs can carry credentials in userinfo.
	if u.Scheme != "https" && u.Scheme != "http" {
		return false
	}

	if u.User == nil {
		return false
	}

	// A password (or token) in the URL is the problem.
	_, hasPassword := u.User.Password()

	return hasPassword
}

// StripURLCredentials returns the URL with credentials removed from userinfo.
// The username is preserved (it's often needed for authentication), but
// the password/token is removed.
func StripURLCredentials(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	if u.User == nil {
		return rawURL
	}

	// Keep the username, drop the password.
	u.User = url.User(u.User.Username())

	return u.String()
}
