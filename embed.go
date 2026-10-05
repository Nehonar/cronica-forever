// Package cronicaforever incluye dentro del programa la web y los datos de
// demostración, para que «cronica demo» funcione sin clonar el repositorio.
package cronicaforever

import "embed"

// Files contiene la web, la muestra del addon y la ficha de demostración.
//
//go:embed docs/index.html docs/app.js docs/estilo.css samples/Cronica.lua personajes/Nehonar-Demo.json
var Files embed.FS
