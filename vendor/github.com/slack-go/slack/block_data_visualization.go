package slack

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// DataVisualizationChartType identifies the chart payload inside a
// DataVisualizationBlock.
type DataVisualizationChartType string

const (
	DataVisualizationChartPie  DataVisualizationChartType = "pie"
	DataVisualizationChartBar  DataVisualizationChartType = "bar"
	DataVisualizationChartArea DataVisualizationChartType = "area"
	DataVisualizationChartLine DataVisualizationChartType = "line"
)

// DataVisualizationChart is implemented by every chart payload valid inside a
// DataVisualizationBlock: pie, bar, area, and line.
type DataVisualizationChart interface {
	DataVisualizationChartType() DataVisualizationChartType
}

// DataVisualizationSegment is a labeled slice in a pie chart.
type DataVisualizationSegment struct {
	// Label is the display name for this slice, shown in the legend and on hover.
	// Slack requires a maximum of 20 characters.
	Label string `json:"label"`
	// Value is the numeric weight of this slice. Slack requires it to be greater
	// than 0.
	Value float64 `json:"value"`
}

// NewDataVisualizationSegment returns a pie-chart segment.
func NewDataVisualizationSegment(label string, value float64) DataVisualizationSegment {
	return DataVisualizationSegment{Label: label, Value: value}
}

// DataVisualizationDataPoint is a labeled point in a bar, area, or line chart.
type DataVisualizationDataPoint struct {
	// Label is the x-axis category this point belongs to. Slack requires it to
	// match one of AxisConfig.Categories and be at most 20 characters.
	Label string `json:"label"`
	// Value is the numeric y-axis value. Slack permits negative values.
	Value float64 `json:"value"`
}

// NewDataVisualizationDataPoint returns a chart data point.
func NewDataVisualizationDataPoint(label string, value float64) DataVisualizationDataPoint {
	return DataVisualizationDataPoint{Label: label, Value: value}
}

// DataVisualizationDataSeries is a named sequence of data points.
type DataVisualizationDataSeries struct {
	// Name is the human-readable identifier displayed in the chart legend. Slack
	// requires it to be unique across all series in the same chart and at most 20
	// characters.
	Name string `json:"name"`
	// Data is the ordered set of data points. Slack requires 1 to 20 points and
	// exactly one point for every AxisConfig.Categories entry.
	Data []DataVisualizationDataPoint `json:"data"`
}

// NewDataVisualizationDataSeries returns a named chart series.
func NewDataVisualizationDataSeries(name string, data ...DataVisualizationDataPoint) DataVisualizationDataSeries {
	return DataVisualizationDataSeries{Name: name, Data: append([]DataVisualizationDataPoint{}, data...)}
}

// DataVisualizationAxisConfig configures axis categories and optional labels.
type DataVisualizationAxisConfig struct {
	// Categories defines valid data point labels and their left-to-right display
	// order. Slack requires each category label to be at most 20 characters.
	Categories []string `json:"categories"`
	// XLabel is an optional descriptive title displayed below the x-axis. Slack
	// requires a maximum of 50 characters.
	XLabel string `json:"x_label,omitempty"`
	// YLabel is an optional descriptive title displayed beside the y-axis. Slack
	// requires a maximum of 50 characters.
	YLabel string `json:"y_label,omitempty"`
}

// NewDataVisualizationAxisConfig returns an axis configuration with the given
// categories.
func NewDataVisualizationAxisConfig(categories ...string) DataVisualizationAxisConfig {
	return DataVisualizationAxisConfig{Categories: append([]string{}, categories...)}
}

// WithXLabel sets the x-axis label.
func (c DataVisualizationAxisConfig) WithXLabel(label string) DataVisualizationAxisConfig {
	c.XLabel = label
	return c
}

// WithYLabel sets the y-axis label.
func (c DataVisualizationAxisConfig) WithYLabel(label string) DataVisualizationAxisConfig {
	c.YLabel = label
	return c
}

// DataVisualizationPieChart is a pie chart payload.
type DataVisualizationPieChart struct {
	Type DataVisualizationChartType `json:"type"`
	// Segments are the labeled slices that make up the pie. Slack requires 1 to
	// 6 segments.
	Segments []DataVisualizationSegment `json:"segments"`
}

// DataVisualizationChartType returns the chart variant.
func (c DataVisualizationPieChart) DataVisualizationChartType() DataVisualizationChartType {
	return c.Type
}

