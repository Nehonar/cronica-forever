// Package luasv lee los archivos SavedVariables que escribe World of Warcraft.
//
// El formato es un subconjunto de Lua: asignaciones de nivel superior
// (`Nombre = valor`) cuyos valores son tablas, cadenas, números, booleanos o nil.
package luasv

import (
	"fmt"
	"strconv"
	"strings"
)

// Table es una tabla Lua: parte de array (claves 1..n implícitas o explícitas) y parte de hash.
type Table struct {
	Array []any
	Hash  map[string]any
}

// Get devuelve el valor de una clave de texto.
func (t *Table) Get(key string) any {
	if t == nil {
		return nil
	}
	return t.Hash[key]
}

// String devuelve el valor de texto de una clave, o "".
func (t *Table) String(key string) string {
	if s, ok := t.Get(key).(string); ok {
		return s
	}
	return ""
}

// Int devuelve el valor numérico entero de una clave, o 0.
func (t *Table) Int(key string) int64 {
	if f, ok := t.Get(key).(float64); ok {
		return int64(f)
	}
	return 0
}

// Table devuelve la subtabla de una clave, o nil.
func (t *Table) Table(key string) *Table {
	if v, ok := t.Get(key).(*Table); ok {
		return v
	}
	return nil
}

// Parse lee todo el archivo y devuelve las variables de nivel superior.
func Parse(src string) (map[string]any, error) {
	p := &parser{src: src}
	out := map[string]any{}
	for {
		p.skip()
		if p.eof() {
			return out, nil
		}
		name, err := p.ident()
		if err != nil {
			return nil, err
		}
		p.skip()
		if !p.consume('=') {
			return nil, p.errf("se esperaba '=' tras %q", name)
		}
		v, err := p.value()
		if err != nil {
			return nil, err
		}
		out[name] = v
		p.skip()
		p.consume(';')
	}
}

type parser struct {
	src string
	pos int
}

func (p *parser) eof() bool { return p.pos >= len(p.src) }

func (p *parser) errf(format string, a ...any) error {
	line := 1 + strings.Count(p.src[:min(p.pos, len(p.src))], "\n")
	return fmt.Errorf("luasv: línea %d: %s", line, fmt.Sprintf(format, a...))
}

func (p *parser) peek() byte {
	if p.eof() {
		return 0
	}
	return p.src[p.pos]
}

func (p *parser) consume(c byte) bool {
	if p.peek() == c {
		p.pos++
		return true
	}
	return false
}

// skip salta espacios y comentarios (`-- ...` y `--[[ ... ]]`).
func (p *parser) skip() {
	for !p.eof() {
		c := p.src[p.pos]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			p.pos++
		case strings.HasPrefix(p.src[p.pos:], "--[["):
			end := strings.Index(p.src[p.pos+4:], "]]")
			if end < 0 {
				p.pos = len(p.src)
			} else {
				p.pos += 4 + end + 2
			}
		case strings.HasPrefix(p.src[p.pos:], "--"):
			end := strings.IndexByte(p.src[p.pos:], '\n')
			if end < 0 {
				p.pos = len(p.src)
			} else {
				p.pos += end + 1
			}
		default:
			return
		}
	}
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentChar(c byte) bool { return isIdentStart(c) || (c >= '0' && c <= '9') }

func (p *parser) ident() (string, error) {
	start := p.pos
	if !isIdentStart(p.peek()) {
		return "", p.errf("se esperaba un nombre")
	}
	for !p.eof() && isIdentChar(p.src[p.pos]) {
		p.pos++
	}
	return p.src[start:p.pos], nil
}

func (p *parser) value() (any, error) {
	p.skip()
	c := p.peek()
	switch {
	case c == '{':
		return p.table()
	case c == '"' || c == '\'':
		return p.str()
	case c == '-' || c == '.' || (c >= '0' && c <= '9'):
		return p.number()
	case isIdentStart(c):
		id, _ := p.ident()
		switch id {
		case "true":
			return true, nil
		case "false":
			return false, nil
		case "nil":
			return nil, nil
		}
		return nil, p.errf("valor inesperado %q", id)
	}
	return nil, p.errf("valor inesperado %q", string(c))
}

