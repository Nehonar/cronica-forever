# Crónica de Forever

Convierte lo que haces en **WoW: Forever** en la historia de tu personaje.

Un addon registra lo que pasa en el juego (misiones, zonas, niveles, equipo azul o mejor, caídas). Un pequeño programa en tu PC lo lee, agrupa las misiones en **cadenas** y **encargos sueltos**, se los pasa a Claude para que los narre y publica la crónica en GitHub Pages.

```
Addon (juego) ──► Cronica.lua ──► programa «cronica» ──► claude -p ──► docs/data/*.json ──► GitHub Pages
                                                   └──► CronicaTextos.lua (para leerlo dentro del juego)
```

Web: https://nehonar.github.io/cronica-forever/

## Estado

| Pieza | Estado |
|---|---|
| Programa para PC (Windows y Linux) | Prototipo funcionando |
| Web | Funcionando, con un personaje de demostración |
| Addon | Pendiente: de momento se prueba con `samples/Cronica.lua` |

## Requisitos

- **Claude Code** instalado y con la sesión iniciada con tu cuenta (`claude` debe funcionar en la terminal). El programa lo usa en modo no interactivo (`claude -p`), así que las narraciones van con tu suscripción, sin clave de API.
- **Git**, si quieres que publique solo en GitHub.

## Uso

1. Descarga el ejecutable de la última versión (pestaña *Releases*) o compílalo con Go:
   ```
   go build -o cronica ./cmd/cronica
   ```
2. Desde la carpeta del repositorio:
   ```
   cronica iniciar
   ```
   Crea `cronica.json`. Revisa las rutas:
   - `savedvariables`: `World of Warcraft/<carpeta de Forever>/WTF/Account/<CUENTA>/SavedVariables/Cronica.lua`
   - `addon_textos`: `World of Warcraft/<carpeta de Forever>/Interface/AddOns/Cronica/CronicaTextos.lua`
   - `publicar`: `true` para subir a GitHub tras cada pasada.
3. Una pasada:
   ```
   cronica procesar
   ```
   O dejarlo vigilando mientras juegas (procesa cada vez que el juego guarda: al salir o con `/reload`):
   ```
   cronica vigilar
   ```

Para probar sin juego ni addon:
```
cronica procesar -sv samples/Cronica.lua -repo /tmp/prueba          # con Claude de verdad
cronica procesar -sv samples/Cronica.lua -repo /tmp/prueba -prueba  # sin llamar a Claude
```

## Cómo agrupa

- **Cadena**: entregas una misión a un PNJ y ese mismo PNJ te da otra en menos de 90 s. Se narra cuando terminas la cadena.
- **Encargos sueltos**: misiones sin relación. Se juntan de 4 en 4, o antes si cambias de zona.
- **Hito de equipo**: la primera vez que te pones una pieza azul o mejor.
- Las caídas y el equipo nuevo se cuentan una sola vez, en el relato del tramo en el que ocurrieron.

## Personajes

Cada personaje tiene su ficha en `personajes/<Nombre>-<Reino>.json` (trasfondo, voz, hilos abiertos, lema). Si aparece un personaje nuevo, el programa crea su ficha vacía; si ya existe una ficha con trasfondo para el mismo nombre (por ejemplo `Nehonar-Demo.json`), la copia como punto de partida.

## Estructura

```
cmd/cronica/        programa
internal/luasv/     lector de SavedVariables (Lua)
internal/model/     eventos que registra el addon
internal/group/     cadenas, encargos sueltos e hitos
internal/narrate/   prompts y llamada a Claude
internal/store/     JSON de la web y textos para el addon
docs/               web (GitHub Pages)
personajes/         fichas de trasfondo
samples/            datos de ejemplo del addon
```