// NewDataVisualizationPieChart returns a pie chart with the given segments.
func NewDataVisualizationPieChart(segments ...DataVisualizationSegment) *DataVisualizationPieChart {
	return &DataVisualizationPieChart{
		Type:     DataVisualizationChartPie,
		Segments: append([]DataVisualizationSegment{}, segments...),
	}
}

// DataVisualizationBarChart is a bar chart payload.
type DataVisualizationBarChart struct {
	Type DataVisualizationChartType `json:"type"`
	// Series are plotted as bar groups. Slack requires 1 to 6 series; for
	// multiple series, bars are grouped by data point label.
	Series []DataVisualizationDataSeries `json:"series"`
	// AxisConfig defines x-axis categories and axis titles. Slack requires this
	// field for bar charts.
	AxisConfig DataVisualizationAxisConfig `json:"axis_config"`
}

// DataVisualizationChartType returns the chart variant.
func (c DataVisualizationBarChart) DataVisualizationChartType() DataVisualizationChartType {
	return c.Type
}

// NewDataVisualizationBarChart returns a bar chart.
func NewDataVisualizationBarChart(axisConfig DataVisualizationAxisConfig, series ...DataVisualizationDataSeries) *DataVisualizationBarChart {
	return &DataVisualizationBarChart{
		Type:       DataVisualizationChartBar,
		Series:     append([]DataVisualizationDataSeries{}, series...),
		AxisConfig: axisConfig,
	}
}

// DataVisualizationAreaChart is an area chart payload.
type DataVisualizationAreaChart struct {
	Type DataVisualizationChartType `json:"type"`
	// Series are plotted as filled areas. Slack requires 1 to 6 series and
	// layers them in array order, with the first series at the back.
	Series []DataVisualizationDataSeries `json:"series"`
	// AxisConfig defines x-axis categories and axis titles. Slack requires this
	// field for area charts.
	AxisConfig DataVisualizationAxisConfig `json:"axis_config"`
}

// DataVisualizationChartType returns the chart variant.
func (c DataVisualizationAreaChart) DataVisualizationChartType() DataVisualizationChartType {
	return c.Type
}

// NewDataVisualizationAreaChart returns an area chart.
func NewDataVisualizationAreaChart(axisConfig DataVisualizationAxisConfig, series ...DataVisualizationDataSeries) *DataVisualizationAreaChart {
	return &DataVisualizationAreaChart{
		Type:       DataVisualizationChartArea,
		Series:     append([]DataVisualizationDataSeries{}, series...),
		AxisConfig: axisConfig,
	}
}

// DataVisualizationLineChart is a line chart payload.
type DataVisualizationLineChart struct {
	Type DataVisualizationChartType `json:"type"`
	// Series are plotted as lines. Slack requires 1 to 6 series.
	Series []DataVisualizationDataSeries `json:"series"`
	// AxisConfig defines x-axis categories and axis titles. Slack requires this
	// field for line charts.
	AxisConfig DataVisualizationAxisConfig `json:"axis_config"`
}

// DataVisualizationChartType returns the chart variant.
func (c DataVisualizationLineChart) DataVisualizationChartType() DataVisualizationChartType {
	return c.Type
}

// NewDataVisualizationLineChart returns a line chart.
func NewDataVisualizationLineChart(axisConfig DataVisualizationAxisConfig, series ...DataVisualizationDataSeries) *DataVisualizationLineChart {
	return &DataVisualizationLineChart{
		Type:       DataVisualizationChartLine,
		Series:     append([]DataVisualizationDataSeries{}, series...),
		AxisConfig: axisConfig,
	}
}

// DataVisualizationBlock defines a block that displays data as a pie, bar,
// area, or line chart.
//
// More Information: https://docs.slack.dev/reference/block-kit/blocks/data-visualization-block/
type DataVisualizationBlock struct {
	Type    MessageBlockType `json:"type"`
	BlockID string           `json:"block_id,omitempty"`
	// Title is the short label displayed above the chart. Slack requires a
	// maximum of 50 characters.
	Title string `json:"title"`
	// Chart is the chart-specific payload. Slack requires one of pie, bar, area,
	// or line.
	Chart DataVisualizationChart `json:"chart"`
}

// BlockType returns the type of the block.
func (s DataVisualizationBlock) BlockType() MessageBlockType {
	return s.Type
}

