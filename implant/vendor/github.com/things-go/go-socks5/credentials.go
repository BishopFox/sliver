package socks5

// CredentialStore is used to support user/pass authentication optional network addr
// if you want to limit user network addr,you can refuse it.
type CredentialStore interface {
	Valid(user, password, userAddr string) bool
}

// StaticCredentials enables using a map directly as a credential store
type StaticCredentials map[string]string

// Valid implement interface CredentialStore
func (s StaticCredentials) Valid(user, password, _ string) bool {
	pass, ok := s[user]
	return ok && password == pass
}
