package luasv

import (
	"os"
	"testing"
)

func TestParseBasics(t *testing.T) {
	src := `
-- comentario
DB = {
	["nombre"] = "Ne\"ho\\nar",
	["nivel"] = 12,
	["neg"] = -3.5,
	["ok"] = true,
	["nada"] = nil,
	plano = 'x',
	{
		["a"] = 1,
	}, -- [1]
	"dos", -- [2]
	[5] = "cinco",
}
Otra = "hola\nmundo"
`
	vars, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	db := vars["DB"].(*Table)
	if got := db.String("nombre"); got != `Ne"ho\nar` {
		t.Errorf("nombre = %q", got)
	}
	if db.Int("nivel") != 12 || db.Get("neg").(float64) != -3.5 || db.Get("ok") != true {
		t.Errorf("valores escalares incorrectos: %+v", db.Hash)
	}
	if db.String("plano") != "x" {
		t.Errorf("clave sin corchetes")
	}
	if len(db.Array) != 5 || db.Array[1] != "dos" || db.Array[4] != "cinco" {
		t.Errorf("array = %#v", db.Array)
	}
	if db.Array[0].(*Table).Int("a") != 1 {
		t.Errorf("subtabla")
	}
	if vars["Otra"] != "hola\nmundo" {
		t.Errorf("Otra = %q", vars["Otra"])
	}
}

func TestParseSample(t *testing.T) {
	b, err := os.ReadFile("../../samples/Cronica.lua")
	if err != nil {
		t.Skip(err)
	}
	vars, err := Parse(string(b))
	if err != nil {
		t.Fatal(err)
	}
	db := vars["CronicaDB"].(*Table)
	ch := db.Table("characters").Table("Nehonar-Demo")
	if ch == nil || len(ch.Table("events").Array) != 28 {
		t.Fatalf("muestra mal leída")
	}
}

func TestEncodeRoundTrip(t *testing.T) {
	in := map[string]any{"t": "a \"b\"\nc", "n": 3, "l": []any{"x", map[string]any{"k": true}}}
	src := "V = " + Encode(in, 0)
	vars, err := Parse(src)
	if err != nil {
		t.Fatal(err, src)
	}
	v := vars["V"].(*Table)
	if v.String("t") != "a \"b\"\nc" || v.Int("n") != 3 || len(v.Table("l").Array) != 2 {
		t.Fatalf("ida y vuelta: %s", src)
	}
}
