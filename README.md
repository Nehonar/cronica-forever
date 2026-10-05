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

## Instalación en Windows (para jugar)

No hace falta descargar este repositorio: basta con **`cronica.exe`**.

1. Descarga `cronica-windows-amd64.exe` (de la última ejecución en *Actions*, artefacto «cronica») y ponlo donde quieras, por ejemplo en `Documentos\Cronica`.
2. **Haz doble clic.** Crónica se queda en la bandeja del sistema (junto al reloj; puede estar dentro de la flechita ^) y la primera vez abre en el navegador la página **Preparar Crónica**, con cuatro pasos:
   - **El juego**: propone las instalaciones que encuentra o te deja **elegir la carpeta** con el selector de Windows, e instala allí el addon;
   - **Claude**: si falta, instala Claude Code con el instalador oficial (sin ventanas de consola) y abre el navegador para que inicies sesión;
   - **GitHub** (opcional, ver abajo);
   - **Arranque**: que Crónica se abra sola al encender el PC.
3. Puedes volver a esa página cuando quieras desde el icono: **Configuración…**. Si vuelves a hacer doble clic con Crónica ya abierta, no se abre otra: se abre la página en el navegador.

Windows Defender puede desconfiar de un programa nuevo sin firma. Si lo pone en cuarentena: *Seguridad de Windows → Protección contra virus y amenazas → Historial de protección → Crónica → Acciones → Restaurar/Permitir*.

Si algo impide arrancar, sale una ventana con el error. El registro está en `%LOCALAPPDATA%\cronica\cronica.log`.

Tus datos (configuración, fichas de personajes y crónica) se guardan en `%APPDATA%\Cronica`.

### Publicar en GitHub (opcional)

