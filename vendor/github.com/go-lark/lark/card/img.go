package card

var _ Element = (*ImgBlock)(nil)

// ImgBlock 图片元素
type ImgBlock struct {
	key       string
	alt       string
	title     *TextBlock
	width     int
	compact   bool
	mode      string
	noPreview bool
}

type imgRenderer struct {
	ElementTag
	ImgKey       string   `json:"img_key"`
	Alt          Renderer `json:"alt"`
	Title        Renderer `json:"title,omitempty"`
	CustomWidth  int      `json:"custom_width,omitempty"`
	CompactWidth bool     `json:"compact_width,omitempty"`
	Mode         string   `json:"mode,omitempty"`
	Preview      *bool    `json:"preview,omitempty"`
}

// Render 渲染为 Renderer
func (i *ImgBlock) Render() Renderer {
	r := imgRenderer{
		ElementTag: ElementTag{
			Tag: "img",
		},
		ImgKey:       i.key,
		Alt:          Text(i.alt).Render(),
		CustomWidth:  i.width,
		CompactWidth: i.compact,
		Mode:         i.mode,
	}
	if i.noPreview {
		f := false
		r.Preview = &f
	}
	if i.title != nil {
		r.Title = i.title.Render()
	}
	return r
}

// Img 图片展示模块
func Img(key string) *ImgBlock {
	return &ImgBlock{key: key}
}

// Alt Hover图片时展示的文案，留空则不展示
func (i *ImgBlock) Alt(s string) *ImgBlock {
	i.alt = s
	return i
}

// Title 图片标题
func (i *ImgBlock) Title(t *TextBlock) *ImgBlock {
	i.title = t
	return i
}

// TitleString 图片标题
func (i *ImgBlock) TitleString(t string) *ImgBlock {
	return i.Title(Text(t))
}

// Width 图片的最大展示宽度，范围 278 ~ 580
func (i *ImgBlock) Width(w int) *ImgBlock {
	i.width = w
	return i
}

// Compact 展示紧凑型的图片，设置后最大宽度为 278px
func (i *ImgBlock) Compact() *ImgBlock {
	i.compact = true
	return i
}

// FitHorizontal 平铺模式
func (i *ImgBlock) FitHorizontal() *ImgBlock {
	i.mode = "fit_horizontal"
	return i
}

// CropCenter 居中裁剪模式（默认）
func (i *ImgBlock) CropCenter() *ImgBlock {
	i.mode = "crop_center"
	return i
}

// NoPreview 设置后，点击图片后将不会放大图片，可与 CardLink 同时设置
func (i *ImgBlock) NoPreview() *ImgBlock {
	i.noPreview = false
	return i
}
