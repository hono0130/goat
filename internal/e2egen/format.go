package e2egen

import (
	"fmt"
	"reflect"
	"sort"
)

func FormatStructLiteral(pkgAlias, typeName string, data map[string]any) string {
	if len(data) == 0 {
		return fmt.Sprintf("&%s.%s{}", pkgAlias, typeName)
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf string
	buf = fmt.Sprintf("&%s.%s{\n", pkgAlias, typeName)
	for _, k := range keys {
		buf += fmt.Sprintf("\t\t\t\t%s: %s,\n", k, FormatValue(data[k]))
	}
	buf += "\t\t\t}"

	return buf
}

func FormatValue(value any) string {
	if value == nil {
		return "nil"
	}

	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map:
		if val.IsNil() {
			return "nil"
		}
	}

	switch v := value.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64, uintptr:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%#v", value)
	}
}
