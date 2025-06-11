package boilerplate

// Boilerplate represents a single boilerplate template.
type Boilerplate struct {
	// Name is the unique identifier for the boilerplate.
	Name string
	// Value is the template string of the boilerplate.
	// It can contain variables in the format [[variable_name]] or {{prompt}}.
	Value string
	// Count is the number of times this boilerplate has been used.
	Count int
}
