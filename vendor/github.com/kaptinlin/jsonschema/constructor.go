package jsonschema

// Default compiler instance for initializing Schema
var defaultCompiler = NewCompiler()

// SetDefaultCompiler allows setting a custom compiler for the constructor API
func SetDefaultCompiler(c *Compiler) {
	defaultCompiler = c
}

// GetDefaultCompiler returns the current default compiler
func GetDefaultCompiler() *Compiler {
	return defaultCompiler
}

// Property represents a Schema property definition
type Property struct {
	Name   string
	Schema *Schema
}

// Prop creates a property definition
func Prop(name string, schema *Schema) Property {
	return Property{Name: name, Schema: schema}
}

// Object creates an object Schema with properties and keywords
func Object(items ...any) *Schema {
	schema := &Schema{Type: SchemaType{"object"}}

	var properties []Property
	var keywords []Keyword

	// Separate properties and keywords
	for _, item := range items {
		switch v := item.(type) {
		case Property:
			properties = append(properties, v)
		case Keyword:
			keywords = append(keywords, v)
		}
	}

	// Set properties
	if len(properties) > 0 {
		props := make(SchemaMap)
		for _, prop := range properties {
			props[prop.Name] = prop.Schema
		}
		schema.Properties = &props
	}

	// Apply keywords
	for _, keyword := range keywords {
		keyword(schema)
	}

	// Initialize Schema to make it directly usable
	schema.initializeSchema(nil, nil)
	return schema
}

// String creates a string Schema with validation keywords
func String(keywords ...Keyword) *Schema {
	schema := &Schema{Type: SchemaType{"string"}}
	for _, keyword := range keywords {
		keyword(schema)
	}
	schema.initializeSchema(nil, nil)
	return schema
}

// Integer creates an integer Schema with validation keywords
func Integer(keywords ...Keyword) *Schema {
	schema := &Schema{Type: SchemaType{"integer"}}
	for _, keyword := range keywords {
		keyword(schema)
	}
	schema.initializeSchema(nil, nil)
	return schema
}

// Number creates a number Schema with validation keywords
func Number(keywords ...Keyword) *Schema {
	schema := &Schema{Type: SchemaType{"number"}}
	for _, keyword := range keywords {
		keyword(schema)
	}
	schema.initializeSchema(nil, nil)
	return schema
}

// Boolean creates a boolean Schema
func Boolean(keywords ...Keyword) *Schema {
	schema := &Schema{Type: SchemaType{"boolean"}}
	for _, keyword := range keywords {
		keyword(schema)
	}
	schema.initializeSchema(nil, nil)
	return schema
}

// Null creates a null Schema
func Null(keywords ...Keyword) *Schema {
	schema := &Schema{Type: SchemaType{"null"}}
	for _, keyword := range keywords {
		keyword(schema)
	}
	schema.initializeSchema(nil, nil)
	return schema
}

// Array creates an array Schema with validation keywords
func Array(keywords ...Keyword) *Schema {
	schema := &Schema{Type: SchemaType{"array"}}
	for _, keyword := range keywords {
		keyword(schema)
	}
	schema.initializeSchema(nil, nil)
	return schema
}

// Any creates a Schema without type restriction
func Any(keywords ...Keyword) *Schema {
	schema := &Schema{}
	for _, keyword := range keywords {
		keyword(schema)
	}
	schema.initializeSchema(nil, nil)
	return schema
}

// Const creates a const Schema
func Const(value any) *Schema {
	schema := &Schema{
		Const: &ConstValue{Value: value, IsSet: true},
	}
	schema.initializeSchema(nil, nil)
	return schema
}

// Enum creates an enum Schema
func Enum(values ...any) *Schema {
	schema := &Schema{Enum: values}
	schema.initializeSchema(nil, nil)
	return schema
}

// OneOf creates a oneOf combination Schema
func OneOf(schemas ...*Schema) *Schema {
	schema := &Schema{OneOf: schemas}
	schema.initializeSchema(nil, nil)
	return schema
}

// AnyOf creates an anyOf combination Schema
func AnyOf(schemas ...*Schema) *Schema {
	schema := &Schema{AnyOf: schemas}
	schema.initializeSchema(nil, nil)
	return schema
}

// AllOf creates an allOf combination Schema
func AllOf(schemas ...*Schema) *Schema {
	schema := &Schema{AllOf: schemas}
	schema.initializeSchema(nil, nil)
	return schema
}

// Not creates a not combination Schema
func Not(schema *Schema) *Schema {
	result := &Schema{Not: schema}
	result.initializeSchema(nil, nil)
	return result
}

// If creates a conditional Schema with if/then/else keywords
func If(condition *Schema) *ConditionalSchema {
	return &ConditionalSchema{condition: condition}
}

// ConditionalSchema represents a conditional schema for if/then/else logic
type ConditionalSchema struct {
	condition *Schema
	then      *Schema
	otherwise *Schema
}

// Then sets the then clause of a conditional schema
func (cs *ConditionalSchema) Then(then *Schema) *ConditionalSchema {
	cs.then = then
	return cs
}

// Else sets the else clause of a conditional schema
func (cs *ConditionalSchema) Else(otherwise *Schema) *Schema {
	cs.otherwise = otherwise
	return cs.ToSchema()
}

// ToSchema converts a conditional schema to a regular schema
func (cs *ConditionalSchema) ToSchema() *Schema {
	schema := &Schema{
		If:   cs.condition,
		Then: cs.then,
		Else: cs.otherwise,
	}
	schema.initializeSchema(nil, nil)
	return schema
}

// Ref creates a reference Schema using $ref keyword
func Ref(ref string) *Schema {
	schema := &Schema{Ref: ref}
	schema.initializeSchema(nil, nil)
	return schema
}
