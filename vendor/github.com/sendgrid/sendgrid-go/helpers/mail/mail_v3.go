package mail

import (
	"encoding/json"
	"fmt"
	"log"
	"net/mail"
	"strings"
)

const (
	// RFC 3696 ( https://tools.ietf.org/html/rfc3696#section-3 )
	// The domain part (after the "@") must not exceed 255 characters
	maxEmailDomainLength = 255
	// The "local part" (before the "@") must not exceed 64 characters
	maxEmailLocalLength = 64
	// Max email length must not exceed 320 characters.
	maxEmailLength = maxEmailDomainLength + maxEmailLocalLength + 1
)

// SGMailV3 contains mail struct
type SGMailV3 struct {
	From             *Email             `json:"from,omitempty"`
	Subject          string             `json:"subject,omitempty"`
	Personalizations []*Personalization `json:"personalizations,omitempty"`
	Content          []*Content         `json:"content,omitempty"`
	Attachments      []*Attachment      `json:"attachments,omitempty"`
	TemplateID       string             `json:"template_id,omitempty"`
	Sections         map[string]string  `json:"sections,omitempty"`
	Headers          map[string]string  `json:"headers,omitempty"`
	Categories       []string           `json:"categories,omitempty"`
	CustomArgs       map[string]string  `json:"custom_args,omitempty"`
	SendAt           int                `json:"send_at,omitempty"`
	BatchID          string             `json:"batch_id,omitempty"`
	Asm              *Asm               `json:"asm,omitempty"`
	IPPoolID         string             `json:"ip_pool_name,omitempty"`
	MailSettings     *MailSettings      `json:"mail_settings,omitempty"`
	TrackingSettings *TrackingSettings  `json:"tracking_settings,omitempty"`
	ReplyTo          *Email             `json:"reply_to,omitempty"`
	ReplyToList      []*Email           `json:"reply_to_list,omitempty"`
}

// Personalization holds mail body struct
type Personalization struct {
	To                  []*Email               `json:"to,omitempty"`
	From                *Email                 `json:"from,omitempty"`
	CC                  []*Email               `json:"cc,omitempty"`
	BCC                 []*Email               `json:"bcc,omitempty"`
	Subject             string                 `json:"subject,omitempty"`
	Headers             map[string]string      `json:"headers,omitempty"`
	Substitutions       map[string]string      `json:"substitutions,omitempty"`
	CustomArgs          map[string]string      `json:"custom_args,omitempty"`
	DynamicTemplateData map[string]interface{} `json:"dynamic_template_data,omitempty"`
	Categories          []string               `json:"categories,omitempty"`
	SendAt              int                    `json:"send_at,omitempty"`
}

