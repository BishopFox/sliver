package mtypes

const (
	APIKeysEndpoint           = "keys"
	APIKeysRegenerateEndpoint = APIKeysEndpoint + "/public"
	APIKeysVersion            = 1
)

type GetAPIKeyListResponse struct {
	Items []APIKey `json:"items"`
}

type CreateAPIKeyResponse struct {
	Key APIKey `json:"key"`
}

type DeleteAPIKeyResponse struct {
	Message string `json:"message"`
}

type RegeneratePublicAPIKeyResponse struct {
	Key     string `json:"key"`
	Message string `json:"message"`
}

type APIKey struct {
	ID             string      `json:"id"`
	Description    string      `json:"description"`
	Kind           string      `json:"kind"`
	Role           string      `json:"role"`
	CreatedAt      ISO8601Time `json:"created_at"`
	UpdatedAt      ISO8601Time `json:"updated_at"`
	DomainName     string      `json:"domain_name"`
	Requestor      string      `json:"requestor"`
	UserName       string      `json:"user_name"`
	IsDisabled     bool        `json:"is_disabled"`
	ExpiresAt      ISO8601Time `json:"expires_at"`
	Secret         string      `json:"secret"`
	DisabledReason string      `json:"disabled_reason"`
}
