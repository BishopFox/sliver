package javaser

// Limits bound allocation, recursion, and input/output consumed by one
// independent stream. Non-positive values passed through WithLimits are
// replaced with the corresponding secure default.
type Limits struct {
	MaxDepth            int
	MaxHandles          int
	MaxClassDescriptors int
	MaxFieldsPerClass   int
	MaxTotalFields      int
	MaxProxyInterfaces  int
	MaxAnnotationItems  int
	MaxArrayLength      int
	MaxStringBytes      int64
	MaxBlockBytes       int64
	MaxTotalBytes       int64
}

// DefaultLimits returns conservative limits suitable for untrusted streams.
func DefaultLimits() Limits {
	return Limits{
		MaxDepth:            256,
		MaxHandles:          100_000,
		MaxClassDescriptors: 4_096,
		MaxFieldsPerClass:   4_096,
		MaxTotalFields:      100_000,
		MaxProxyInterfaces:  4_096,
		MaxAnnotationItems:  100_000,
		MaxArrayLength:      1_000_000,
		MaxStringBytes:      16 << 20,
		MaxBlockBytes:       16 << 20,
		MaxTotalBytes:       64 << 20,
	}
}

func (l Limits) normalized() Limits {
	d := DefaultLimits()
	if l.MaxDepth <= 0 {
		l.MaxDepth = d.MaxDepth
	}
	if l.MaxHandles <= 0 {
		l.MaxHandles = d.MaxHandles
	}
	if l.MaxClassDescriptors <= 0 {
		l.MaxClassDescriptors = d.MaxClassDescriptors
	}
	if l.MaxFieldsPerClass <= 0 {
		l.MaxFieldsPerClass = d.MaxFieldsPerClass
	}
	if l.MaxTotalFields <= 0 {
		l.MaxTotalFields = d.MaxTotalFields
	}
	if l.MaxProxyInterfaces <= 0 {
		l.MaxProxyInterfaces = d.MaxProxyInterfaces
	}
	if l.MaxAnnotationItems <= 0 {
		l.MaxAnnotationItems = d.MaxAnnotationItems
	}
	if l.MaxArrayLength <= 0 {
		l.MaxArrayLength = d.MaxArrayLength
	}
	if l.MaxStringBytes <= 0 {
		l.MaxStringBytes = d.MaxStringBytes
	}
	if l.MaxBlockBytes <= 0 {
		l.MaxBlockBytes = d.MaxBlockBytes
	}
	if l.MaxTotalBytes <= 0 {
		l.MaxTotalBytes = d.MaxTotalBytes
	}
	return l
}

type decoderConfig struct {
	limits   Limits
	resolver ClassDataResolver
}

type encoderConfig struct {
	limits Limits
}

// Option configures a Decoder and/or Encoder.
type Option interface {
	applyDecoder(*decoderConfig)
	applyEncoder(*encoderConfig)
}

type option struct {
	decode func(*decoderConfig)
	encode func(*encoderConfig)
}

func (o option) applyDecoder(c *decoderConfig) {
	if o.decode != nil {
		o.decode(c)
	}
}

func (o option) applyEncoder(c *encoderConfig) {
	if o.encode != nil {
		o.encode(c)
	}
}

// WithLimits applies resource limits to both encoding and decoding.
func WithLimits(limits Limits) Option {
	limits = limits.normalized()
	return option{
		decode: func(c *decoderConfig) { c.limits = limits },
		encode: func(c *encoderConfig) { c.limits = limits },
	}
}

// WithClassDataResolver installs the allowlisted schema resolver used while
// decoding classes with custom writeObject data.
func WithClassDataResolver(resolver ClassDataResolver) Option {
	return option{decode: func(c *decoderConfig) { c.resolver = resolver }}
}
