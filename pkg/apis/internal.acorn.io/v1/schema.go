package v1

type Field struct {
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	Type        FieldType `json:"type,omitempty"`
	Match       bool      `json:"match,omitempty"`
	Optional    bool      `json:"optional,omitempty"`
}

type FieldType struct {
	Kind        string       `json:"kind,omitempty"`
	Object      *Object      `json:"object,omitempty"`
	Array       *Array       `json:"array,omitempty"`
	Constraints []Constraint `json:"constraints,omitempty"`
	Default     string       `json:"default,omitempty"`
	Alternates  []FieldType  `json:"alternates,omitempty"`
}

type Constraint struct {
	Description string     `json:"description,omitempty"`
	Op          string     `json:"op,omitempty"`
	Right       string     `json:"right,omitempty"`
	Type        *FieldType `json:"type,omitempty"`
}

type Object struct {
	Path         string  `json:"path,omitempty"`
	Reference    bool    `json:"reference,omitempty"`
	Description  string  `json:"description,omitempty"`
	Fields       []Field `json:"fields,omitempty"`
	AllowNewKeys bool    `json:"allowNewKeys,omitempty"`
}

type Array struct {
	Types []FieldType `json:"items,omitempty"`
}
