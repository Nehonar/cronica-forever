
CronicaDB = {
	["version"] = 1,
	["characters"] = {
		["Tobias-Demo"] = {
			["name"] = "Tobias",
			["realm"] = "Demo",
			["race"] = "Humano",
			["class"] = "Guerrero",
			["classFile"] = "WARRIOR",
			["level"] = 6,
			["played"] = 11040,
			["events"] = {
				{
					["t"] = 1791100000,
					["type"] = "login",
					["level"] = 1,
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Abadía de Villanorte",
				}, -- [1]
				{
					["t"] = 1791100060,
					["type"] = "quest_accept",
					["id"] = 783,
					["title"] = "Una amenaza interna",
					["npc"] = "Ayudante Willem",
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Valle de Villanorte",
					["level"] = 1,
					["text"] = "Así que vienes a la abadía a formarte, ¿eh? Pues llegas en mal momento. Algo se mueve en el valle y el alguacil McBride necesita a todo el que sepa sostener un arma.\n\nVe a verle dentro de la abadía. Él te dirá qué hacer.",
					["objectives"] = "Habla con el alguacil McBride.",
				}, -- [2]
				{
					["t"] = 1791100180,
					["type"] = "quest_turnin",
					["id"] = 783,
					["title"] = "Una amenaza interna",
					["npc"] = "Alguacil McBride",
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Abadía de Villanorte",
					["level"] = 1,
					["reward"] = "Bien. Necesito manos, y las tuyas parecen acostumbradas al trabajo duro. Se nota que tienes ganas, aunque esa armadura te quede grande.",
				}, -- [3]
				{
					["t"] = 1791100200,
					["type"] = "quest_accept",
					["id"] = 7,
					["title"] = "Limpieza del campamento kóbold",
					["npc"] = "Alguacil McBride",
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Abadía de Villanorte",
					["level"] = 1,
					["text"] = "Los kóbolds han salido de las minas y han montado un campamento al norte de la abadía, junto a los viñedos. Si los dejamos crecer, tendremos un problema que no podremos contener.\n\nVe allí y reduce su número. No te pido que acabes con todos: te pido que entiendan que este valle no es suyo.",
					["objectives"] = "Mata 10 kóbolds alimaña.",
				}, -- [4]
				{
					["t"] = 1791100260,
					["type"] = "quest_accept",
					["id"] = 33,
					["title"] = "Lobos en la frontera",
					["npc"] = "Eagan Peltskinner",
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Valle de Villanorte",
					["level"] = 1,
					["text"] = "Los lobos han bajado de las colinas antes de lo normal este año. Atacan al ganado y la abadía necesita carne para el invierno.\n\nTráeme carne de lobo de las bestias del valle y me encargaré de que no se desperdicie nada.",
					["objectives"] = "Consigue 8 trozos de carne de lobo duro.",
				}, -- [5]
				{
					["t"] = 1791101500,
					["type"] = "level",
					["level"] = 2,
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Valle de Villanorte",
				}, -- [6]
				{
					["t"] = 1791101700,
					["type"] = "quest_turnin",
					["id"] = 7,
					["title"] = "Limpieza del campamento kóbold",
					["npc"] = "Alguacil McBride",
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Abadía de Villanorte",
					["level"] = 2,
					["reward"] = "Buen trabajo. Pero lo que me cuentas me preocupa: si ese campamento era tan grande, puede que haya más ahí dentro.",
				}, -- [7]
				{
					["t"] = 1791101730,
					["type"] = "quest_accept",
					["id"] = 15,
					["title"] = "Investigar la Cresta del Eco",
					["npc"] = "Alguacil McBride",
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Abadía de Villanorte",
					["level"] = 2,
					["text"] = "Los kóbolds vienen de la Cresta del Eco, una mina abandonada al norte del valle. Necesito saber cuántos hay ahí dentro antes de mandar a nadie más.\n\nEntra, mira y vuelve. Y si tienes que pelear, pelea.",
					["objectives"] = "Mata 10 kóbolds excavadores.",
				}, -- [8]
				{
					["t"] = 1791101800,
					["type"] = "quest_turnin",
					["id"] = 33,
					["title"] = "Lobos en la frontera",
					["npc"] = "Eagan Peltskinner",
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Valle de Villanorte",
					["level"] = 2,
					["reward"] = "Buena carne, y bien cortada. Algo me dice que no es la primera vez que cargas sacos.",
				}, -- [9]
				{
					["t"] = 1791102900,
					["type"] = "level",
					["level"] = 3,
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Cresta del Eco",
				}, -- [10]
				{
					["t"] = 1791103000,
					["type"] = "quest_turnin",
					["id"] = 15,
					["title"] = "Investigar la Cresta del Eco",
					["npc"] = "Alguacil McBride",
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Abadía de Villanorte",
					["level"] = 3,
					["reward"] = "¿Tantos? Me lo temía. Esto ya no es una plaga, es una invasión.",
				}, -- [11]
				{
					["t"] = 1791103040,
					["type"] = "quest_accept",
					["id"] = 21,
					["title"] = "Escaramuza en la Cresta del Eco",
					["npc"] = "Alguacil McBride",
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Abadía de Villanorte",
					["level"] = 3,
					["text"] = "Ya no podemos esperar. Los kóbolds de la Cresta del Eco están demasiado cerca de la abadía. Vuelve a la mina y ataca su corazón: los obreros que cavan y los que los vigilan.\n\nEsta vez no vas a mirar. Vas a luchar.",
					["objectives"] = "Mata 12 kóbolds obreros.",
				}, -- [12]
				{
					["t"] = 1791103100,
					["type"] = "quest_accept",
					["id"] = 18,
					["title"] = "Hermandad de ladrones",
					["npc"] = "Ayudante Willem",
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Valle de Villanorte",
					["level"] = 3,
					["text"] = "Unos bandidos se han instalado en los viñedos del este. Llevan pañuelos rojos y se hacen llamar Defias. Roban a los viñadores y se ríen de la guardia.\n\nTráeme sus pañuelos. Quiero que sepan que alguien los está contando.",
					["objectives"] = "Consigue 12 pañuelos rojos.",
				}, -- [13]
				{
					["t"] = 1791104200,
					["type"] = "level",
					["level"] = 4,
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Viñedos de Villanorte",
				}, -- [14]
				{
					["t"] = 1791104600,
					["type"] = "quest_turnin",
					["id"] = 18,
					["title"] = "Hermandad de ladrones",
					["npc"] = "Ayudante Willem",
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Valle de Villanorte",
					["level"] = 4,
					["reward"] = "Doce pañuelos. Doce bandidos menos robando a gente honrada. No está mal para un novicio.",
				}, -- [15]
				{
					["t"] = 1791105200,
					["type"] = "death",
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Cresta del Eco",
					["level"] = 4,
				}, -- [16]
				{
					["t"] = 1791105800,
					["type"] = "quest_turnin",
					["id"] = 21,
					["title"] = "Escaramuza en la Cresta del Eco",
					["npc"] = "Alguacil McBride",
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Abadía de Villanorte",
					["level"] = 4,
					["reward"] = "Has vuelto, y con el escudo abollado. Eso me dice que hiciste lo que había que hacer.",
				}, -- [17]
				{
					["t"] = 1791105830,
					["type"] = "quest_accept",
					["id"] = 54,
					["title"] = "Informe a Villadorada",
					["npc"] = "Alguacil McBride",
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Abadía de Villanorte",
					["level"] = 4,
					["text"] = "Has hecho más por este valle en unos días que muchos en meses. Pero aquí ya no te necesito tanto como te van a necesitar fuera.\n\nLleva este informe al alguacil Dughan, en Villadorada. Cuéntale lo de los kóbolds y lo de los bandidos de pañuelo rojo.",
					["objectives"] = "Lleva el informe al alguacil Dughan, en Villadorada.",
				}, -- [18]
				{
					["t"] = 1791106000,
					["type"] = "equip",
					["item"] = "Espada corta del vigía",
					["quality"] = 3,
					["slot"] = "MAINHAND",
					["itemType"] = "Arma",
					["itemSubType"] = "Espadas de una mano",
					["stats"] = {
						["ITEM_MOD_STRENGTH_SHORT"] = 3,
						["ITEM_MOD_STAMINA_SHORT"] = 2,
					},
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Valle de Villanorte",
					["level"] = 4,
				}, -- [19]
				{
					["t"] = 1791106100,
					["type"] = "level",
					["level"] = 5,
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Valle de Villanorte",
				}, -- [20]
				{
					["t"] = 1791106600,
					["type"] = "zone",
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Villadorada",
					["level"] = 5,
				}, -- [21]
				{
					["t"] = 1791106700,
					["type"] = "quest_turnin",
					["id"] = 54,
					["title"] = "Informe a Villadorada",
					["npc"] = "Alguacil Dughan",
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Villadorada",
					["level"] = 5,
					["reward"] = "McBride habla bien de ti, y no habla bien de casi nadie. Descansa en la posada. Mañana hablaremos de lo que pasa en Elwynn.",
				}, -- [22]
				{
					["t"] = 1791106900,
					["type"] = "quest_accept",
					["id"] = 60,
					["title"] = "Velas de kóbold",
					["npc"] = "William Pestle",
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Villadorada",
					["level"] = 5,
					["text"] = "¿Has visto las velas que llevan los kóbolds en la cabeza? Esa cera es estupenda para mis preparados, y no voy a ir yo a quitársela.\n\nSi me traes unas cuantas, te pagaré bien.",
					["objectives"] = "Consigue 8 velas de kóbold.",
				}, -- [23]
				{
					["t"] = 1791107600,
					["type"] = "quest_accept",
					["id"] = 88,
					["title"] = "¡La princesa debe morir!",
					["npc"] = "Ma Stonefield",
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Granja de los Stonefield",
					["level"] = 5,
					["text"] = "Esa cerda de los Maclure, Princesa la llaman, se pasa el día destrozando mis huertos. Los Maclure no hacen nada y yo ya no aguanto más.\n\nAcaba con ella y tráeme su collar como prueba.",
					["objectives"] = "Trae el collar de Princesa.",
				}, -- [24]
				{
					["t"] = 1791109000,
					["type"] = "level",
					["level"] = 6,
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Mina Fargodeep",
				}, -- [25]
				{
					["t"] = 1791109500,
					["type"] = "quest_turnin",
					["id"] = 60,
					["title"] = "Velas de kóbold",
					["npc"] = "William Pestle",
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Villadorada",
					["level"] = 6,
					["reward"] = "¡Perfectas! Y todavía huelen a mina. Toma, te lo has ganado.",
				}, -- [26]
				{
					["t"] = 1791110800,
					["type"] = "quest_turnin",
					["id"] = 88,
					["title"] = "¡La princesa debe morir!",
					["npc"] = "Ma Stonefield",
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Granja de los Stonefield",
					["level"] = 6,
					["reward"] = "Por fin. No me mires así: a veces la paz entre vecinos cuesta un disgusto.",
				}, -- [27]
				{
					["t"] = 1791111040,
					["type"] = "logout",
					["zone"] = "Bosque de Elwynn",
					["subzone"] = "Villadorada",
					["level"] = 6,
				}, -- [28]
			},
		},
	},
}
