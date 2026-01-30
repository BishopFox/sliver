package lark

// MsgTemplateBuilder for build template
type MsgTemplateBuilder struct {
	id          string
	versionName string
	data        map[string]interface{}
}

// NewTemplateBuilder creates a text builder
func NewTemplateBuilder() *MsgTemplateBuilder {
	return &MsgTemplateBuilder{}
}

// BindTemplate .
func (tb *MsgTemplateBuilder) BindTemplate(id, versionName string, data map[string]interface{}) *TemplateContent {
	tb.id = id
	tb.versionName = versionName
	tb.data = data

	tc := &TemplateContent{
		Type: "template",
		Data: templateData{
			TemplateID:          tb.id,
			TemplateVersionName: tb.versionName,
		},
	}

	if data != nil {
		tc.Data.TemplateVariable = tb.data
	}

	return tc
}
