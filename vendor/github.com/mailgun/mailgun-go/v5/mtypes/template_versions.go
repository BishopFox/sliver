package mtypes

type TemplateVersion struct {
	Tag       string         `json:"tag"`
	Template  string         `json:"template,omitempty"`
	Engine    TemplateEngine `json:"engine"`
	CreatedAt RFC2822Time    `json:"createdAt"`
	Comment   string         `json:"comment"`
	Active    bool           `json:"active"`
}

type TemplateVersionListResp struct {
	Template struct {
		Template
		Versions []TemplateVersion `json:"versions,omitempty"`
	} `json:"template"`
	Paging Paging `json:"paging"`
}
