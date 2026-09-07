package plivo

type PhoneNumberComplianceRequirementService struct {
	client *Client
}

type PhoneNumberComplianceRequirementGetParams struct {
	CountryISO string `json:"country_iso" url:"country_iso"`
	NumberType string `json:"number_type" url:"number_type"`
	UserType   string `json:"user_type" url:"user_type"`
}

type PhoneNumberComplianceRequirementResponse struct {
	ApiID         string                                      `json:"api_id"`
	RequirementID string                                      `json:"requirement_id"`
	CountryISO    string                                      `json:"country_iso"`
	NumberType    string                                      `json:"number_type"`
	UserType      string                                      `json:"user_type"`
	DocumentTypes []PhoneNumberComplianceRequirementDocType    `json:"document_types"`
}

type PhoneNumberComplianceRequirementDocType struct {
	DocumentTypeID string                                    `json:"document_type_id"`
	Name           string                                    `json:"name"`
	Description    string                                    `json:"description"`
	ProofRequired  bool                                      `json:"proof_required"`
	RequiredFields []PhoneNumberComplianceRequirementField    `json:"required_fields"`
}

type PhoneNumberComplianceRequirementField struct {
	FieldName    string `json:"field_name"`
	FriendlyName string `json:"friendly_name"`
	Description  string `json:"description,omitempty"`
	HelpText     string `json:"help_text,omitempty"`
	FieldType    string `json:"field_type"`
	Required     bool   `json:"required"`
	Format       string `json:"format,omitempty"`
	Enums        string `json:"enums,omitempty"`
	MinLength    int    `json:"min_length,omitempty"`
	MaxLength    int    `json:"max_length,omitempty"`
}

func (service *PhoneNumberComplianceRequirementService) Get(params PhoneNumberComplianceRequirementGetParams) (response *PhoneNumberComplianceRequirementResponse, err error) {
	req, err := service.client.NewRequest("GET", params, "PhoneNumber/Compliance/Requirements")
	if err != nil {
		return
	}
	response = &PhoneNumberComplianceRequirementResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}
