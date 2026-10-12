package narrate

import "strings"

// Trasfondo canónico de Warcraft (época de WoW Classic) por zona, para que el cronista
// sitúe los relatos en la historia real del mundo. Solo lo que la gente de la zona
// sabe: nada de giros que se descubren en misiones posteriores.
type zoneLore struct {
	keys []string // nombres de la zona en el juego (español) y en inglés, en minúsculas
	text string
}

var lore = []zoneLore{
	{[]string{"bosque de elwynn", "elwynn"},
		"Corazón del reino humano de Ventormenta. Tras la Segunda Guerra la capital se reconstruyó, pero el rey Varian Wrynn desapareció y el reino lo gobierna una regencia con el joven Anduin; el ejército está lejos y la Guardia de Ventormenta no da abasto. Kóbolds en las minas, gnolls en los bosques y bandidos de la Hermandad Defias asaltan granjas y caminos. Los Defias nacieron de los canteros que reconstruyeron Ventormenta: la Casa de los Nobles se negó a pagarles, hubo disturbios y acabaron proscritos."},
	{[]string{"páramos de poniente", "paramos de poniente", "westfall"},
		"Antiguo granero de Ventormenta, hoy tierra abandonada: el reino dejó de protegerla y los granjeros, arruinados, se defienden solos con la Milicia del Pueblo, refugiada en la Colina del Centinela. La Hermandad Defias (canteros a los que la nobleza no pagó la reconstrucción de la capital) domina granjas y caminos, y usa ingenios mecánicos; gnolls y múrlocs completan el desorden. Mucha gente desesperada se ha unido a los Defias por hambre."},
	{[]string{"dun morogh"},
		"Valles nevados de Khaz Modan, hogar de los enanos Barbabronce y refugio de los gnomos, que perdieron su capital, Gnomeregan, ante una invasión de troggs y la dejaron contaminada. Trolls Peloescarcha en las cuevas, troggs que salen de la tierra y frío que pone a prueba a cualquiera."},
	{[]string{"forjaz", "ironforge"},
		"Capital de los enanos Barbabronce, tallada dentro de la montaña, alrededor de la Gran Forja; la gobierna el rey Magni Barbabronce. Ciudad de herreros, mineros y de la Liga de Expedicionarios, que busca los orígenes de los enanos bajo la tierra. Bajo la ciudad yacen las tumbas de los antiguos reyes enanos."},
	{[]string{"salón de los feudales", "salon de los feudales", "hall of thanes"},
		"Las tumbas de los antiguos reyes y señores enanos bajo Forjaz: un lugar sagrado para los Barbabronce, ahora inquieto, con espíritus que no encuentran descanso."},
	{[]string{"loch modan"},
		"Tierras enanas en torno a un gran lago contenido por una presa de piedra; el pueblo de Thelsamar resiste a los troggs, a los saboteadores enanos Hierro Negro y a las bestias de los montes."},
	{[]string{"montañas crestagrana", "montanas crestagrana", "redridge"},
		"Montañas al este de Elwynn, con el pueblo de Villa del Lago junto al lago Everstill. Orcos de la tribu Roca Negra bajan de las cumbres y gnolls asolan los caminos; la guerra con los orcos sigue viva aquí."},
	{[]string{"bosque del ocaso", "duskwood"},
		"Bosque sumido en una penumbra eterna desde hace años; en Villa Oscura la Guardia Nocturna resiste a muertos vivientes, lobos y criaturas que salen del cementerio y de los bosques. Una tierra de miedo, leyendas y maldiciones."},
	{[]string{"ventormenta", "stormwind"},
		"Capital del reino humano, reconstruida tras la Segunda Guerra. El rey Varian Wrynn está desaparecido; gobierna una regencia en nombre del joven Anduin, con el alto alguacil Bolvar Fordragón al frente. Catedral de la Luz, Casa de los Nobles y barrios llenos de soldados, mercaderes y refugiados."},
	{[]string{"teldrassil"},
		"El nuevo árbol del mundo de los elfos de la noche, plantado por los druidas tras la Tercera Guerra; sobre sus ramas está Darnassus. Algo corrompe el bosque: los furbolgs y las bestias se vuelven agresivos, y los centinelas de la sacerdotisa Tyrande Susurravientos vigilan."},
	{[]string{"costa oscura", "darkshore"},
		"Costa brumosa al sur de Teldrassil, con el puerto de Auberdine y ruinas élficas antiquísimas. Los furbolgs han enloquecido por una corrupción que se extiende por el bosque."},
	{[]string{"durotar"},
		"Tierra roja y árida donde Thrall asentó a la nueva Horda tras escapar de los campos de internamiento; su capital es Orgrimmar. Los trolls Lanza Negra son aliados, aunque un grupo de ellos se ha vuelto contra la Horda en las Islas del Eco."},
	{[]string{"mulgore"},
		"Praderas de los tauren, unidos bajo el jefe Cairne Pezuña de Sangre en la Cima del Trueno. Pueblo pacífico y espiritual, ligado a la Madre Tierra; la Compañía Ventura saquea sus tierras y los gnolls los acosan."},
	{[]string{"claros de tirisfal", "tirisfal"},
		"Tierras del antiguo reino de Lordaeron, arrasado por la Plaga. Los Renegados, muertos vivientes libres de la voluntad del Rey Exánime, siguen a la Dama Oscura Sylvanas Brisaveloz desde Entrañas, bajo las ruinas de la capital. La Cruzada Escarlata los persigue con fanatismo."},
	{[]string{"los baldíos", "los baldios", "barrens"},
		"Una sabana inmensa y seca en el corazón de Kalimdor. El Cruce es el puesto avanzado de la Horda; centauros, jabaespines y la Compañía Ventura se disputan la tierra, y en la costa está Trinquete, puerto neutral de los goblins."},
}

// LoreFor devuelve el trasfondo de la zona (o de alguna de sus subzonas), si se conoce.
func LoreFor(zone string, subzones []string) string {
	names := append([]string{zone}, subzones...)
	for _, n := range names {
		n = strings.ToLower(strings.TrimSpace(n))
		if n == "" {
			continue
		}
		for _, l := range lore {
			for _, k := range l.keys {
				if n == k || strings.Contains(n, k) {
					return l.text
				}
			}
		}
	}
	return ""
}
