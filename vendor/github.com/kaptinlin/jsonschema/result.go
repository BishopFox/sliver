package jsonschema

import "github.com/kaptinlin/go-i18n"

// EvaluationError represents an error that occurred during schema evaluation
type EvaluationError struct {
	Keyword string         `json:"keyword"`
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Params  map[string]any `json:"params"`
}

// NewEvaluationError creates a new evaluation error with the specified details
func NewEvaluationError(keyword string, code string, message string, params ...map[string]any) *EvaluationError {
	if len(params) > 0 {
		return &EvaluationError{
			Keyword: keyword,
			Code:    code,
			Message: message,
			Params:  params[0],
		}
	}
	return &EvaluationError{
		Keyword: keyword,
		Code:    code,
		Message: message,
	}
}

// Error returns a string representation of the evaluation error.
func (e *EvaluationError) Error() string {
	return replace(e.Message, e.Params)
}

// Localize returns a localized error message using the provided localizer.
func (e *EvaluationError) Localize(localizer *i18n.Localizer) string {
	if localizer != nil {
		return localizer.Get(e.Code, i18n.Vars(e.Params))
	}
	return e.Error()
}

// Flag represents a simple validation result with just validity status
type Flag struct {
	Valid bool `json:"valid"`
}

// List represents a flat list of validation errors
type List struct {
	Valid            bool              `json:"valid"`
	EvaluationPath   string            `json:"evaluationPath"`
	SchemaLocation   string            `json:"schemaLocation"`
	InstanceLocation string            `json:"instanceLocation"`
	Annotations      map[string]any    `json:"annotations,omitempty"`
	Errors           map[string]string `json:"errors,omitempty"`
	Details          []List            `json:"details,omitempty"`
}

// EvaluationResult represents the complete result of a schema validation
type EvaluationResult struct {
	schema           *Schema                     `json:"-"`
	Valid            bool                        `json:"valid"`
	EvaluationPath   string                      `json:"evaluationPath"`
	SchemaLocation   string                      `json:"schemaLocation"`
	InstanceLocation string                      `json:"instanceLocation"`
	Annotations      map[string]any              `json:"annotations,omitempty"`
	Errors           map[string]*EvaluationError `json:"errors,omitempty"` // Store error messages here
	Details          []*EvaluationResult         `json:"details,omitempty"`
}

// NewEvaluationResult creates a new evaluation result for the given schema
func NewEvaluationResult(schema *Schema) *EvaluationResult {
	e := &EvaluationResult{
		schema: schema,
		Valid:  true,
	}
	//nolint:errcheck
	e.CollectAnnotations()

	return e
}

// SetEvaluationPath sets the evaluation path for this result
func (e *EvaluationResult) SetEvaluationPath(evaluationPath string) *EvaluationResult {
	e.EvaluationPath = evaluationPath

	return e
}

// Error returns a string representation of the evaluation failure.
func (e *EvaluationResult) Error() string {
	return "evaluation failed"
}

// SetSchemaLocation sets the schema location for this result
func (e *EvaluationResult) SetSchemaLocation(location string) *EvaluationResult {
	e.SchemaLocation = location

	return e
}

// SetInstanceLocation sets the instance location for this result
func (e *EvaluationResult) SetInstanceLocation(instanceLocation string) *EvaluationResult {
	e.InstanceLocation = instanceLocation

	return e
}

// SetInvalid marks this result as invalid
func (e *EvaluationResult) SetInvalid() *EvaluationResult {
	e.Valid = false

	return e
}

// IsValid returns whether this result is valid
func (e *EvaluationResult) IsValid() bool {
	return e.Valid
}

// AddError adds an evaluation error to this result
func (e *EvaluationResult) AddError(err *EvaluationError) *EvaluationResult {
	if e.Errors == nil {
		e.Errors = make(map[string]*EvaluationError)
	}

	if e.Valid {
		e.Valid = false
	}

	e.Errors[err.Keyword] = err
	return e
}

// AddDetail adds a detailed evaluation result to this result
func (e *EvaluationResult) AddDetail(detail *EvaluationResult) *EvaluationResult {
	if e.Details == nil {
		e.Details = make([]*EvaluationResult, 0)
	}

	e.Details = append(e.Details, detail)
	return e
}

// AddAnnotation adds an annotation to this result
func (e *EvaluationResult) AddAnnotation(keyword string, annotation any) *EvaluationResult {
	if e.Annotations == nil {
		e.Annotations = make(map[string]any)
	}

	e.Annotations[keyword] = annotation
	return e
}

