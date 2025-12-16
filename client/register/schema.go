package register

type Schema struct {
}

// TypeSchema describes any serializable type (struct/slice/map/base).
type TypeSchema struct {
	Kind   string        `json:"kind"`   // struct/map/slice/base
	Name   string        `json:"name"`   // type name
	Fields []FieldSchema `json:"fields"` // only struct
	Elem   *TypeSchema   `json:"elem"`   // slice/map element
}

type FieldSchema struct {
	Name     string      `json:"name"`               // field name
	Type     string      `json:"type"`               // full type string
	JsonTag  string      `json:"json_tag"`           // parsed json tag
	Embedded bool        `json:"embedded,omitempty"` // whether the field is anonymous (embedded)
	Schema   *TypeSchema `json:"schema,omitempty"`   // nested schema
}

type ArgDesc struct {
	Index int    `json:"index"`
	Name  string `json:"name"`
	Type  string `json:"type"`
}

type ReturnDesc struct {
	Index   int    `json:"index"`
	Type    string `json:"type"`
	IsError bool   `json:"is_error"`
}

type ApiInfo struct {
	Version       string        `json:"version"`
	Environment   string        `json:"env,omitempty"`
	Service       string        `json:"service"`
	Method        string        `json:"method"`
	HasCtx        bool          `json:"has_ctx"`
	Args          []ArgDesc     `json:"args"`
	Returns       []ReturnDesc  `json:"returns"`
	ArgSchemas    []*TypeSchema `json:"arg_schemas"`
	ReturnSchemas []*TypeSchema `json:"return_schemas"`
}

type CallDesc struct {
	CtxType      string   // "*gin.Context" or "context.Context"
	ArgTypes     []string // original parameter types (ordered)
	RetTypes     []string // original return types (ordered)
	ErrorIndexes []int    // positions of error returns
}
