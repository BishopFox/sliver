package card

import "time"

var (
	_ Element = (*PickerBlock)(nil)
	_ Element = (*DatePickerBlock)(nil)
	_ Element = (*DatetimePickerBlock)(nil)
	_ Element = (*TimePickerBlock)(nil)
)

// PickerBlock 选择器基础元素
type PickerBlock struct {
	tag             string
	initialDate     string
	initialTime     string
	initialDatetime string
	placeholder     string
	value           map[string]interface{}
	confirm         *ConfirmBlock
}

type pickerRenderer struct {
	ElementTag
	InitialDate     string                 `json:"initial_date,omitempty"`
	InitialTime     string                 `json:"initial_time,omitempty"`
	InitialDatetime string                 `json:"initial_datetime,omitempty"`
	Placeholder     Renderer               `json:"placeholder,omitempty"`
	Value           map[string]interface{} `json:"value,omitempty"`
	Confirm         Renderer               `json:"confirm,omitempty"`
}

// Render 渲染为 Renderer
func (p *PickerBlock) Render() Renderer {
	ret := pickerRenderer{
		ElementTag: ElementTag{
			Tag: p.tag,
		},
		InitialDate:     p.initialDate,
		InitialTime:     p.initialTime,
		InitialDatetime: p.initialDatetime,
		Value:           p.value,
	}
	if p.confirm != nil {
		ret.Confirm = p.confirm.Render()
	}
	if p.placeholder != "" {
		ret.Placeholder = Text(p.placeholder).Render()
	}
	return ret
}

// Placeholder 未选中值时展示的内容，无默认值时必填
func (p *PickerBlock) Placeholder(s string) *PickerBlock {
	p.placeholder = s
	return p
}

// Value 选定后发送给业务方的数据
func (p *PickerBlock) Value(m map[string]interface{}) *PickerBlock {
	p.value = m
	return p
}

// Confirm 选中后二次确认的弹框
func (p *PickerBlock) Confirm(title, text string) *PickerBlock {
	p.confirm = Confirm(title, text)
	return p
}

// DatePickerBlock 日期选择器元素
type DatePickerBlock struct {
	*PickerBlock
}

// DatePicker 日期选择器
func DatePicker() *DatePickerBlock {
	return &DatePickerBlock{PickerBlock: &PickerBlock{tag: "date_picker"}}
}

// InitialDate 设置选择器的默认日期
func (d *DatePickerBlock) InitialDate(t time.Time) *DatePickerBlock {
	return d.InitialDateString(t.Format("2006-01-02"))
}

// InitialDateString 设置选择器的默认日期
func (d *DatePickerBlock) InitialDateString(date string) *DatePickerBlock {
	d.initialDate = date
	return d
}

// DatetimePickerBlock 日期时间选择器元素
type DatetimePickerBlock struct {
	*PickerBlock
}

// DatetimePicker 日期时间选择器
func DatetimePicker() *DatetimePickerBlock {
	return &DatetimePickerBlock{PickerBlock: &PickerBlock{tag: "picker_datetime"}}
}

// InitialDatetime 设置选择器的默认日期时间
func (d *DatetimePickerBlock) InitialDatetime(t time.Time) *DatetimePickerBlock {
	return d.InitialDatetimeString(t.Format("2006-01-02 15:04"))
}

// InitialDatetimeString 设置选择器的默认日期时间
func (d *DatetimePickerBlock) InitialDatetimeString(date string) *DatetimePickerBlock {
	d.initialDatetime = date
	return d
}

// TimePickerBlock 时间选择器元素
type TimePickerBlock struct {
	*PickerBlock
}

// TimePicker 时间选择器
func TimePicker() *TimePickerBlock {
	return &TimePickerBlock{PickerBlock: &PickerBlock{tag: "picker_time"}}
}

// InitialTime 设置选择器的默认时间
func (d *TimePickerBlock) InitialTime(t time.Time) *TimePickerBlock {
	return d.InitialTimeString(t.Format("15:04"))
}

// InitialTimeString 设置选择器的默认时间
func (d *TimePickerBlock) InitialTimeString(date string) *TimePickerBlock {
	d.initialTime = date
	return d
}