// CollectAnnotations collects annotations from child results
func (e *EvaluationResult) CollectAnnotations() *EvaluationResult {
	if e.Annotations == nil {
		e.Annotations = make(map[string]any)
	}

	if e.schema.Title != nil {
		e.Annotations["title"] = e.schema.Title
	}
	if e.schema.Description != nil {
		e.Annotations["description"] = e.schema.Description
	}
	if e.schema.Default != nil {
		e.Annotations["default"] = e.schema.Default
	}
	if e.schema.Deprecated != nil {
		e.Annotations["deprecated"] = e.schema.Deprecated
	}
	if e.schema.ReadOnly != nil {
		e.Annotations["readOnly"] = e.schema.ReadOnly
	}
	if e.schema.WriteOnly != nil {
		e.Annotations["writeOnly"] = e.schema.WriteOnly
	}
	if e.schema.Examples != nil {
		e.Annotations["examples"] = e.schema.Examples
	}

	return e
}

// ToFlag converts EvaluationResult to a simple Flag struct
func (e *EvaluationResult) ToFlag() *Flag {
	return &Flag{
		Valid: e.Valid,
	}
}

// ToList converts the evaluation results into a list format with optional hierarchy
// includeHierarchy is variadic; if not provided, it defaults to true
func (e *EvaluationResult) ToList(includeHierarchy ...bool) *List {
	// Set default value for includeHierarchy to true
	hierarchyIncluded := true
	if len(includeHierarchy) > 0 {
		hierarchyIncluded = includeHierarchy[0]
	}

	return e.ToLocalizeList(nil, hierarchyIncluded)
}

// ToLocalizeList converts the evaluation results into a list format with optional hierarchy with localization.
// includeHierarchy is variadic; if not provided, it defaults to true
func (e *EvaluationResult) ToLocalizeList(localizer *i18n.Localizer, includeHierarchy ...bool) *List {
	// Set default value for includeHierarchy to true
	hierarchyIncluded := true
	if len(includeHierarchy) > 0 {
		hierarchyIncluded = includeHierarchy[0]
	}

	list := &List{
		Valid:            e.Valid,
		EvaluationPath:   e.EvaluationPath,
		SchemaLocation:   e.SchemaLocation,
		InstanceLocation: e.InstanceLocation,
		Annotations:      e.Annotations,
		Errors:           e.convertErrors(localizer),
		Details:          make([]List, 0),
	}

	if hierarchyIncluded {
		for _, detail := range e.Details {
			childList := detail.ToLocalizeList(localizer, true) // recursively include hierarchy
			list.Details = append(list.Details, *childList)
		}
	} else {
		e.flattenDetailsToList(localizer, list, e.Details) // flat structure
	}

	return list
}

func (e *EvaluationResult) flattenDetailsToList(localizer *i18n.Localizer, list *List, details []*EvaluationResult) {
	for _, detail := range details {
		flatDetail := List{
			Valid:            detail.Valid,
			EvaluationPath:   detail.EvaluationPath,
			SchemaLocation:   detail.SchemaLocation,
			InstanceLocation: detail.InstanceLocation,
			Annotations:      detail.Annotations,
			Errors:           detail.convertErrors(localizer),
		}
		list.Details = append(list.Details, flatDetail)

		if len(detail.Details) > 0 {
			e.flattenDetailsToList(localizer, list, detail.Details)
		}
	}
}

func (e *EvaluationResult) convertErrors(localizer *i18n.Localizer) map[string]string {
	errors := make(map[string]string)
	for key, err := range e.Errors {
		if localizer != nil {
			errors[key] = err.Localize(localizer)
		} else {
			errors[key] = err.Error()
		}
	}
	return errors
}

// GetDetailedErrors collects all detailed validation errors from the nested Details hierarchy.
// This method helps users access specific validation failures that might be buried in nested structures.
// Returns a map where keys are field paths and values are the most specific error messages.
// For localized messages, pass a localizer; for default English messages, call without arguments.
func (e *EvaluationResult) GetDetailedErrors(localizer ...*i18n.Localizer) map[string]string {
	var loc *i18n.Localizer
	if len(localizer) > 0 {
		loc = localizer[0]
	}

	detailedErrors := make(map[string]string)
	e.collectDetailedErrors(detailedErrors, loc, "")
	return detailedErrors
}

// collectDetailedErrors recursively traverses the Details hierarchy to collect leaf-level errors.
// This is a private helper method that implements the core logic for detailed error collection.
func (e *EvaluationResult) collectDetailedErrors(collector map[string]string, localizer *i18n.Localizer, basePath string) {
	// Collect errors from current level
	if len(e.Errors) > 0 {
		currentPath := basePath + e.InstanceLocation
		for key, err := range e.Errors {
			fieldPath := currentPath
			if fieldPath != "" && key != "" {
				fieldPath = fieldPath + "/" + key
			} else if key != "" {
				fieldPath = key
			}

			if localizer != nil {
				collector[fieldPath] = err.Localize(localizer)
			} else {
				collector[fieldPath] = err.Error()
			}
		}
	}

	// Recursively collect from Details
	for _, detail := range e.Details {
		detail.collectDetailedErrors(collector, localizer, basePath+e.InstanceLocation)
	}
}
