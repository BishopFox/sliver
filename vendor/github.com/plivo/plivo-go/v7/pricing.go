package plivo

type PricingService struct {
	client *Client
}

type PricingGetParams struct {
	CountryISO string `json:"country_iso" url:"country_iso"`
}

type Pricing struct {
	APIID       string `json:"api_id" url:"api_id"`
	Country     string `json:"country" url:"country"`
	CountryCode int    `json:"country_code" url:"country_code"`
	CountryISO  string `json:"country_iso" url:"country_iso"`
	Message     struct {
		Inbound struct {
			Rate string `json:"rate" url:"rate"`
		} `json:"inbound" url:"inbound"`
		Outbound struct {
			Rate string `json:"rate" url:"rate"`
		} `json:"outbound" url:"outbound"`
		OutboundNetworksList []struct {
			GroupName string `json:"group_name" url:"group_name"`
			Rate      string `json:"rate" url:"rate"`
		} `json:"outbound_networks_list" url:"outbound_networks_list"`
	} `json:"message" url:"message"`
	PhoneNumbers struct {
		Local struct {
			Rate string `json:"rate" url:"rate"`
		} `json:"local" url:"local"`
		Tollfree struct {
			Rate string `json:"rate" url:"rate"`
		} `json:"tollfree" url:"tollfree"`
	} `json:"phone_numbers" url:"phone_numbers"`
	Voice struct {
		Inbound struct {
			IP struct {
				Rate string `json:"rate" url:"rate"`
			} `json:"ip" url:"ip"`
			Local struct {
				Rate string `json:"rate" url:"rate"`
			} `json:"local" url:"local"`
			Tollfree struct {
				Rate string `json:"rate" url:"rate"`
			} `json:"tollfree" url:"tollfree"`
		} `json:"inbound" url:"inbound"`
		Outbound struct {
			IP struct {
				Rate string `json:"rate" url:"rate"`
			} `json:"ip" url:"ip"`
			Local struct {
				Rate string `json:"rate" url:"rate"`
			} `json:"local" url:"local"`
			Rates []struct {
				OriginationPrefix []string `json:"origination_prefix" url:"origination_prefix"`
				Prefix            []string `json:"prefix" url:"prefix"`
				Rate              string   `json:"rate" url:"rate"`
				VoiceNetworkGroup string   `json:"voice_network_group" url:"voice_network_group"`
			} `json:"rates" url:"rates"`
			Tollfree struct {
				Rate string `json:"rate" url:"rate"`
			} `json:"tollfree" url:"tollfree"`
		} `json:"outbound" url:"outbound"`
	} `json:"voice" url:"voice"`
}

func (service *PricingService) Get(countryISO string) (response *Pricing, err error) {
	req, err := service.client.NewRequest("GET", PricingGetParams{countryISO}, "Pricing")
	if err != nil {
		return nil, err
	}
	response = &Pricing{}
	err = service.client.ExecuteRequest(req, response)
	return
}
