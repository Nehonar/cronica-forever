package luasv

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Quote devuelve s como literal de cadena Lua válido.
func Quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case 0:
			b.WriteString(`\0`)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// Encode escribe un valor Go como expresión Lua. Admite map[string]any, []any,
// []map[string]any, string, bool, enteros y float64.
func Encode(v any, indent int) string {
	pad := strings.Repeat("\t", indent)
	switch x := v.(type) {
	case nil:
		return "nil"
	case string:
		return Quote(x)
	case bool:
		if x {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var b strings.Builder
		b.WriteString("{\n")
		for _, k := range keys {
			fmt.Fprintf(&b, "%s\t[%s] = %s,\n", pad, Quote(k), Encode(x[k], indent+1))
		}
		b.WriteString(pad + "}")
		return b.String()
	case []map[string]any:
		arr := make([]any, len(x))
		for i := range x {
			arr[i] = x[i]
		}
		return Encode(arr, indent)
	case []any:
		var b strings.Builder
		b.WriteString("{\n")
		for i, e := range x {
			fmt.Fprintf(&b, "%s\t%s, -- [%d]\n", pad, Encode(e, indent+1), i+1)
		}
		b.WriteString(pad + "}")
		return b.String()
	}
	return Quote(fmt.Sprint(v))
}