Si quieres que tu crónica sea también una web pública:
- Necesitas **Git para Windows** (https://git-scm.com/download/win) y un repositorio tuyo en GitHub.
- En **Preparar Crónica** pega la dirección de tu repositorio: Crónica lo descarga en `%APPDATA%\Cronica\web` y prueba a subir; la primera vez, **el gestor de credenciales de Git abre una ventana de GitHub** para que entres. Crónica no ve ni guarda tu contraseña.
- Activa GitHub Pages en el repositorio: *Settings → Pages → rama principal, carpeta /docs*.

### En el juego

- **Cuándo llega a Crónica**: WoW solo deja que un addon guarde sus datos al cerrar sesión o al recargar (`/reload`). Juega normal: al salir, Crónica lo recibe todo y el cronista escribe. Si quieres verlo antes, haz `/reload` en una pausa.
- **Enviar a Crónica con una tecla**: en *Opciones → Atajos → Crónica*, «Enviar a Crónica» recarga la interfaz (lo mismo que `/reload`). Un addon no puede recargar por su cuenta: Blizzard lo reserva para sus propias teclas y macros.
- **Frases del personaje**: `/cronica frases no | local | voz`. En *local* solo las ves tú; en *voz* aparecen en pantalla y, al pulsar tu tecla (o hacer clic en ellas), tu personaje las dice en /decir. Blizzard no deja que un addon hable por su cuenta, por eso hace falta la tecla.
- `/cronica espera <minutos>`: como mucho una frase cada tantos minutos (4 por defecto). `/cronica`: estado y comandos.

## Probarlo en 3 pasos (Linux)

```bash
# 1. Claude Code tiene que responder en la terminal
echo "di hola" | claude -p

# 2. Instalar el programa (necesita Go 1.22 o superior)
go install github.com/Nehonar/cronica-forever/cmd/cronica@latest
export PATH="$PATH:$HOME/go/bin"     # añádelo a ~/.bashrc para que se quede

# 3. Demostración: narra una sesión de ejemplo y abre la web en http://localhost:8000
cronica demo
```

`cronica demo` crea la carpeta `cronica-demo` donde lo ejecutes. Ctrl+C cierra la web. Para verla otra vez sin narrar de nuevo: `cronica ver -repo cronica-demo`.

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

### La app de la bandeja

```
cronica bandeja
```

Pone un icono en la bandeja del sistema y hace todo a la vez: vigila el archivo del addon, narra, publica y sirve en local la web y **el cronista**, un chat con Claude para crear la historia de cada personaje nuevo (cuestionario corto, pregunta a pregunta, que termina en una ficha que puedes guardar o pedir que cambie).

Mientras un personaje no tiene historia, su progreso se registra pero no se narra; el icono se marca y el menú muestra «✦ Nombre — crear su historia». Al guardar la ficha, se narra todo lo que esperaba.

En GNOME, para ver iconos de bandeja hace falta la extensión «AppIndicator and KStatusNotifierItem Support» (Ubuntu ya la trae). En KDE, XFCE, Cinnamon y Windows funciona tal cual.

### Que arranque solo

```
cronica preparar      # comprueba Claude Code; si falta, ofrece instalarlo (instalador oficial) e iniciar sesión
cronica instalar      # una sola vez: abre la bandeja al iniciar sesión en el PC (Linux: autoarranque del escritorio; Windows: Inicio del usuario, sin permisos de administrador)
cronica estado        # comprueba Claude, la configuración y muestra el registro
cronica desinstalar   # lo quita
```

En reposo solo mira la fecha del archivo del addon cada pocos segundos. Al arrancar comprueba que Claude Code tiene la sesión iniciada; si no, muestra un aviso en el escritorio (como mucho uno por hora) y deja el aviso en `CronicaTextos.lua` para que el addon lo enseñe en el juego. Lo que no se pudo narrar se reintenta solo en cuanto vuelve a haber sesión.

Si al arrancar Claude Code no está instalado o no tiene sesión, aparece una ventana preguntando si quieres arreglarlo y, si aceptas, se abre **Preparar Crónica**. Desde ahí se ejecuta, sin ventanas de consola, el instalador oficial de Anthropic (`curl -fsSL https://claude.ai/install.sh | bash` en Linux, `irm https://claude.ai/install.ps1 | iex` en Windows) y `claude auth login`, que abre el navegador en la página de Anthropic; si la página te da un código, se pega en Preparar Crónica. Crónica nunca ve ni guarda tus credenciales. En Linux las ventanas usan `zenity` o `kdialog` si están instalados. (`cronica iniciar` y `cronica preparar` siguen existiendo para quien prefiera la terminal.)

Para probar sin juego ni addon:
```
cronica procesar -sv samples/Cronica.lua -repo /tmp/prueba          # con Claude de verdad
cronica procesar -sv samples/Cronica.lua -repo /tmp/prueba -prueba  # sin llamar a Claude
```

## Borrar la crónica de un personaje

En *Personajes*, elige el personaje → **Borrar su crónica…**:
- **Solo los relatos**: se quedan su historia y sus frases; lo jugado hasta ahora no se vuelve a contar y el cronista sigue desde ese momento.
- **El personaje entero**: relatos, misiones, historia y frases. Si vuelves a jugarlo, aparece como personaje nuevo.

El borrado se publica en GitHub como cualquier otro cambio (lo anterior queda en el historial del repositorio).

## Cómo agrupa

- **Cadena**: entregas una misión a un PNJ y ese mismo PNJ te da otra en menos de 90 s. Se narra cuando terminas la cadena, si tiene 3 misiones o más; las de 2 (por ejemplo «ve a hablar con X» y su encargo) se cuentan con las sueltas.
- **Encargos sueltos**: misiones sin relación. Se juntan de 4 en 4, o antes si cambias de zona.
- **Hito de equipo**: la primera vez que te pones una pieza azul o mejor.
- Las caídas y el equipo nuevo se cuentan una sola vez, en el relato del tramo en el que ocurrieron.

## Personajes

Cada personaje tiene su ficha en `personajes/<Nombre>-<Reino>.json` (trasfondo, voz, hilos abiertos, lema). Si aparece un personaje nuevo, el programa crea su ficha vacía; si ya existe una ficha con trasfondo para el mismo nombre (por ejemplo `Tobias-Demo.json`, el personaje de prueba), la copia como punto de partida.

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
