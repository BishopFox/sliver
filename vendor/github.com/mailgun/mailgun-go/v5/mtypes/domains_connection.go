package mtypes

// TODO(v6): remove this file

type DomainConnectionResponse struct {
	Connection DomainConnection `json:"connection"`
}

// DomainConnection specifies the domain connection options.
type DomainConnection struct {
	RequireTLS       bool `json:"require_tls"`
	SkipVerification bool `json:"skip_verification"`
}
