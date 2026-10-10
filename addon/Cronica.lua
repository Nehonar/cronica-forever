
CronicaDB = {
	["characters"] = {
		["TobiasBrenn-ForeverBeta"] = {
			["class"] = "Guerrero",
			["classFile"] = "WARRIOR",
			["events"] = {
				{
					["level"] = 7,
					["subzone"] = "Villadorada",
					["t"] = 1791643520,
					["type"] = "login",
					["zone"] = "Bosque de Elwynn",
				}, -- [1]
				{
					["id"] = 106,
					["level"] = 7,
					["npc"] = "Maybell Maclure",
					["objectives"] = "Lleva la carta de Maybell a Tommy Joe Stonefield.",
					["subzone"] = "Villadorada",
					["t"] = 1791643530,
					["text"] = "Mi familia y los Stonefield no se hablan, pero yo quiero a Tommy Joe Stonefield. ¿Le llevarías esta carta? Está junto al río, al sur de la granja de su familia.",
					["title"] = "La joven enamorada",
					["type"] = "quest_accept",
					["zone"] = "Bosque de Elwynn",
				}, -- [2]
				{
					["id"] = 106,
					["level"] = 7,
					["npc"] = "Tommy Joe Stonefield",
					["reward"] = "¿Una carta de Maybell? ¡Cielos! Gracias, de verdad.",
					["subzone"] = "Villadorada",
					["t"] = 1791643830,
					["title"] = "La joven enamorada",
					["type"] = "quest_turnin",
					["zone"] = "Bosque de Elwynn",
				}, -- [3]
				{
					["id"] = 111,
					["level"] = 7,
					["npc"] = "Tommy Joe Stonefield",
					["objectives"] = "Lleva el colgante de Tommy Joe a Maybell Maclure.",
					["subzone"] = "Villadorada",
					["t"] = 1791643850,
					["text"] = "Tengo que darle una respuesta a Maybell, pero si su familia me ve, se acabó. Llévale mi colgante, ella sabrá lo que significa.",
					["title"] = "Hablar con Maybell",
					["type"] = "quest_accept",
					["zone"] = "Bosque de Elwynn",
				}, -- [4]
				{
					["level"] = 8,
					["played"] = 4998,
					["subzone"] = "Villadorada",
					["t"] = 1791643850,
					["type"] = "level",
					["zone"] = "Bosque de Elwynn",
				}, -- [5]
				{
					["armor"] = 0,
					["item"] = "Espada del viñador",
					["itemSubType"] = "Espadas de una mano",
					["itemType"] = "Arma",
					["level"] = 8,
					["quality"] = 3,
					["slot"] = "MAINHAND",
					["stats"] = {
						["ITEM_MOD_STAMINA_SHORT"] = 3,
						["ITEM_MOD_STRENGTH_SHORT"] = 4,
					},
					["subzone"] = "Villadorada",
					["t"] = 1791644252,
					["type"] = "equip",
					["zone"] = "Bosque de Elwynn",
				}, -- [6]
				{
					["level"] = 8,
					["subzone"] = "Villadorada",
					["t"] = 1791644257,
					["type"] = "death",
					["zone"] = "Bosque de Elwynn",
				}, -- [7]
				{
					["id"] = 120,
					["level"] = 8,
					["npc"] = "Alguacil Dughan",
					["objectives"] = "y",
					["subzone"] = "Villadorada",
					["t"] = 1791644258,
					["text"] = "x",
					["title"] = "Pañuelos rojos",
					["type"] = "quest_accept",
					["zone"] = "Bosque de Elwynn",
				}, -- [8]
				{
					["id"] = 120,
					["level"] = 8,
					["subzone"] = "Villadorada",
					["t"] = 1791644260,
					["title"] = "Pañuelos rojos",
					["type"] = "quest_abandon",
					["zone"] = "Bosque de Elwynn",
				}, -- [9]
				{
					["id"] = 111,
					["level"] = 8,
					["npc"] = "Maybell Maclure",
					["reward"] = "¡Su colgante! Gracias… nadie debe saberlo.",
					["subzone"] = "Villadorada",
					["t"] = 1791644860,
					["title"] = "Hablar con Maybell",
					["type"] = "quest_turnin",
					["zone"] = "Bosque de Elwynn",
				}, -- [10]
				{
					["id"] = 176,
					["level"] = 8,
					["npc"] = "Cartel de «Se busca»",
					["objectives"] = "Trae la garra de Hogger al alguacil Dughan, en Villadorada.",
					["subzone"] = "Villadorada",
					["t"] = 1791645060,
					["text"] = "SE BUSCA: un gnoll enorme llamado Hogger aterroriza el oeste del Bosque de Elwynn. La guardia de Ventormenta paga una recompensa por su cabeza. Se le ha visto cerca del Bosque Brumoso, al suroeste. Entregad la prueba al alguacil Dughan en Villadorada.",
					["title"] = "Se busca: Hogger",
					["type"] = "quest_accept",
					["zone"] = "Bosque de Elwynn",
				}, -- [11]
				{
					["amount"] = 250,
					["faction"] = "Ventormenta",
					["level"] = 8,
					["subzone"] = "Villadorada",
					["t"] = 1791645063,
					["type"] = "rep",
					["zone"] = "Bosque de Elwynn",
				}, -- [12]
				{
					["faction"] = "Ventormenta",
					["level"] = 8,
					["standing"] = "Amistoso",
					["standingID"] = 5,
					["subzone"] = "Villadorada",
					["t"] = 1791645065,
					["text"] = "La capital de los humanos en Azeroth.",
					["type"] = "standing",
					["zone"] = "Bosque de Elwynn",
				}, -- [13]
			},
			["level"] = 8,
			["name"] = "Tobias",
			["played"] = 6215,
			["race"] = "Humano",
			["realm"] = "Forever Beta",
			["seenItems"] = {
				[2079] = true,
			},
			["standings"] = {
				["Ventormenta"] = 5,
			},
			["surname"] = "Brenn",
		},
	},
	["config"] = {
		["espera"] = 240,
		["frases"] = "voz",
	},
	["version"] = 2,
}
