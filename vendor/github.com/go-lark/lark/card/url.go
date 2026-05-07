package card

var _ Element = (*URLBlock)(nil)

// URLBlock 链接模块
type URLBlock struct {
	href                         string
	hrefAndroid, hrefIOS, hrefPC string
}

type urlRenderer struct {
	URL        string `json:"url,omitempty"`
	AndroidURL string `json:"android_url,omitempty"`
	IosURL     string `json:"ios_url,omitempty"`
	PcURL      string `json:"pc_url,omitempty"`
}

// Render 渲染为 Renderer
func (u *URLBlock) Render() Renderer {
	return urlRenderer{
		URL:        u.href,
		AndroidURL: u.hrefAndroid,
		IosURL:     u.hrefIOS,
		PcURL:      u.hrefPC,
	}
}

// URL 链接模块
func URL() *URLBlock {
	return &URLBlock{}
}

// Href 默认跳转链接
func (u *URLBlock) Href(s string) *URLBlock {
	u.href = s
	return u
}

// MultiHref 多端跳转链接，设置后 Href 将被忽略
func (u *URLBlock) MultiHref(android, ios, pc string) *URLBlock {
	u.hrefAndroid = android
	u.hrefIOS = ios
	u.hrefPC = pc
	if u.href == "" {
		u.href = u.hrefPC
	}
	return u
}
