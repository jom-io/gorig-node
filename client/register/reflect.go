package register

import (
	"fmt"
	"reflect"
	"strings"
)

// BuildTypeSchema recursively parses type info (struct / slice / map / base).
// It keeps a cache to avoid infinite recursion on self-referencing structs.
func BuildTypeSchema(t reflect.Type) *TypeSchema {
	cache := map[reflect.Type]*TypeSchema{}
	inProgress := map[reflect.Type]bool{}
	return buildTypeSchema(t, cache, inProgress)
}

func buildTypeSchema(t reflect.Type, cache map[reflect.Type]*TypeSchema, inProgress map[reflect.Type]bool) *TypeSchema {
	// Strip pointers
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Break cycles early with shallow placeholder
	if inProgress[t] {
		return &TypeSchema{
			Kind: kindString(t),
			Name: t.String(),
		}
	}

	// Return cached result if available
	if ts, ok := cache[t]; ok {
		return ts
	}

	inProgress[t] = true

	switch t.Kind() {

	case reflect.Struct:
		ts := &TypeSchema{
			Kind: "struct",
			Name: t.String(),
		}
		cache[t] = ts

		fields := []FieldSchema{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)

			if f.PkgPath != "" { // skip unexported fields
				continue
			}

			jsonTag := parseJsonTag(f.Tag.Get("json"))

			fields = append(fields, FieldSchema{
				Name:     f.Name,
				Type:     f.Type.String(),
				JsonTag:  jsonTag,
				Embedded: f.Anonymous,
				Schema:   buildTypeSchema(f.Type, cache, inProgress),
			})
		}
		ts.Fields = fields
		delete(inProgress, t)
		return ts

	case reflect.Slice:
		ts := &TypeSchema{
			Kind: "slice",
		}
		cache[t] = ts
		elemType := t.Elem()
		elemSchema := buildTypeSchema(elemType, cache, inProgress)
		ts.Elem = ensureElemSchema(elemSchema, elemType)
		delete(inProgress, t)
		return ts

	case reflect.Map:
		ts := &TypeSchema{
			Kind: "map",
		}
		cache[t] = ts
		elemType := t.Elem()
		elemSchema := buildTypeSchema(elemType, cache, inProgress)
		ts.Elem = ensureElemSchema(elemSchema, elemType)
		delete(inProgress, t)
		return ts

	default:
		ts := &TypeSchema{
			Kind: "base",
			Name: t.String(),
		}
		cache[t] = ts
		delete(inProgress, t)
		return ts
	}
}

func kindString(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Struct:
		return "struct"
	case reflect.Map:
		return "map"
	case reflect.Slice:
		return "slice"
	default:
		return "base"
	}
}

// ensureElemSchema guarantees slice/map Elem is non-nil, even for self-referential types.
func ensureElemSchema(elem *TypeSchema, rt reflect.Type) *TypeSchema {
	if elem != nil {
		return elem
	}
	base := stripPtr(rt)
	return &TypeSchema{
		Kind: kindString(base),
		Name: base.String(),
	}
}

func stripPtr(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// validateSchemaType walks the type and rejects unsupported shapes for schema generation.
// - map keys must be string (json requirement)
// - recursion guarded by visited map
func validateSchemaType(t reflect.Type) error {
	return validateSchemaTypeInner(t, map[reflect.Type]bool{})
}

func validateSchemaTypeInner(t reflect.Type, visited map[reflect.Type]bool) error {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if visited[t] {
		return nil
	}
	visited[t] = true

	switch t.Kind() {
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return fmt.Errorf("map key must be string, got %s", t.Key().String())
		}
		return validateSchemaTypeInner(t.Elem(), visited)
	case reflect.Slice:
		return validateSchemaTypeInner(t.Elem(), visited)
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" { // skip unexported
				continue
			}
			if err := validateSchemaTypeInner(f.Type, visited); err != nil {
				return fmt.Errorf("field %s: %w", f.Name, err)
			}
		}
	}

	delete(visited, t)
	return nil
}

func parseJsonTag(tag string) (name string) {
	if tag == "" {
		return ""
	}
	parts := strings.Split(tag, ",")
	name = parts[0]
	return
}
