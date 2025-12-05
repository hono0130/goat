package e2egen

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
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

	var b strings.Builder
	fmt.Fprintf(&b, "&%s.%s{\n", pkgAlias, typeName)
	for _, k := range keys {
		fmt.Fprintf(&b, "\t\t\t\t%s: %s,\n", k, FormatValue(data[k]))
	}
	b.WriteString("\t\t\t}")
	return b.String()
}

func FormatValue(value any) string {
	if value == nil {
		return "nil"
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map:
		if v.IsNil() {
			return "nil"
		}
	case reflect.String:
		return fmt.Sprintf("%q", value)
	case reflect.Bool:
		return fmt.Sprintf("%t", value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return fmt.Sprintf("%d", value)
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%v", value)
	}
	return fmt.Sprintf("%#v", value)
}
