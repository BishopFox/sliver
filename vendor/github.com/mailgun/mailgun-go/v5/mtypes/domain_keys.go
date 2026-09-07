package mtypes

type ListAllDomainsKeysResponse struct {
	TotalCount int         `json:"total_count"`
	Items      []DomainKey `json:"items"`
	Paging     Paging      `json:"paging"`
}

type ListDomainKeysResponse struct {
	Items  []DomainKey `json:"items"`
	Paging Paging      `json:"paging"` // NOTE: API does not return this as of now.
}

type UpdateDomainDkimAuthorityResponse struct {
	Message           string      `json:"message"`
	SendingDNSRecords []DNSRecord `json:"sending_dns_records"`
	Changed           bool        `json:"changed"`
}

type DomainKey struct {
	SigningDomain string    `json:"signing_domain"`
	Selector      string    `json:"selector"`
	DNSRecord     DNSRecord `json:"dns_record"`
}
