package register

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

func makeWrapper(service, method string, fn interface{}, userArgNames []string) (
	meta MethodMeta, api ApiInfo, err error,
) {
	v := reflect.ValueOf(fn)
	t := v.Type()

	if t.Kind() != reflect.Func {
		err = fmt.Errorf("%s.%s is not a function", service, method)
		return
	}

	meta.FnValue = v
	meta.FnType = t

	numIn := t.NumIn()
	numOut := t.NumOut()

	// ---------------------------
	// Input parameters: ctx validation
	// ---------------------------
	meta.HasCtx = false
	if numIn > 0 && isCtxType(t.In(0)) {
		meta.HasCtx = true
		meta.CtxType = t.In(0)
	}

	meta.InTypes = make([]reflect.Type, numIn)
	for i := 0; i < numIn; i++ {
		pt := t.In(i)
		// ctx must only appear as the first argument
		if i > 0 && isCtxType(pt) {
			err = fmt.Errorf("%s.%s ctx must be the first parameter", service, method)
			return
		}
		meta.InTypes[i] = pt
	}

	// ---------------------------
	// Output parameters: error position validation
	// ---------------------------
	meta.OutTypes = make([]reflect.Type, numOut)
	copy(meta.OutTypes, typeList(t))

	errorCount := 0
	errorIndex := -1

	for i := 0; i < numOut; i++ {
		outT := t.Out(i)

		if outT.String() == "error" {
			errorCount++
			errorIndex = i
		}
	}

	// ❌ Multiple errors are not allowed
	if errorCount > 1 {
		err = fmt.Errorf("%s.%s cannot have multiple error returns", service, method)
		return
	}

	// ❌ If error exists, it must be the last return value
	if errorCount == 1 && errorIndex != numOut-1 {
		err = fmt.Errorf("%s.%s error must be the last return value", service, method)
		return
	}

	// ---------------------------
	// Validate schema compatibility (map key, unsupported kinds, etc.)
	// ---------------------------
	if err = validateMethodSchema(meta); err != nil {
		err = fmt.Errorf("%s.%s schema validation failed: %w", service, method, err)
		return
	}

	// No-error signatures accepted:
	// func Foo()
	// func Foo() (A, B)
	// Error-as-last accepted:
	// func Foo() error
	// func Foo() (A, B, error)

	// ---------------------------
	// Build ApiInfo schema
	// ---------------------------
	api.Service = service
	api.Method = method
	api.HasCtx = meta.HasCtx

	// 1. Decide argument names
	argCount := numIn
	if meta.HasCtx {
		argCount--
	}

	meta.ArgNames = make([]string, argCount)

	for i := 0; i < argCount; i++ {
		if i < len(userArgNames) && userArgNames[i] != "" {
			meta.ArgNames[i] = userArgNames[i]
		} else {
			paramType := t.In(i + boolToInt(meta.HasCtx))
			meta.ArgNames[i] = autoArgName(paramType)
		}
	}

	// 2. Fill ApiInfo.Args
	api.Args = []ArgDesc{}
	argIdx := 0
	for i := 0; i < numIn; i++ {
		if i == 0 && meta.HasCtx {
			continue
		}
		api.Args = append(api.Args, ArgDesc{
			Index: argIdx,
			Name:  meta.ArgNames[argIdx],
			Type:  t.In(i).String(),
		})
		argIdx++
	}

	// 3. Fill ApiInfo.Returns
	api.Returns = []ReturnDesc{}
	for i := 0; i < numOut; i++ {
		outT := t.Out(i)
		isErr := (outT.String() == "error")

		api.Returns = append(api.Returns, ReturnDesc{
			Index:   i,
			Type:    outT.String(),
			IsError: isErr,
		})
	}

	// fill api.ArgSchemas
	api.ArgSchemas = make([]*TypeSchema, len(api.Args))
	for i, _ := range api.Args {
		argT := meta.InTypes[i+boolToInt(meta.HasCtx)]
		api.ArgSchemas[i] = BuildTypeSchema(argT)
	}

	// fill api.ReturnSchemas
	api.ReturnSchemas = make([]*TypeSchema, len(api.Returns))
	for i, _ := range api.Returns {
		outT := meta.OutTypes[i]
		api.ReturnSchemas[i] = BuildTypeSchema(outT)
	}

	return
}

func typeList(t reflect.Type) []reflect.Type {
	arr := make([]reflect.Type, t.NumOut())
	for i := 0; i < t.NumOut(); i++ {
		arr[i] = t.Out(i)
	}
	return arr
}

func isCtxType(t reflect.Type) bool {
	ctxT := reflect.TypeOf((*context.Context)(nil)).Elem()
	return t == ctxT
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// validateMethodSchema enforces constraints for SDK serialization.
// - map key must be string (JSON requirement, aligns with generator)
// - skips ctx and error return
func validateMethodSchema(meta MethodMeta) error {
	// args
	for i, t := range meta.InTypes {
		if meta.HasCtx && i == 0 {
			continue
		}
		if err := validateSchemaType(t); err != nil {
			return fmt.Errorf("arg %d: %w", i, err)
		}
	}

	// returns
	for i, t := range meta.OutTypes {
		if t.String() == "error" {
			continue
		}
		if err := validateSchemaType(t); err != nil {
			return fmt.Errorf("return %d: %w", i, err)
		}
	}
	return nil
}

func autoArgName(t reflect.Type) string {
	// Unwrap pointer
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// slice
	if t.Kind() == reflect.Slice {
		return pluralize(autoArgName(t.Elem()))
	}

	// map
	if t.Kind() == reflect.Map {
		return autoArgName(t.Elem()) + "Map"
	}

	// struct or named type
	name := t.Name()
	if name != "" {
		return lowerFirst(name)
	}

	// Fallback for basic types
	switch t.Kind() {
	case reflect.String:
		return "s"
	case reflect.Int, reflect.Int64:
		return "n"
	case reflect.Bool:
		return "flag"
	case reflect.Float32, reflect.Float64:
		return "f"
	}

	return "arg"
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func pluralize(s string) string {
	if strings.HasSuffix(s, "s") {
		return s
	}
	return s + "s"
}
