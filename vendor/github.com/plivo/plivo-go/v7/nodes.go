package plivo

type NodeActionResponse struct {
	ApiID string `json:"api_id" url:"api_id"`
	Error string `json:"error" url:"error"`
}

type Node struct {
	ApiID     string `json:"api_id" url:"api_id"`
	NodeID    string `json:"node_id" url:"node_id"`
	PhloID    string `json:"phlo_id" url:"phlo_id"`
	Name      string `json:"name" url:"name"`
	NodeType  string `json:"node_type" url:"node_type"`
	CreatedOn string `json:"created_on" url:"created_on"`
}

type PhloMultiPartyCall struct {
	Node
	BaseResource
}

type PhloMultiPartyCallActionPayload struct {
	Action        string `json:"action" url:"action"`
	To            string `json:"to" url:"to"`
	Role          string `json:"role" url:"role"`
	TriggerSource string `json:"trigger_source" url:"trigger_source"`
}

func (self *PhloMultiPartyCall) update(params PhloMultiPartyCallActionPayload) (response *NodeActionResponse, err error) {
	req, err := self.client.NewRequest("POST", params, "phlo/%s/%s/%s", self.PhloID, self.NodeType,
		self.NodeID)
	if err != nil {
		return
	}
	response = &NodeActionResponse{}
	err = self.client.ExecuteRequest(req, response)

	return
}

func (self *PhloMultiPartyCall) Call(params PhloMultiPartyCallActionPayload) (*NodeActionResponse, error) {
	return self.update(params)
}

func (self *PhloMultiPartyCall) WarmTransfer(params PhloMultiPartyCallActionPayload) (response *NodeActionResponse,
	err error) {
	return self.update(params)
}

func (self *PhloMultiPartyCall) ColdTransfer(params PhloMultiPartyCallActionPayload) (response *NodeActionResponse,
	err error) {
	return self.update(params)
}

const HOLD = "hold"
const UNHOLD = "unhold"
const HANGUP = "hangup"
const RESUME_CALL = "resume_call"
const ABORT_TRANSFER = "abort_transfer"
const VOICEMAIL_DROP = "voicemail_drop"

type PhloMultiPartyCallMemberActionPayload struct {
	Action string `json:"action" url:"action"`
}

type PhloMultiPartyCallMember struct {
	NodeID        string `json:"node_id" url:"node_id"`
	PhloID        string `json:"phlo_id" url:"phlo_id"`
	NodeType      string `json:"node_type" url:"node_type"`
	MemberAddress string `json:"member_address" url:"member_address"`
	BaseResource
}

func (self *PhloMultiPartyCall) Member(memberID string) (response *PhloMultiPartyCallMember) {
	response = &PhloMultiPartyCallMember{self.NodeID, self.PhloID, self.NodeType, memberID, BaseResource{self.client}}
	return
}

func (self *PhloMultiPartyCallMember) AbortTransfer() (*NodeActionResponse, error) {
	return self.update(PhloMultiPartyCallMemberActionPayload{ABORT_TRANSFER})
}

func (service *PhloMultiPartyCallMember) ResumeCall() (*NodeActionResponse, error) {
	return service.update(PhloMultiPartyCallMemberActionPayload{RESUME_CALL})
}
func (service *PhloMultiPartyCallMember) VoiceMailDrop() (*NodeActionResponse, error) {
	return service.update(PhloMultiPartyCallMemberActionPayload{VOICEMAIL_DROP})
}
func (service *PhloMultiPartyCallMember) HangUp() (*NodeActionResponse, error) {
	return service.update(PhloMultiPartyCallMemberActionPayload{HANGUP})
}
func (service *PhloMultiPartyCallMember) Hold() (*NodeActionResponse, error) {
	return service.update(PhloMultiPartyCallMemberActionPayload{HOLD})
}
func (service *PhloMultiPartyCallMember) UnHold() (*NodeActionResponse, error) {
	return service.update(PhloMultiPartyCallMemberActionPayload{UNHOLD})
}

func (service *PhloMultiPartyCallMember) update(params PhloMultiPartyCallMemberActionPayload) (response *NodeActionResponse, err error) {
	req, err := service.client.NewRequest("POST", params, "phlo/%s/%s/%s/members/%s", service.PhloID, service.NodeType,
		service.NodeID, service.MemberAddress)
	if err != nil {
		return
	}
	response = &NodeActionResponse{}
	err = service.client.ExecuteRequest(req, response)

	return
}
