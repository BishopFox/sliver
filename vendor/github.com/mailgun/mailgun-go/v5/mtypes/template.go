package mtypes

type TemplateEngine string

// Used by CreateTemplate() and AddTemplateVersion() to specify the template engine
const (
	TemplateEngineHandlebars = TemplateEngine("handlebars")
	TemplateEngineGo         = TemplateEngine("go")
)

type Template struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	CreatedAt   RFC2822Time     `json:"createdAt"`
	Version     TemplateVersion `json:"version,omitempty"`
}

type TemplateResp struct {
	Item    Template `json:"template"`
	Message string   `json:"message"`
}

type ListTemplateResp struct {
	Items  []Template `json:"items"`
	Paging Paging     `json:"paging"`
}
