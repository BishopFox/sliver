package plivo

const (
	SMS = "sms"
	MMS = "mms"
)

type MessageService struct {
	client *Client
	Message
}

type MessageCreateParams struct {
	Src  string `json:"src,omitempty" url:"src,omitempty"`
	Dst  string `json:"dst,omitempty" url:"dst,omitempty"`
	Text string `json:"text,omitempty" url:"text,omitempty"`
	// Optional parameters.
	Type      string      `json:"type,omitempty" url:"type,omitempty"`
	URL       string      `json:"url,omitempty" url:"url,omitempty"`
	Method    string      `json:"method,omitempty" url:"method,omitempty"`
	Trackable bool        `json:"trackable,omitempty" url:"trackable,omitempty"`
	Log       interface{} `json:"log,omitempty" url:"log,omitempty"`
	MediaUrls []string    `json:"media_urls,omitempty" url:"media_urls,omitempty"`
	MediaIds  []string    `json:"media_ids,omitempty" url:"media_ids,omitempty"`
	// Either one of src and powerpackuuid should be given
	PowerpackUUID       string       `json:"powerpack_uuid,omitempty" url:"powerpack_uuid,omitempty"`
	MessageExpiry       int          `json:"message_expiry,omitempty" url:"message_expiry,omitempty"`
	Template            *Template    `json:"template,omitempty" url:"template,omitempty"`
	Interactive         *Interactive `json:"interactive,omitempty" url:"interactive,omitempty"`
	Location            *Location    `json:"location,omitempty" url:"location,omitempty"`
	DLTEntityID         string       `json:"dlt_entity_id,omitempty" url:"dlt_entity_id,omitempty"`
	DLTTemplateID       string       `json:"dlt_template_id,omitempty" url:"dlt_template_id,omitempty"`
	DLTTemplateCategory string       `json:"dlt_template_category,omitempty" url:"dlt_template_category,omitempty"`
}

type Message struct {
	ApiID                           string `json:"api_id,omitempty" url:"api_id,omitempty"`
	ToNumber                        string `json:"to_number,omitempty" url:"to_number,omitempty"`
	FromNumber                      string `json:"from_number,omitempty" url:"from_number,omitempty"`
	CloudRate                       string `json:"cloud_rate,omitempty" url:"cloud_rate,omitempty"`
	MessageType                     string `json:"message_type,omitempty" url:"message_type,omitempty"`
	ResourceURI                     string `json:"resource_uri,omitempty" url:"resource_uri,omitempty"`
	CarrierRate                     string `json:"carrier_rate,omitempty" url:"carrier_rate,omitempty"`
	MessageDirection                string `json:"message_direction,omitempty" url:"message_direction,omitempty"`
	MessageState                    string `json:"message_state,omitempty" url:"message_state,omitempty"`
	TotalAmount                     string `json:"total_amount,omitempty" url:"total_amount,omitempty"`
	MessageUUID                     string `json:"message_uuid,omitempty" url:"message_uuid,omitempty"`
	MessageTime                     string `json:"message_time,omitempty" url:"message_time,omitempty"`
	ErrorCode                       string `json:"error_code,omitempty" url:"error_code,omitempty"`
	ErrorMessage                    string `json:"error_message,omitempty" url:"error_message,omitempty"`
	PowerpackID                     string `json:"powerpack_id,omitempty" url:"powerpack_id,omitempty"`
	RequesterIP                     string `json:"requester_ip,omitempty" url:"requester_ip,omitempty"`
	IsDomestic                      *bool  `json:"is_domestic,omitempty" url:"is_domestic,omitempty"`
	ReplacedSender                  string `json:"replaced_sender,omitempty" url:"replaced_sender,omitempty"`
	TendlcCampaignID                string `json:"tendlc_campaign_id" url:"tendlc_campaign_id,omitempty"`
	TendlcRegistrationStatus        string `json:"tendlc_registration_status" url:"tendlc_registration_status,omitempty"`
	DestinationCountryISO2          string `json:"destination_country_iso2" url:"destination_country_iso2,omitempty"`
	ConversationID                  string `json:"conversation_id" url:"conversation_id,omitempty"`
	ConversationOrigin              string `json:"conversation_origin" url:"conversation_origin,omitempty"`
	ConversationExpirationTimestamp string `json:"conversation_expiration_timestamp" url:"conversation_expiration_timestamp,omitempty"`
	DLTEntityID                     string `json:"dlt_entity_id" url:"dlt_entity_id,omitempty"`
	DLTTemplateID                   string `json:"dlt_template_id" url:"dlt_template_id,omitempty"`
	DLTTemplateCategory             string `json:"dlt_template_category" url:"dlt_template_category,omitempty"`
	DestinationNetwork              string `json:"destination_network" url:"destination_network,omitempty"`
	CarrierFeesRate                 string `json:"carrier_fees_rate" url:"carrier_fees_rate,omitempty"`
	CarrierFees                     string `json:"carrier_fees" url:"carrier_fees,omitempty"`
	Log                             string `json:"log" url:"log,omitempty"`
	MessageSentTime                 string `json:"message_sent_time" url:"message_sent_time,omitempty"`
	MessageUpdatedTime              string `json:"message_updated_time" url:"message_updated_time,omitempty"`
}

