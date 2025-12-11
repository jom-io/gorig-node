package register

import (
	"errors"
	"fmt"
	"github.com/goccy/go-json"
	"reflect"
)

type WrappedRequest struct {
	Args map[string]json.RawMessage `json:"args"`
}

type WrappedResponse struct {
	Resp  map[string]json.RawMessage `json:"resp"`
	Error string                     `json:"error"`
}

func PackRequest(meta MethodMeta, args []reflect.Value) ([]byte, error) {
	w := WrappedRequest{Args: map[string]json.RawMessage{}}

	for i, v := range args {
		b, err := json.Marshal(v.Interface())
		if err != nil {
			return nil, err
		}
		w.Args[fmt.Sprintf("arg%d", i)] = b
	}

	return json.Marshal(w)
}

func UnpackRequest(meta MethodMeta, body []byte, ctxVal reflect.Value) ([]reflect.Value, error) {
	var w WrappedRequest
	if err := json.Unmarshal(body, &w); err != nil {
		return nil, err
	}

	var inVals []reflect.Value
	if meta.HasCtx {
		inVals = append(inVals, ctxVal)
	}

	argIndex := 0
	for i := 0; i < len(meta.InTypes); i++ {
		if i == 0 && meta.HasCtx {
			continue
		}
		key := fmt.Sprintf("arg%d", argIndex)
		raw := w.Args[key]

		ptr := reflect.New(meta.InTypes[i])
		if err := json.Unmarshal(raw, ptr.Interface()); err != nil {
			return nil, err
		}

		inVals = append(inVals, ptr.Elem())
		argIndex++
	}

	return inVals, nil
}

func PackResponse(meta MethodMeta, results []reflect.Value) ([]byte, error) {
	resp := WrappedResponse{
		Resp: map[string]json.RawMessage{},
	}

	numOut := len(meta.OutTypes)

	// --- Case 1: no return values ---
	if numOut == 0 {
		return json.Marshal(resp)
	}

	lastT := meta.OutTypes[numOut-1]
	hasError := lastT.String() == "error"

	// --- Case 2: has error (as last return) ---
	if hasError {
		errVal := results[numOut-1]
		if !errVal.IsNil() {
			resp.Error = errVal.Interface().(error).Error()
		}
		// Pack normal return values before the error
		for i := 0; i < numOut-1; i++ {
			b, err := json.Marshal(results[i].Interface())
			if err != nil {
				return nil, err
			}
			resp.Resp[fmt.Sprintf("resp%d", i)] = b
		}
		return json.Marshal(resp)
	}

	// --- Case 3: no error return values ---
	for i := 0; i < numOut; i++ {
		b, err := json.Marshal(results[i].Interface())
		if err != nil {
			return nil, err
		}
		resp.Resp[fmt.Sprintf("resp%d", i)] = b
	}

	return json.Marshal(resp)
}

func UnpackResponse(meta MethodMeta, body []byte) ([]reflect.Value, error) {
	var w WrappedResponse
	if err := json.Unmarshal(body, &w); err != nil {
		return nil, err
	}

	numOut := len(meta.OutTypes)
	outVals := make([]reflect.Value, numOut)

	if numOut == 0 {
		return outVals, nil
	}

	lastT := meta.OutTypes[numOut-1]
	hasError := lastT.String() == "error"

	if hasError {
		// Normal return values
		for i := 0; i < numOut-1; i++ {
			ptr := reflect.New(meta.OutTypes[i])
			if err := json.Unmarshal(w.Resp[fmt.Sprintf("resp%d", i)], ptr.Interface()); err != nil {
				return nil, err
			}
			outVals[i] = ptr.Elem()
		}

		// error
		if w.Error == "" {
			outVals[numOut-1] = reflect.Zero(reflect.TypeOf((*error)(nil)).Elem())
		} else {
			outVals[numOut-1] = reflect.ValueOf(errors.New(w.Error))
		}
		return outVals, nil
	}

	// No error
	for i := 0; i < numOut; i++ {
		ptr := reflect.New(meta.OutTypes[i])
		if err := json.Unmarshal(w.Resp[fmt.Sprintf("resp%d", i)], ptr.Interface()); err != nil {
			return nil, err
		}
		outVals[i] = ptr.Elem()
	}
	return outVals, nil
}
