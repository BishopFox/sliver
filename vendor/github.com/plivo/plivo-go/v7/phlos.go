package plivo

const nodeType = "multi_party_call"

type Phlo struct {
	BaseResource
	ApiId     string `json:"api_id" url:"api_id"`
	PhloId    string `json:"phlo_id" url:"phlo_id"`
	Name      string `json:"name" url:"name"`
	CreatedOn string `json:"created_on" url:"created_on"`
}

type Phlos struct {
	BaseResourceInterface
}

func NewPhlos(client *PhloClient) (phlos *Phlos) {
	phlos = &Phlos{}
	phlos.client = client

	return
}

type PhloRun struct {
	ApiID     string `json:"api_id" url:"api_id"`
	PhloRunID string `json:"phlo_run_id" url:"phlo_run_id"`
	PhloID    string `json:"phlo_id" url:"phlo_id"`
	Message   string `json:"message" url:"message"`
}

func (self *Phlos) Get(phloId string) (response *Phlo, err error) {
	req, err := self.client.NewRequest("GET", nil, "phlo/%s", phloId)
	if err != nil {
		return
	}
	response = &Phlo{}
	response.client = self.client
	err = self.client.ExecuteRequest(req, response)

	return
}

func (self *Phlo) Node(nodeId string) (response *Node, err error) {
	req, err := self.client.NewRequest("GET", nil, "phlo/%s/%s/%s", self.PhloId, nodeType, nodeId)
	if err != nil {
		return
	}
	response = &Node{}
	err = self.client.ExecuteRequest(req, response)
	return
}

func (self *Phlo) MultiPartyCall(nodeId string) (response *PhloMultiPartyCall, err error) {
	req, err := self.client.NewRequest("GET", nil, "phlo/%s/%s/%s", self.PhloId, nodeType, nodeId)
	if err != nil {
		return
	}
	response = &PhloMultiPartyCall{}
	response.client = self.client
	err = self.client.ExecuteRequest(req, response)
	return
}

func (self *Phlo) Run(data map[string]interface{}) (response *PhloRun, err error) {
	req, err := self.client.NewRequest("POST", data, "account/%s/phlo/%s/", self.client.AuthId, self.PhloId)
	if err != nil {
		return
	}
	response = &PhloRun{}
	err = self.client.ExecuteRequest(req, response)
	return
}
