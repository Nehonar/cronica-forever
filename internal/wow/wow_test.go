package wow

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestFindInstallAndLatest(t *testing.T) {
	root := filepath.Join(t.TempDir(), "World of Warcraft")
	for _, d := range []string{"_classic_era_/Interface", "_forever_/WTF/Account/CUENTA1/SavedVariables", "_forever_/WTF/Account/CUENTA2/SavedVariables", "_retail_/Interface"} {
		os.MkdirAll(filepath.Join(root, d), 0o755)
	}
	found := Find(root)
	if len(found) != 3 || found[0].Name != "_forever_" {
		t.Fatalf("Forever debe ir primero: %+v", found)
	}
	if acc := Accounts(found[0]); len(acc) != 2 {
		t.Fatalf("cuentas: %v", acc)
	}
	files := fstest.MapFS{
		"addon/Cronica/Cronica.toc":       {Data: []byte("## Title: Crónica")},
		"addon/Cronica/CronicaTextos.lua": {Data: []byte("-- vacío")},
	}
	dir, err := InstallAddon(found[0], files)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "CronicaTextos.lua"), []byte("-- con relatos"), 0o644)
	InstallAddon(found[0], files) // reinstalar no debe borrar los relatos
	if b, _ := os.ReadFile(filepath.Join(dir, "CronicaTextos.lua")); string(b) != "-- con relatos" {
		t.Fatal("reinstalar ha pisado los relatos")
	}
	if LatestSavedVariables(found[0].Path) != "" {
		t.Fatal("aún no hay datos")
	}
	p := SavedVariablesPath(found[0], "CUENTA2")
	os.WriteFile(p, []byte("CronicaDB = {}"), 0o644)
	if LatestSavedVariables(found[0].Path) != p {
		t.Fatal("debe encontrar el Cronica.lua de la cuenta usada")
	}
}
