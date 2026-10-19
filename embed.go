// Package cronicaforever incluye dentro del programa la web y los datos de
// demostración, para que «cronica demo» funcione sin clonar el repositorio.
package cronicaforever

import "embed"

// Files contiene la web, el addon, la muestra del addon y la ficha de demostración.
//
//go:embed docs/index.html docs/app.js docs/estilo.css docs/oficios/recetas.json docs/banco samples/Cronica.lua personajes/Tobias-Demo.json addon/Cronica/Cronica.toc addon/Cronica/Cronica.lua addon/Cronica/Oficios.lua addon/Cronica/Bindings.xml addon/Cronica/CronicaTextos.lua
var Files embed.FS

// Recetas es la tabla de recetas de Herrería y Minería (docs/oficios/recetas.json).
func Recetas() []byte {
	b, _ := Files.ReadFile("docs/oficios/recetas.json")
	return b
}