// Stores response for ending a message.
type MessageCreateResponseBody struct {
	Message     string   `json:"message" url:"message"`
	ApiID       string   `json:"api_id" url:"api_id"`
	MessageUUID []string `json:"message_uuid" url:"message_uuid"`
	Error       string   `json:"error" url:"error"`
}

type MediaDeleteResponse struct {
	Error string `json:"error,omitempty"`
}
type MMSMedia struct {
	ApiID       string `json:"api_id,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	MediaID     string `json:"media_id,omitempty"`
	MediaURL    string `json:"media_url,omitempty"`
	MessageUUID string `json:"message_uuid,omitempty"`
	Size        int64  `json:"size,omitempty"`
}

type MessageList struct {
	ApiID string `json:"api_id" url:"api_id"`
	Meta  struct {
		Previous *string
		Next     *string
		Offset   int64
		Limit    int64
	} `json:"meta"`
	Objects []Message `json:"objects" url:"objects"`
}

type MediaListResponseBody struct {
	Objects []MMSMedia `json:"objects" url:"objects"`
}

type MessageListParams struct {
	Limit                     int    `url:"limit,omitempty"`
	Offset                    int    `url:"offset,omitempty"`
	PowerpackID               string `url:"powerpack_id,omitempty"`
	Subaccount                string `url:"subaccount,omitempty"`
	MessageDirection          string `url:"message_direction,omitempty"`
	MessageState              string `url:"message_state,omitempty"`
	ErrorCode                 int    `url:"error_code,omitempty"`
	MessageTime               string `url:"message_time,omitempty"`
	MessageTimeGreaterThan    string `url:"message_time__gt,omitempty"`
	MessageTimeGreaterOrEqual string `url:"message_time__gte,omitempty"`
	MessageTimeLessThan       string `url:"message_time__lt,omitempty"`
	MessageTimeLessOrEqual    string `url:"message_time__lte,omitempty"`
	TendlcCampaignID          string `url:"tendlc_campaign_id,omitempty"`
	TendlcRegistrationStatus  string `url:"tendlc_registration_status,omitempty"`
	DestinationCountryISO2    string `url:"destination_country_iso2,omitempty"`
	MessageType               string `url:"message_type,omitempty,enum:sms,mms,whatsapp"`
	ConversationID            string `url:"conversation_id,omitempty"`
	ConversationOrigin        string `url:"conversation_origin,omitempty,enum:service,utility,authentication,marketing"`
}

type Template struct {
	Name       string      `mapstructure:"name" json:"name" validate:"required"`
	Language   string      `mapstructure:"language" json:"language" validate:"required"`
	Components []Component `mapstructure:"components" json:"components"`
}
type Component struct {
	Type       string      `mapstructure:"type" json:"type" validate:"required"`
	SubType    string      `mapstructure:"sub_type" json:"sub_type,omitempty"`
	Index      string      `mapstructure:"index" json:"index,omitempty"`
	Parameters []Parameter `mapstructure:"parameters" json:"parameters"`
	Cards      []Card      `mapstructure:"cards" json:"cards,omitempty"`
}

type Card struct {
	CardIndex  int         `mapstructure:"card_index" json:"card_index,omitempty"`
	Components []Component `mapstructure:"components" json:"components,omitempty"`
}

type Parameter struct {
	Type          string    `mapstructure:"type" json:"type" validate:"required"`
	Text          string    `mapstructure:"text" json:"text,omitempty"`
	Media         string    `mapstructure:"media" json:"media,omitempty"`
	Payload       string    `mapstructure:"payload" json:"payload,omitempty"`
	Currency      *Currency `mapstructure:"currency" json:"currency,omitempty"`
	DateTime      *DateTime `mapstructure:"date_time" json:"date_time,omitempty"`
	Location      *Location `mapstructure:"location" json:"location,omitempty"`
	ParameterName *string   `mapstructure:"parameter_name" json:"parameter_name,omitempty"`
}

type Location struct {
	Longitude string `mapstructure:"longitude" json:"longitude,omitempty"`
	Latitude  string `mapstructure:"latitude" json:"latitude,omitempty"`
	Name      string `mapstructure:"name" json:"name,omitempty"`
	Address   string `mapstructure:"address" json:"address,omitempty"`
}

type Currency struct {
	FallbackValue string `mapstructure:"fallback_value" json:"fallback_value"`
	CurrencyCode  string `mapstructure:"currency_code" json:"currency_code"`
	Amount1000    int    `mapstructure:"amount_1000" json:"amount_1000"`
}

type DateTime struct {
	FallbackValue string `mapstructure:"fallback_value" json:"fallback_value"`
}

type Interactive struct {
	Type   string  `mapstructure:"type" json:"type,omitempty"`
	Header *Header `mapstructure:"header" json:"header,omitempty"`
	Body   *Body   `mapstructure:"body" json:"body,omitempty"`
	Footer *Footer `mapstructure:"footer" json:"footer,omitempty"`
	Action *Action `mapstructure:"action" json:"action,omitempty"`
}

type Header struct {
	Type  string  `mapstructure:"type" json:"type,omitempty"`
	Text  *string `mapstructure:"text" json:"text,omitempty"`
	Media *string `mapstructure:"media" json:"media,omitempty"`
}

type Body struct {
	Text string `mapstructure:"text" json:"text,omitempty"`
}

type Footer struct {
	Text string `mapstructure:"text" json:"text,omitempty"`
}

type Action struct {
	Button  []*Buttons `mapstructure:"buttons" json:"buttons,omitempty"`
	Section []*Section `mapstructure:"sections" json:"sections,omitempty"`
}

type Buttons struct {
	ID     string `mapstructure:"id" json:"id,omitempty"`
	Title  string `mapstructure:"title" json:"title,omitempty"`
	CTAURL string `mapstructure:"cta_url" json:"cta_url,omitempty"`
}

type Section struct {
	Title string `mapstructure:"title" json:"title,omitempty"`
	Row   []*Row `mapstructure:"rows" json:"rows,omitempty"`
}

type Row struct {
	ID          string `mapstructure:"id" json:"id,omitempty"`
	Title       string `mapstructure:"title" json:"title,omitempty"`
	Description string `mapstructure:"description" json:"description,omitempty"`
}

func (service *MessageService) List(params MessageListParams) (response *MessageList, err error) {
	req, err := service.client.NewRequest("GET", params, "Message")
	if err != nil {
		return
	}
	response = &MessageList{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *MessageService) Get(messageUuid string) (response *Message, err error) {
	req, err := service.client.NewRequest("GET", nil, "Message/%s", messageUuid)
	if err != nil {
		return
	}
	response = &Message{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *MessageService) Create(params MessageCreateParams) (response *MessageCreateResponseBody, err error) {
	req, err := service.client.NewRequest("POST", params, "Message")
	if err != nil {
		return
	}
	response = &MessageCreateResponseBody{}
	err = service.client.ExecuteRequest(req, response)
	return
}

func (service *MessageService) ListMedia(messageUuid string) (response *MediaListResponseBody, err error) {
	req, err := service.client.NewRequest("GET", nil, "Message/%s/Media/", messageUuid)
	if err != nil {
		return
	}
	response = &MediaListResponseBody{}
	err = service.client.ExecuteRequest(req, response)
	return
}