// ID returns the ID of the block.
func (s DataVisualizationBlock) ID() string {
	return s.BlockID
}

// Validate checks whether the block satisfies Slack's documented data
// visualization constraints.
func (s DataVisualizationBlock) Validate() error {
	if s.Type != MBTDataVisualization {
		return fmt.Errorf("type must be %q", MBTDataVisualization)
	}
	if s.Title == "" {
		return fmt.Errorf("title must have a minimum length of 1")
	}
	if runeLen(s.Title) > 50 {
		return fmt.Errorf("title cannot be longer than 50 characters")
	}
	if isNilDataVisualizationChart(s.Chart) {
		return fmt.Errorf("chart is required")
	}

	switch chart := s.Chart.(type) {
	case *DataVisualizationPieChart:
		return validateDataVisualizationPieChart(chart)
	case DataVisualizationPieChart:
		return validateDataVisualizationPieChart(&chart)
	case *DataVisualizationBarChart:
		return validateDataVisualizationSeriesChart(DataVisualizationChartBar, chart.Type, chart.Series, chart.AxisConfig)
	case DataVisualizationBarChart:
		return validateDataVisualizationSeriesChart(DataVisualizationChartBar, chart.Type, chart.Series, chart.AxisConfig)
	case *DataVisualizationAreaChart:
		return validateDataVisualizationSeriesChart(DataVisualizationChartArea, chart.Type, chart.Series, chart.AxisConfig)
	case DataVisualizationAreaChart:
		return validateDataVisualizationSeriesChart(DataVisualizationChartArea, chart.Type, chart.Series, chart.AxisConfig)
	case *DataVisualizationLineChart:
		return validateDataVisualizationSeriesChart(DataVisualizationChartLine, chart.Type, chart.Series, chart.AxisConfig)
	case DataVisualizationLineChart:
		return validateDataVisualizationSeriesChart(DataVisualizationChartLine, chart.Type, chart.Series, chart.AxisConfig)
	default:
		return fmt.Errorf("unsupported data_visualization chart type %q", s.Chart.DataVisualizationChartType())
	}
}

func isNilDataVisualizationChart(chart DataVisualizationChart) bool {
	switch chart := chart.(type) {
	case nil:
		return true
	case *DataVisualizationPieChart:
		return chart == nil
	case *DataVisualizationBarChart:
		return chart == nil
	case *DataVisualizationAreaChart:
		return chart == nil
	case *DataVisualizationLineChart:
		return chart == nil
	default:
		return false
	}
}

func validateDataVisualizationPieChart(chart *DataVisualizationPieChart) error {
	if chart.Type != DataVisualizationChartPie {
		return fmt.Errorf("chart type must be %q", DataVisualizationChartPie)
	}
	if len(chart.Segments) < 1 {
		return fmt.Errorf("pie chart must have at least 1 segment")
	}
	if len(chart.Segments) > 6 {
		return fmt.Errorf("pie chart cannot have more than 6 segments")
	}
	for i, segment := range chart.Segments {
		if segment.Label == "" {
			return fmt.Errorf("segment %d label must have a minimum length of 1", i)
		}
		if runeLen(segment.Label) > 20 {
			return fmt.Errorf("segment %d label cannot be longer than 20 characters", i)
		}
		if segment.Value <= 0 {
			return fmt.Errorf("segment %d value must be greater than 0", i)
		}
	}
	return nil
}

