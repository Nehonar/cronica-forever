package oficios

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Nehonar/cronica-forever/internal/luasv"
)

const sv = `CronicaDB = {
	["nombres"] = { [2770] = "Mena de cobre", [2840] = "Barra de cobre" },
	["calidad"] = { [2770] = 1 },
	["characters"] = { ["Nehonar-Reino"] = { ["oficios"] = {
		["bolsas"] = { [2770] = 20, [2840] = 5 },
		["banco"] = { [2770] = 3 },
		["bancoVisto"] = true,
		["dinero"] = 12345,
		["profesiones"] = { ["Herrería"] = { ["nivel"] = 87, ["max"] = 150 } },
		["recetas"] = { ["Herrería"] = { ["nivel"] = 87, ["max"] = 150, ["lista"] = {
			{ ["nombre"] = "Hacha de cobre", ["color"] = "amarillo", ["id"] = 3491, ["mats"] = { { ["id"] = 2840, ["n"] = 8, ["nombre"] = "Barra de cobre" } } },
		} } },
		["instructor"] = { ["Herrería"] = { { ["nombre"] = "Cinturón de cobre", ["nivel"] = 35, ["precio"] = 500, ["id"] = 2851, ["estado"] = "available" } } },
	} } },
}`

func TestParseAndWrite(t *testing.T) {
	vars, err := luasv.Parse(sv)
	if err != nil {
		t.Fatal(err)
	}
	db := vars["CronicaDB"].(*luasv.Table)
	d, ok := Parse(db.Table("characters").Table("Nehonar-Reino"))
	if !ok || len(d.Bags) != 2 || d.Bags[0] != (Stack{2770, 20}) || d.Bank[0].N != 3 || !d.BankSeen || d.Money != 12345 {
		t.Fatalf("%+v", d)
	}
	if d.Skills["Herrería"].Level != 87 || d.Known["Herrería"][0].Mats[0].N != 8 || d.Trainer["Herrería"][0].Price != 500 {
		t.Fatalf("%+v", d)
	}
	names, q := Names(db)
	if names["2770"] != "Mena de cobre" || q["2770"] != 1 {
		t.Fatalf("%v %v", names, q)
	}
	dir := t.TempDir()
	d.Key, d.Name = "Nehonar-Reino", "Nehonar"
	if err := Write(dir, []Data{d}, names, q); err != nil {
		t.Fatal(err)
	}
	var idx Index
	b, _ := os.ReadFile(filepath.Join(dir, "oficios", "datos", "indice.json"))
	if json.Unmarshal(b, &idx) != nil || len(idx.Characters) != 1 || idx.Names["2840"] != "Barra de cobre" {
		t.Fatalf("%s", b)
	}
	if ids := RecipeIDs([]byte(`{"objetos":{"2840":{},"12":{}}}`)); len(ids) != 2 || ids[0] != 12 {
		t.Fatal(ids)
	}
}