// Email holds email name and address info
type Email struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"email,omitempty"`
}

// Content defines content of the mail body
type Content struct {
	Type  string `json:"type,omitempty"`
	Value string `json:"value,omitempty"`
}

// Attachment holds attachement information
type Attachment struct {
	Content     string `json:"content,omitempty"`
	Type        string `json:"type,omitempty"`
	Name        string `json:"name,omitempty"`
	Filename    string `json:"filename,omitempty"`
	Disposition string `json:"disposition,omitempty"`
	ContentID   string `json:"content_id,omitempty"`
}

// Asm contains Grpip Id and int array of groups ID
type Asm struct {
	GroupID         int   `json:"group_id,omitempty"`
	GroupsToDisplay []int `json:"groups_to_display,omitempty"`
}

// MailSettings defines mail and spamCheck settings
type MailSettings struct {
	BCC                         *BccSetting       `json:"bcc,omitempty"`
	BypassListManagement        *Setting          `json:"bypass_list_management,omitempty"`
	BypassSpamManagement        *Setting          `json:"bypass_spam_management,omitempty"`
	BypassBounceManagement      *Setting          `json:"bypass_bounce_management,omitempty"`
	BypassUnsubscribeManagement *Setting          `json:"bypass_unsubscribe_management,omitempty"`
	Footer                      *FooterSetting    `json:"footer,omitempty"`
	SandboxMode                 *Setting          `json:"sandbox_mode,omitempty"`
	SpamCheckSetting            *SpamCheckSetting `json:"spam_check,omitempty"`
}

// TrackingSettings holds tracking settings and mail settings
type TrackingSettings struct {
	ClickTracking        *ClickTrackingSetting        `json:"click_tracking,omitempty"`
	OpenTracking         *OpenTrackingSetting         `json:"open_tracking,omitempty"`
	SubscriptionTracking *SubscriptionTrackingSetting `json:"subscription_tracking,omitempty"`
	GoogleAnalytics      *GaSetting                   `json:"ganalytics,omitempty"`
	BCC                  *BccSetting                  `json:"bcc,omitempty"`
	BypassListManagement *Setting                     `json:"bypass_list_management,omitempty"`
	Footer               *FooterSetting               `json:"footer,omitempty"`
	SandboxMode          *SandboxModeSetting          `json:"sandbox_mode,omitempty"`
}

// BccSetting holds email bcc setings  to enable of disable
// default is false
type BccSetting struct {
	Enable *bool  `json:"enable,omitempty"`
	Email  string `json:"email,omitempty"`
}

// FooterSetting holds enaable/disable settings
// and the format of footer i.e HTML/Text
type FooterSetting struct {
	Enable *bool  `json:"enable,omitempty"`
	Text   string `json:"text,omitempty"`
	Html   string `json:"html,omitempty"`
}

// ClickTrackingSetting ...
type ClickTrackingSetting struct {
	Enable     *bool `json:"enable,omitempty"`
	EnableText *bool `json:"enable_text,omitempty"`
}

// OpenTrackingSetting ...
type OpenTrackingSetting struct {
	Enable          *bool  `json:"enable,omitempty"`
	SubstitutionTag string `json:"substitution_tag,omitempty"`
}

// SandboxModeSetting ...
type SandboxModeSetting struct {
	Enable      *bool             `json:"enable,omitempty"`
	ForwardSpam *bool             `json:"forward_spam,omitempty"`
	SpamCheck   *SpamCheckSetting `json:"spam_check,omitempty"`
}

// SpamCheckSetting holds spam settings and
// which can be enable or disable and
// contains spamThreshold value
type SpamCheckSetting struct {
	Enable        *bool  `json:"enable,omitempty"`
	SpamThreshold int    `json:"threshold,omitempty"`
	PostToURL     string `json:"post_to_url,omitempty"`
}

// SubscriptionTrackingSetting ...
type SubscriptionTrackingSetting struct {
	Enable          *bool  `json:"enable,omitempty"`
	Text            string `json:"text,omitempty"`
	Html            string `json:"html,omitempty"`
	SubstitutionTag string `json:"substitution_tag,omitempty"`
}

// GaSetting ...
type GaSetting struct {
	Enable          *bool  `json:"enable,omitempty"`
	CampaignSource  string `json:"utm_source,omitempty"`
	CampaignTerm    string `json:"utm_term,omitempty"`
	CampaignContent string `json:"utm_content,omitempty"`
	CampaignName    string `json:"utm_campaign,omitempty"`
	CampaignMedium  string `json:"utm_medium,omitempty"`
}

// Setting enables the mail settings
type Setting struct {
	Enable *bool `json:"enable,omitempty"`
}

// NewV3Mail ...
func NewV3Mail() *SGMailV3 {
	return &SGMailV3{
		Personalizations: make([]*Personalization, 0),
		Content:          make([]*Content, 0),
		Attachments:      make([]*Attachment, 0),
	}
}

// NewV3MailInit ...
func NewV3MailInit(from *Email, subject string, to *Email, content ...*Content) *SGMailV3 {
	m := new(SGMailV3)
	m.SetFrom(from)
	m.Subject = subject
	p := NewPersonalization()
	p.AddTos(to)
	m.AddPersonalizations(p)
	m.AddContent(content...)
	return m
}

// GetRequestBody ...
func GetRequestBody(m *SGMailV3) []byte {
	b, err := json.Marshal(m)
	if err != nil {
		log.Println(err)
	}
	return b
}

// AddPersonalizations ...
func (s *SGMailV3) AddPersonalizations(p ...*Personalization) *SGMailV3 {
	s.Personalizations = append(s.Personalizations, p...)
	return s
}

// AddContent ...
func (s *SGMailV3) AddContent(c ...*Content) *SGMailV3 {
	s.Content = append(s.Content, c...)
	return s
}

// AddAttachment ...
func (s *SGMailV3) AddAttachment(a ...*Attachment) *SGMailV3 {
	s.Attachments = append(s.Attachments, a...)
	return s
}

// SetFrom ...
func (s *SGMailV3) SetFrom(e *Email) *SGMailV3 {
	s.From = e
	return s
}

// SetReplyTo ...
func (s *SGMailV3) SetReplyTo(e *Email) *SGMailV3 {
	s.ReplyTo = e
	return s
}

// SetReplyToList ...
func (s *SGMailV3) SetReplyToList(e []*Email) *SGMailV3 {
	s.ReplyToList = e
	return s
}

// SetTemplateID ...
func (s *SGMailV3) SetTemplateID(templateID string) *SGMailV3 {
	s.TemplateID = templateID
	return s
}

// AddSection ...
func (s *SGMailV3) AddSection(key string, value string) *SGMailV3 {
	if s.Sections == nil {
		s.Sections = make(map[string]string)
	}

	s.Sections[key] = value
	return s
}

// SetHeader ...
func (s *SGMailV3) SetHeader(key string, value string) *SGMailV3 {
	if s.Headers == nil {
		s.Headers = make(map[string]string)
	}

	s.Headers[key] = value
	return s
}

// AddCategories ...
func (s *SGMailV3) AddCategories(category ...string) *SGMailV3 {
	s.Categories = append(s.Categories, category...)
	return s
}

// SetCustomArg ...
func (s *SGMailV3) SetCustomArg(key string, value string) *SGMailV3 {
	if s.CustomArgs == nil {
		s.CustomArgs = make(map[string]string)
	}

	s.CustomArgs[key] = value
	return s
}

// SetSendAt ...
func (s *SGMailV3) SetSendAt(sendAt int) *SGMailV3 {
	s.SendAt = sendAt
	return s
}

// SetBatchID ...
func (s *SGMailV3) SetBatchID(batchID string) *SGMailV3 {
	s.BatchID = batchID
	return s
}

// SetASM ...
func (s *SGMailV3) SetASM(asm *Asm) *SGMailV3 {
	s.Asm = asm
	return s
}

// SetIPPoolID ...
func (s *SGMailV3) SetIPPoolID(ipPoolID string) *SGMailV3 {
	s.IPPoolID = ipPoolID
	return s
}

// SetMailSettings ...
func (s *SGMailV3) SetMailSettings(mailSettings *MailSettings) *SGMailV3 {
	s.MailSettings = mailSettings
	return s
}

// SetTrackingSettings ...
func (s *SGMailV3) SetTrackingSettings(trackingSettings *TrackingSettings) *SGMailV3 {
	s.TrackingSettings = trackingSettings
	return s
}

// NewPersonalization ...
func NewPersonalization() *Personalization {
	return &Personalization{
		To:                  make([]*Email, 0),
		CC:                  make([]*Email, 0),
		BCC:                 make([]*Email, 0),
		Headers:             make(map[string]string),
		Substitutions:       make(map[string]string),
		CustomArgs:          make(map[string]string),
		DynamicTemplateData: make(map[string]interface{}),
		Categories:          make([]string, 0),
	}
}

// AddTos ...
func (p *Personalization) AddTos(to ...*Email) {
	p.To = append(p.To, to...)
}

// AddFrom ...
func (p *Personalization) AddFrom(from *Email) {
	p.From = from
}

// AddCCs ...
func (p *Personalization) AddCCs(cc ...*Email) {
	p.CC = append(p.CC, cc...)
}

// AddBCCs ...
func (p *Personalization) AddBCCs(bcc ...*Email) {
	p.BCC = append(p.BCC, bcc...)
}

// SetHeader ...
func (p *Personalization) SetHeader(key string, value string) {
	p.Headers[key] = value
}

// SetSubstitution ...
func (p *Personalization) SetSubstitution(key string, value string) {
	p.Substitutions[key] = value
}

// SetCustomArg ...
func (p *Personalization) SetCustomArg(key string, value string) {
	p.CustomArgs[key] = value
}

// SetDynamicTemplateData ...
func (p *Personalization) SetDynamicTemplateData(key string, value interface{}) {
	p.DynamicTemplateData[key] = value
}

// SetSendAt ...
func (p *Personalization) SetSendAt(sendAt int) {
	p.SendAt = sendAt
}

// NewAttachment ...
func NewAttachment() *Attachment {
	return &Attachment{}
}

// SetContent ...
func (a *Attachment) SetContent(content string) *Attachment {
	a.Content = content
	return a
}

// SetType ...
func (a *Attachment) SetType(contentType string) *Attachment {
	a.Type = contentType
	return a
}

// SetFilename ...
func (a *Attachment) SetFilename(filename string) *Attachment {
	a.Filename = filename
	return a
}

// SetDisposition ...
func (a *Attachment) SetDisposition(disposition string) *Attachment {
	a.Disposition = disposition
	return a
}

// SetContentID ...
func (a *Attachment) SetContentID(contentID string) *Attachment {
	a.ContentID = contentID
	return a
}

// NewASM ...
func NewASM() *Asm {
	return &Asm{}
}

// SetGroupID ...
func (a *Asm) SetGroupID(groupID int) *Asm {
	a.GroupID = groupID
	return a
}

// AddGroupsToDisplay ...
func (a *Asm) AddGroupsToDisplay(groupsToDisplay ...int) *Asm {
	a.GroupsToDisplay = append(a.GroupsToDisplay, groupsToDisplay...)
	return a
}

// NewMailSettings ...
func NewMailSettings() *MailSettings {
	return &MailSettings{}
}

// SetBCC ...
func (m *MailSettings) SetBCC(bcc *BccSetting) *MailSettings {
	m.BCC = bcc
	return m
}

// SetBypassListManagement ...
func (m *MailSettings) SetBypassListManagement(bypassListManagement *Setting) *MailSettings {
	m.BypassListManagement = bypassListManagement
	return m
}

// SetBypassSpamManagement ...
func (m *MailSettings) SetBypassSpamManagement(bypassSpamManagement *Setting) *MailSettings {
	m.BypassSpamManagement = bypassSpamManagement
	return m
}

// SetBypassBounceManagement ...
func (m *MailSettings) SetBypassBounceManagement(bypassBounceManagement *Setting) *MailSettings {
	m.BypassBounceManagement = bypassBounceManagement
	return m
}

// SetBypassUnsubscribeManagement ...
func (m *MailSettings) SetBypassUnsubscribeManagement(bypassUnsubscribeManagement *Setting) *MailSettings {
	m.BypassUnsubscribeManagement = bypassUnsubscribeManagement
	return m
}

// SetFooter ...
func (m *MailSettings) SetFooter(footerSetting *FooterSetting) *MailSettings {
	m.Footer = footerSetting
	return m
}

// SetSandboxMode ...
func (m *MailSettings) SetSandboxMode(sandboxMode *Setting) *MailSettings {
	m.SandboxMode = sandboxMode
	return m
}

// SetSpamCheckSettings ...
func (m *MailSettings) SetSpamCheckSettings(spamCheckSetting *SpamCheckSetting) *MailSettings {
	m.SpamCheckSetting = spamCheckSetting
	return m
}

// NewTrackingSettings ...
func NewTrackingSettings() *TrackingSettings {
	return &TrackingSettings{}
}

// SetClickTracking ...
func (t *TrackingSettings) SetClickTracking(clickTracking *ClickTrackingSetting) *TrackingSettings {
	t.ClickTracking = clickTracking
	return t

}

// SetOpenTracking ...
func (t *TrackingSettings) SetOpenTracking(openTracking *OpenTrackingSetting) *TrackingSettings {
	t.OpenTracking = openTracking
	return t
}

// SetSubscriptionTracking ...
func (t *TrackingSettings) SetSubscriptionTracking(subscriptionTracking *SubscriptionTrackingSetting) *TrackingSettings {
	t.SubscriptionTracking = subscriptionTracking
	return t
}

// SetGoogleAnalytics ...
func (t *TrackingSettings) SetGoogleAnalytics(googleAnalytics *GaSetting) *TrackingSettings {
	t.GoogleAnalytics = googleAnalytics
	return t
}

// NewBCCSetting ...
func NewBCCSetting() *BccSetting {
	return &BccSetting{}
}

// SetEnable ...
func (b *BccSetting) SetEnable(enable bool) *BccSetting {
	setEnable := enable
	b.Enable = &setEnable
	return b
}

// SetEmail ...
func (b *BccSetting) SetEmail(email string) *BccSetting {
	b.Email = email
	return b
}

// NewFooterSetting ...
func NewFooterSetting() *FooterSetting {
	return &FooterSetting{}
}

// SetEnable ...
func (f *FooterSetting) SetEnable(enable bool) *FooterSetting {
	setEnable := enable
	f.Enable = &setEnable
	return f
}

// SetText ...
func (f *FooterSetting) SetText(text string) *FooterSetting {
	f.Text = text
	return f
}

// SetHTML ...
func (f *FooterSetting) SetHTML(html string) *FooterSetting {
	f.Html = html
	return f
}

// NewOpenTrackingSetting ...
func NewOpenTrackingSetting() *OpenTrackingSetting {
	return &OpenTrackingSetting{}
}

// SetEnable ...
func (o *OpenTrackingSetting) SetEnable(enable bool) *OpenTrackingSetting {
	setEnable := enable
	o.Enable = &setEnable
	return o
}

// SetSubstitutionTag ...
func (o *OpenTrackingSetting) SetSubstitutionTag(subTag string) *OpenTrackingSetting {
	o.SubstitutionTag = subTag
	return o
}

// NewSubscriptionTrackingSetting ...
func NewSubscriptionTrackingSetting() *SubscriptionTrackingSetting {
	return &SubscriptionTrackingSetting{}
}

// SetEnable ...
func (s *SubscriptionTrackingSetting) SetEnable(enable bool) *SubscriptionTrackingSetting {
	setEnable := enable
	s.Enable = &setEnable
	return s
}

// SetText ...
func (s *SubscriptionTrackingSetting) SetText(text string) *SubscriptionTrackingSetting {
	s.Text = text
	return s
}

// SetHTML ...
func (s *SubscriptionTrackingSetting) SetHTML(html string) *SubscriptionTrackingSetting {
	s.Html = html
	return s
}

// SetSubstitutionTag ...
func (s *SubscriptionTrackingSetting) SetSubstitutionTag(subTag string) *SubscriptionTrackingSetting {
	s.SubstitutionTag = subTag
	return s
}

// NewGaSetting ...
func NewGaSetting() *GaSetting {
	return &GaSetting{}
}

// SetEnable ...
func (g *GaSetting) SetEnable(enable bool) *GaSetting {
	setEnable := enable
	g.Enable = &setEnable
	return g
}

// SetCampaignSource ...
func (g *GaSetting) SetCampaignSource(campaignSource string) *GaSetting {
	g.CampaignSource = campaignSource
	return g
}

// SetCampaignContent ...
func (g *GaSetting) SetCampaignContent(campaignContent string) *GaSetting {
	g.CampaignContent = campaignContent
	return g
}

// SetCampaignTerm ...
func (g *GaSetting) SetCampaignTerm(campaignTerm string) *GaSetting {
	g.CampaignTerm = campaignTerm
	return g
}

// SetCampaignName ...
func (g *GaSetting) SetCampaignName(campaignName string) *GaSetting {
	g.CampaignName = campaignName
	return g
}

// SetCampaignMedium ...
func (g *GaSetting) SetCampaignMedium(campaignMedium string) *GaSetting {
	g.CampaignMedium = campaignMedium
	return g
}

// NewSetting ...
func NewSetting(enable bool) *Setting {
	setEnable := enable
	return &Setting{Enable: &setEnable}
}

// NewEmail ...
func NewEmail(name string, address string) *Email {
	return &Email{
		Name:    name,
		Address: address,
	}
}

// NewSingleEmail ...
func NewSingleEmail(from *Email, subject string, to *Email, plainTextContent string, htmlContent string) *SGMailV3 {
	var contents []*Content
	if plainTextContent != "" {
		contents = append(contents, NewContent("text/plain", plainTextContent))
	}
	if htmlContent != "" {
		contents = append(contents, NewContent("text/html", htmlContent))
	}
	return NewV3MailInit(from, subject, to, contents...)
}

// NewSingleEmailPlainText is used to build *SGMailV3 object having only 'plain-text' as email content.
func NewSingleEmailPlainText(from *Email, subject string, to *Email, plainTextContent string) *SGMailV3 {
	plainText := NewContent("text/plain", plainTextContent)
	return NewV3MailInit(from, subject, to, plainText)
}

// NewContent ...
func NewContent(contentType string, value string) *Content {
	return &Content{
		Type:  contentType,
		Value: value,
	}
}

// NewClickTrackingSetting ...
func NewClickTrackingSetting() *ClickTrackingSetting {
	return &ClickTrackingSetting{}
}

// SetEnable ...
func (c *ClickTrackingSetting) SetEnable(enable bool) *ClickTrackingSetting {
	setEnable := enable
	c.Enable = &setEnable
	return c
}

// SetEnableText ...
func (c *ClickTrackingSetting) SetEnableText(enableText bool) *ClickTrackingSetting {
	setEnable := enableText
	c.EnableText = &setEnable
	return c
}

// NewSpamCheckSetting ...
func NewSpamCheckSetting() *SpamCheckSetting {
	return &SpamCheckSetting{}
}

// SetEnable ...
func (s *SpamCheckSetting) SetEnable(enable bool) *SpamCheckSetting {
	setEnable := enable
	s.Enable = &setEnable
	return s
}

// SetSpamThreshold ...
func (s *SpamCheckSetting) SetSpamThreshold(spamThreshold int) *SpamCheckSetting {
	s.SpamThreshold = spamThreshold
	return s
}

// SetPostToURL ...
func (s *SpamCheckSetting) SetPostToURL(postToURL string) *SpamCheckSetting {
	s.PostToURL = postToURL
	return s
}

// NewSandboxModeSetting ...
func NewSandboxModeSetting(enable bool, forwardSpam bool, spamCheck *SpamCheckSetting) *SandboxModeSetting {
	setEnable := enable
	setForwardSpam := forwardSpam
	return &SandboxModeSetting{
		Enable:      &setEnable,
		ForwardSpam: &setForwardSpam,
		SpamCheck:   spamCheck,
	}
}

// ParseEmail parses a string that contains an rfc822 formatted email address
// and returns an instance of *Email.
func ParseEmail(emailInfo string) (*Email, error) {
	e, err := mail.ParseAddress(emailInfo)
	if err != nil {
		return nil, err
	}

	if len(e.Address) > maxEmailLength {
		return nil, fmt.Errorf("Invalid email length. Total length should not exceed %d characters.", maxEmailLength)
	}

	parts := strings.Split(e.Address, "@")
	local, domain := parts[0], parts[1]

	if len(domain) > maxEmailDomainLength {
		return nil, fmt.Errorf("Invalid email length. Domain length should not exceed %d characters.", maxEmailDomainLength)
	}

	if len(local) > maxEmailLocalLength {
		return nil, fmt.Errorf("Invalid email length. Local part length should not exceed %d characters.", maxEmailLocalLength)
	}

	return NewEmail(e.Name, e.Address), nil
}