func validateDataVisualizationSeriesChart(
	expectedType DataVisualizationChartType,
	actualType DataVisualizationChartType,
	series []DataVisualizationDataSeries,
	axisConfig DataVisualizationAxisConfig,
) error {
	if actualType != expectedType {
		return fmt.Errorf("chart type must be %q", expectedType)
	}
	if len(series) < 1 {
		return fmt.Errorf("%s chart must have at least 1 series", expectedType)
	}
	if len(series) > 6 {
		return fmt.Errorf("%s chart cannot have more than 6 series", expectedType)
	}
	if len(axisConfig.Categories) == 0 {
		return fmt.Errorf("axis_config.categories must have at least 1 category")
	}
	if runeLen(axisConfig.XLabel) > 50 {
		return fmt.Errorf("axis_config.x_label cannot be longer than 50 characters")
	}
	if runeLen(axisConfig.YLabel) > 50 {
		return fmt.Errorf("axis_config.y_label cannot be longer than 50 characters")
	}

	categories := make(map[string]struct{}, len(axisConfig.Categories))
	for i, category := range axisConfig.Categories {
		if category == "" {
			return fmt.Errorf("axis_config.categories[%d] must have a minimum length of 1", i)
		}
		if runeLen(category) > 20 {
			return fmt.Errorf("axis_config.categories[%d] cannot be longer than 20 characters", i)
		}
		if _, exists := categories[category]; exists {
			return fmt.Errorf("axis_config.categories must not contain duplicate category %q", category)
		}
		categories[category] = struct{}{}
	}

	seriesNames := make(map[string]struct{}, len(series))
	for i, s := range series {
		if s.Name == "" {
			return fmt.Errorf("series %d name must have a minimum length of 1", i)
		}
		if runeLen(s.Name) > 20 {
			return fmt.Errorf("series %d name cannot be longer than 20 characters", i)
		}
		if _, exists := seriesNames[s.Name]; exists {
			return fmt.Errorf("series names must be unique: %q", s.Name)
		}
		seriesNames[s.Name] = struct{}{}

		if len(s.Data) < 1 {
			return fmt.Errorf("series %d data must have at least 1 point", i)
		}
		if len(s.Data) > 20 {
			return fmt.Errorf("series %d data cannot have more than 20 points", i)
		}
		if len(s.Data) != len(axisConfig.Categories) {
			return fmt.Errorf("series %d data must contain exactly one point for every category", i)
		}

		seenLabels := make(map[string]struct{}, len(s.Data))
		for j, point := range s.Data {
			if point.Label == "" {
				return fmt.Errorf("series %d data point %d label must have a minimum length of 1", i, j)
			}
			if runeLen(point.Label) > 20 {
				return fmt.Errorf("series %d data point %d label cannot be longer than 20 characters", i, j)
			}
			if _, exists := categories[point.Label]; !exists {
				return fmt.Errorf("series %d data point %d label %q must match axis_config.categories", i, j, point.Label)
			}
			if _, exists := seenLabels[point.Label]; exists {
				return fmt.Errorf("series %d data must not contain duplicate label %q", i, point.Label)
			}
			seenLabels[point.Label] = struct{}{}
		}
	}

	return nil
}

func runeLen(s string) int {
	return utf8.RuneCountInString(s)
}

// UnmarshalJSON parses the chart-specific payload.
func (s *DataVisualizationBlock) UnmarshalJSON(data []byte) error {
	var raw struct {
		Type    MessageBlockType `json:"type"`
		BlockID string           `json:"block_id"`
		Title   string           `json:"title"`
		Chart   json.RawMessage  `json:"chart"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	var probe struct {
		Type DataVisualizationChartType `json:"type"`
	}
	if err := json.Unmarshal(raw.Chart, &probe); err != nil {
		return err
	}

	var chart DataVisualizationChart
	switch probe.Type {
	case DataVisualizationChartPie:
		chart = &DataVisualizationPieChart{}
	case DataVisualizationChartBar:
		chart = &DataVisualizationBarChart{}
	case DataVisualizationChartArea:
		chart = &DataVisualizationAreaChart{}
	case DataVisualizationChartLine:
		chart = &DataVisualizationLineChart{}
	default:
		return fmt.Errorf("unsupported data_visualization chart type %q", probe.Type)
	}

	if err := json.Unmarshal(raw.Chart, chart); err != nil {
		return err
	}

	s.Type = raw.Type
	s.BlockID = raw.BlockID
	s.Title = raw.Title
	s.Chart = chart
	return nil
}

// DataVisualizationBlockOption configures optional fields on a new
// DataVisualizationBlock.
type DataVisualizationBlockOption func(*DataVisualizationBlock)

// DataVisualizationBlockOptionBlockID sets the block ID.
func DataVisualizationBlockOptionBlockID(blockID string) DataVisualizationBlockOption {
	return func(b *DataVisualizationBlock) { b.BlockID = blockID }
}

// NewDataVisualizationBlock returns a new DataVisualizationBlock with the given
// title and chart payload.
func NewDataVisualizationBlock(title string, chart DataVisualizationChart, options ...DataVisualizationBlockOption) *DataVisualizationBlock {
	block := &DataVisualizationBlock{
		Type:  MBTDataVisualization,
		Title: title,
		Chart: chart,
	}
	for _, opt := range options {
		if opt != nil {
			opt(block)
		}
	}
	return block
}
