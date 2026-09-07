package plivo

type PhoneNumberComplianceLinkService struct {
	client *Client
}

type PhoneNumberComplianceLinkNumber struct {
	Number                  string `json:"number"`
	ComplianceApplicationID string `json:"compliance_application_id"`
}

type PhoneNumberComplianceBulkLinkParams struct {
	Numbers []PhoneNumberComplianceLinkNumber `json:"numbers"`
}

type PhoneNumberComplianceLinkReport struct {
	Number  string `json:"number"`
	Status  string `json:"status"`
	Remarks string `json:"remarks"`
}

type PhoneNumberComplianceBulkLinkResponse struct {
	ApiID        string                            `json:"api_id"`
	TotalCount   int                               `json:"total_count"`
	UpdatedCount int                               `json:"updated_count"`
	Report       []PhoneNumberComplianceLinkReport `json:"report"`
}

func (service *PhoneNumberComplianceLinkService) BulkLink(params PhoneNumberComplianceBulkLinkParams) (response *PhoneNumberComplianceBulkLinkResponse, err error) {
	req, err := service.client.NewRequest("POST", params, "PhoneNumber/Compliance/Link")
	if err != nil {
		return
	}
	response = &PhoneNumberComplianceBulkLinkResponse{}
	err = service.client.ExecuteRequest(req, response)
	return
}
