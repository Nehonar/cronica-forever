# Crónica de Forever

Convierte lo que haces en **WoW: Forever** en la historia de tu personaje.

Un addon registra lo que pasa en el juego (misiones, zonas, niveles, mazmorras, equipo azul o mejor, reputación, caídas). Un pequeño programa en tu PC lo lee y, cuando el personaje ha vivido bastantes **momentos clave**, le pide a Claude una **historia corta** (un minuto narrado), la dobla con una voz de España (opcional, gratis con Azure) y publica la crónica en GitHub Pages con un botón para escucharla.

```
Addon (juego) ──► Cronica.lua ──► programa «cronica» ──► claude -p ──► docs/data/*.json ──► GitHub Pages
                                                   ├──► Azure (voz) ──► docs/audio/<personaje>/historia-N.mp3
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

## Nombre y apellido

En Forever varios personajes pueden llamarse igual con distinto apellido, así que el addon identifica a cada uno por nombre, apellido y reino. Las crónicas de versiones anteriores (que solo usaban el nombre) pasan solas al personaje con apellido que encaja (misma raza y clase; si hay varios, el de más nivel); los relatos que mezclaban dos personajes se quitan. En *Personajes → Sus relatos…* puedes quitar cualquier relato suelto.

## Borrar la crónica de un personaje

En *Personajes*, elige el personaje → **Borrar su crónica…**:
- **Solo los relatos**: se quedan su historia y sus frases; lo jugado hasta ahora no se vuelve a contar y el cronista sigue desde ese momento.
- **El personaje entero**: relatos, misiones, historia y frases. Si vuelves a jugarlo, aparece como personaje nuevo.

Para borrarlo **todo** (todos los personajes, por ejemplo cuando salga el juego oficial): *Configuración → Empezar de cero*.

El borrado se publica en GitHub como cualquier otro cambio (lo anterior queda en el historial del repositorio).

## Cuándo se escribe una historia

Desde la 0.6, las misiones ya no se narran una a una (en el juego ya las lee Spoken con su texto completo). El cronista escribe **historias cortas** cuando el personaje reúne **8 puntos** de momentos clave, o en cuanto pasa algo gordo si ya lleva **4**:

| Momento | Puntos |
|---|---|
| Subir de nivel | 1 (nivel 10, 20, 30…: 3, momento gordo) |
| Terminar una mazmorra (algún jefe derrotado) | 4, momento gordo |
| Terminar una cadena de 3 misiones o más | 2 |
| Ponerse por primera vez una pieza azul | 1 (morada o mejor: 3, momento gordo) |
| Subir de rango con una facción (Amistoso o más) | 2 |
| Pisar una zona por primera vez | 1 |
| Cada 5 misiones entregadas | 1 |

Las misiones de la historia solo dan el contexto (dónde estaba y a quién ayudaba). Si sales del juego con pocos puntos, no se escribe nada: se sigue sumando la próxima vez. El addon lleva la misma cuenta y te avisa en el juego cuando hay una historia lista (`/cronica` enseña cuántos puntos llevas). Los capítulos por zona de la 0.5 se quedan como estaban.

**Cadena**: entregas una misión a un PNJ y ese mismo PNJ te da otra en menos de 90 s.

## Voz del narrador (opcional)

En *Configuración → Voz del narrador* pegas la clave y la región de un recurso **Servicios de Voz** de Azure con el plan gratuito **F0** (500.000 caracteres al mes; si se acaban, deja de doblar hasta el mes siguiente, nunca cobra). Eliges la voz (por defecto Álvaro, un 8 % más lento y un 4 % más grave), la pruebas y, desde ese momento, cada historia nueva se dobla sola y la web enseña un botón para escucharla. La clave se guarda solo en `cronica.json`, en tu PC.

## Personajes

Cada personaje tiene su ficha en `personajes/<Nombre>-<Reino>.json` (trasfondo, voz, hilos abiertos, lema). Si aparece un personaje nuevo, el programa crea su ficha vacía; si ya existe una ficha con trasfondo para el mismo nombre (por ejemplo `Tobias-Demo.json`, el personaje de prueba), la copia como punto de partida.

## Estructura

```
cmd/cronica/        programa
internal/luasv/     lector de SavedVariables (Lua)
internal/model/     eventos que registra el addon
internal/group/     historias por puntos (y los capítulos y relatos de antes)
internal/narrate/   prompts y llamada a Claude
internal/voz/       doblaje con Azure
internal/store/     JSON de la web y textos para el addon
docs/               web (GitHub Pages)
personajes/         fichas de trasfondo
samples/            datos de ejemplo del addon
```

## Banco y oficios (app para el móvil)

El addon también guarda lo que llevas en las bolsas y en el banco (el banco, cada vez que lo abres), tus profesiones, las recetas que conoces (al abrir la ventana de la profesión, con su color) y lo que enseña el instructor (al abrirlo). Al salir del juego, el programa lo publica en `docs/oficios/datos/` y la app **Banco y oficios** (`docs/banco/`) lo enseña:

- **Inventario**: bolsas y banco en cuadrícula, con los colores de calidad, buscador y en qué recetas se usa cada cosa.
- **Oficios**: cada profesión se despliega con sus recetas (las que sabes, las de tu nivel o todas), su color, sus materiales y cuántos tienes. Con «+» las añades a **tu lista para farmear**, que suma los materiales y te dice cuántos tienes y cuántos te faltan (y, si faltan barras, qué hay que fundir). La lista se guarda en el móvil.

La tabla de recetas (`docs/oficios/recetas.json`) sale del paquete [wow-classic-items](https://github.com/Nakroma/wow-classic-items) (MIT) con `tools/recetas.py`. Los umbrales de color de cada receta se estiman y se corrigen con el color que ves en el juego. Los nombres en español los pide el addon al juego.

**En el móvil**: abre `https://<usuario>.github.io/cronica-forever/banco/` y «Añadir a pantalla de inicio», o instala el APK de Android (`android/`, se compila en GitHub Actions y se publica en la versión «app-banco» del repositorio).