func (p *parser) number() (any, error) {
	start := p.pos
	if p.peek() == '-' {
		p.pos++
	}
	if strings.HasPrefix(p.src[p.pos:], "0x") || strings.HasPrefix(p.src[p.pos:], "0X") {
		p.pos += 2
		for !p.eof() && strings.IndexByte("0123456789abcdefABCDEF", p.src[p.pos]) >= 0 {
			p.pos++
		}
		n, err := strconv.ParseInt(strings.Replace(strings.ToLower(p.src[start:p.pos]), "0x", "", 1), 16, 64)
		if err != nil {
			return nil, p.errf("número no válido %q", p.src[start:p.pos])
		}
		return float64(n), nil
	}
	for !p.eof() && strings.IndexByte("0123456789.eE+-", p.src[p.pos]) >= 0 {
		// '+'/'-' solo valen tras el exponente
		if (p.src[p.pos] == '+' || p.src[p.pos] == '-') && p.pos > start && p.src[p.pos-1] != 'e' && p.src[p.pos-1] != 'E' {
			break
		}
		p.pos++
	}
	raw := p.src[start:p.pos]
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil, p.errf("número no válido %q", raw)
	}
	return f, nil
}

func (p *parser) str() (any, error) {
	quote := p.src[p.pos]
	p.pos++
	var b strings.Builder
	for {
		if p.eof() {
			return nil, p.errf("cadena sin cerrar")
		}
		c := p.src[p.pos]
		p.pos++
		if c == quote {
			return b.String(), nil
		}
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		if p.eof() {
			return nil, p.errf("escape incompleto")
		}
		e := p.src[p.pos]
		p.pos++
		switch e {
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 'r':
			b.WriteByte('\r')
		case 'a':
			b.WriteByte('\a')
		case 'b':
			b.WriteByte('\b')
		case 'f':
			b.WriteByte('\f')
		case 'v':
			b.WriteByte('\v')
		case '\\', '"', '\'':
			b.WriteByte(e)
		case '\n':
			b.WriteByte('\n')
		default:
			if e >= '0' && e <= '9' { // \ddd decimal
				n := int(e - '0')
				for i := 0; i < 2 && !p.eof() && p.src[p.pos] >= '0' && p.src[p.pos] <= '9'; i++ {
					n = n*10 + int(p.src[p.pos]-'0')
					p.pos++
				}
				b.WriteByte(byte(n))
			} else {
				b.WriteByte(e)
			}
		}
	}
}

func (p *parser) table() (any, error) {
	p.pos++ // {
	t := &Table{Hash: map[string]any{}}
	for {
		p.skip()
		if p.consume('}') {
			return t, nil
		}
		var key any
		hasKey := false
		if p.peek() == '[' {
			p.pos++
			k, err := p.value()
			if err != nil {
				return nil, err
			}
			p.skip()
			if !p.consume(']') {
				return nil, p.errf("se esperaba ']'")
			}
			p.skip()
			if !p.consume('=') {
				return nil, p.errf("se esperaba '=' tras la clave")
			}
			key, hasKey = k, true
		} else if isIdentStart(p.peek()) {
			save := p.pos
			id, _ := p.ident()
			p.skip()
			if p.consume('=') {
				key, hasKey = id, true
			} else {
				p.pos = save
			}
		}
		v, err := p.value()
		if err != nil {
			return nil, err
		}
		if hasKey {
			switch k := key.(type) {
			case string:
				t.Hash[k] = v
			case float64:
				idx := int(k)
				if float64(idx) == k && idx >= 1 {
					for len(t.Array) < idx {
						t.Array = append(t.Array, nil)
					}
					t.Array[idx-1] = v
				} else {
					t.Hash[strconv.FormatFloat(k, 'f', -1, 64)] = v
				}
			default:
				t.Hash[fmt.Sprint(k)] = v
			}
		} else {
			t.Array = append(t.Array, v)
		}
		p.skip()
		if !p.consume(',') && !p.consume(';') {
			p.skip()
			if !p.consume('}') {
				return nil, p.errf("se esperaba ',' o '}'")
			}
			return t, nil
		}
	}
}
