package utils

import (
	"reflect"
	"runtime"
	"strings"
)

func GetFuncName(i interface{}) string {
	fn := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	// Strip package path, keep only the function name
	idx := strings.LastIndex(fn, ".")
	if idx >= 0 {
		return fn[idx+1:]
	}
	return fn
}
