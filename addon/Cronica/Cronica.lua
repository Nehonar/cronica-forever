-- Crónica: registra lo que vive tu personaje para que el programa Crónica lo narre.
-- Los datos se guardan en CronicaDB (WTF/Account/<CUENTA>/SavedVariables/Cronica.lua),
-- que el juego escribe al salir o al hacer /reload.

local ADDON = ...
local VERSION = "0.1.0"

-- Valores por defecto de la configuración (por cuenta).
local DEFAULTS = {
	frases = "local",   -- "no", "local" (solo tú) o "voz" (/decir con una tecla)
	espera = 240,       -- segundos mínimos entre frases
}

-- Probabilidad de que el personaje diga algo en cada momento.
local CHANCE = {
	nivel = 1.0, equipo = 1.0, zona = 0.8, muerte = 0.6,
	aceptar = 0.15, entregar = 0.15, descanso = 0.3,
}

-- Frases genéricas por si el programa aún no ha escrito la baraja del personaje.
local GENERIC = {
	nivel = { "Algo ha cambiado. Me noto más fuerte.", "Un paso más. Y otro. Así se llega." },
	muerte = { "Me he caído más veces de las que recuerdo. Arriba.", "No ha sido mi mejor momento." },
	zona = { "Tierra nueva. A ver qué me tiene guardado.", "Nunca había pisado este sitio." },
	aceptar = { "Otro encargo. Bien.", "Alguien tiene que hacerlo." },
	entregar = { "Hecho. ¿Qué más?", "Uno menos." },
	equipo = { "{objeto}. Esto se nota.", "No está mal, {objeto}. Nada mal." },
	descanso = { "Un rato de descanso no le hace mal a nadie.", "Cerveza, fuego y un banco. Suficiente." },
}

local SLOT_NAMES = {
	[1] = "HEAD", [2] = "NECK", [3] = "SHOULDER", [5] = "CHEST", [6] = "WAIST", [7] = "LEGS",
	[8] = "FEET", [9] = "WRIST", [10] = "HANDS", [11] = "FINGER", [12] = "FINGER",
	[13] = "TRINKET", [14] = "TRINKET", [15] = "BACK", [16] = "MAINHAND", [17] = "OFFHAND", [18] = "RANGED",
}

local GOLD = "|cffd8b061"
local RED = "|cffff5040"
local function say(msg) DEFAULT_CHAT_FRAME:AddMessage(GOLD .. "Crónica:|r " .. msg) end

-- ---------------------------------------------------------------------------
-- Datos del personaje
-- ---------------------------------------------------------------------------

local db, char, key
local sessionStart = GetTime()
local loginGrace = 0           -- durante unos segundos tras entrar no se registra el equipo
local detailCache, completeCache = {}, {}
local acceptedNow, turnedIn = {}, {}

-- En Forever los personajes tienen nombre y apellido, y solo el nombre completo es
-- único: UnitName("player") devuelve los dos (en otros clientes, el segundo es nil).
local function playerNames()
	local first, surname = UnitName("player")
	if surname == "" then surname = nil end
	return first or "?", surname
end

local function charKey()
	local first, surname = playerNames()
	local realm = (GetNormalizedRealmName and GetNormalizedRealmName()) or (GetRealmName and GetRealmName()) or "Reino"
	realm = tostring(realm):gsub("[^%w]", "")
	local name = (first .. (surname or "")):gsub("[%s%-]", "")
	return name .. "-" .. realm
end

local function where()
	local zone = (GetRealZoneText and GetRealZoneText()) or ""
	local sub = (GetSubZoneText and GetSubZoneText()) or ""
	if sub == "" and GetMinimapZoneText then sub = GetMinimapZoneText() or "" end
	if sub == zone then sub = "" end
	return zone, sub
end

local function add(ev)
	if not char then return end
	ev.t = ev.t or time()
	local zone, sub = where()
	ev.zone = ev.zone or zone
	if (ev.subzone == nil or ev.subzone == "") and sub ~= "" then ev.subzone = sub end
	ev.level = ev.level or UnitLevel("player")
	table.insert(char.events, ev)
	char.level = UnitLevel("player")
	return ev
end

local function setupCharacter()
	key = charKey()
	db.characters = db.characters or {}
	char = db.characters[key] or { events = {}, played = 0, seenItems = {} }
	db.characters[key] = char
	char.events = char.events or {}
	char.seenItems = char.seenItems or {}
	char.played = char.played or 0
	char.name, char.surname = playerNames()
	char.realm = (GetRealmName and GetRealmName()) or ""
	local race = UnitRace("player")
	local class, classFile = UnitClass("player")
	char.race, char.class, char.classFile = race, class, classFile
	char.level = UnitLevel("player")
end

-- ---------------------------------------------------------------------------
-- Misiones
-- ---------------------------------------------------------------------------

local function npcName()
	return UnitName("questnpc") or UnitName("npc") or ""
end

local function titleFor(id)
	if C_QuestLog and C_QuestLog.GetTitleForQuestID then
		local t = C_QuestLog.GetTitleForQuestID(id)
		if t then return t end
	end
	return detailCache[id] and detailCache[id].title or ("Misión " .. id)
end

local function recordAccept(id)
	if not id or id == 0 or acceptedNow[id] then return end
	acceptedNow[id] = true
	local d = detailCache[id] or {}
	add({
		type = "quest_accept", id = id, title = d.title or titleFor(id), npc = d.npc or "",
		text = d.text or "", objectives = d.objectives or "",
	})
end

local function onQuestDetail()
	local id = GetQuestID and GetQuestID() or 0
	if id and id > 0 then
		detailCache[id] = {
			title = GetTitleText and GetTitleText() or "",
			text = GetQuestText and GetQuestText() or "",
			objectives = GetObjectiveText and GetObjectiveText() or "",
			npc = npcName(),
		}
	end
end

local function onQuestComplete()
	local id = GetQuestID and GetQuestID() or 0
	if id and id > 0 then
		completeCache[id] = { reward = GetRewardText and GetRewardText() or "", npc = npcName() }
	end
end

local function onTurnedIn(id)
	if not id or turnedIn[id] then return end
	turnedIn[id] = true
	local c = completeCache[id] or {}
	add({ type = "quest_turnin", id = id, title = titleFor(id), npc = c.npc or npcName(), reward = c.reward or "" })
end

-- ---------------------------------------------------------------------------
-- Equipo
-- ---------------------------------------------------------------------------

local function itemStats(link)
	local fn = (C_Item and C_Item.GetItemStats) or GetItemStats
	if not fn then return nil end
	local ok, t = pcall(fn, link)
	if not ok or type(t) ~= "table" then return nil end
	local out, armor = {}, 0
	for k, v in pairs(t) do
		if k == "RESISTANCE0_NAME" then armor = v
		elseif type(v) == "number" and v ~= 0 then out[k] = v end
	end
	return out, armor
end

local function onEquip(slot, tries)
	if not char or GetTime() < loginGrace then return end
	local link = GetInventoryItemLink("player", slot)
	if not link then return end
	local name, _, quality, _, _, itemType, itemSubType = GetItemInfo(link)
	if not name then
		-- La información del objeto aún no está en caché: reintentar.
		if (tries or 0) < 5 and C_Timer then C_Timer.After(1, function() onEquip(slot, (tries or 0) + 1) end) end
		return
	end
	if (quality or 0) < 3 then return end
	local itemID = tonumber(link:match("item:(%d+)")) or name
	if char.seenItems[itemID] then return end
	char.seenItems[itemID] = true
	local stats, armor = itemStats(link)
	add({
		type = "equip", item = name, quality = quality, slot = SLOT_NAMES[slot] or tostring(slot),
		itemType = itemType or "", itemSubType = itemSubType or "", armor = armor or 0, stats = stats,
	})
	Cronica_Frase("equipo", name)
end

-- ---------------------------------------------------------------------------
-- Frases del personaje
-- ---------------------------------------------------------------------------

local lastPhrase = -1e9
local pending                     -- { text, expires } en modo «voz»
local queued                      -- frase guardada para cuando acabe el combate
local bags = {}

local subtitle = CreateFrame("Button", "CronicaSubtitulo", UIParent)
subtitle:SetSize(700, 60)
subtitle:SetPoint("BOTTOM", UIParent, "BOTTOM", 0, 190)
subtitle:SetFrameStrata("HIGH")
subtitle:Hide()
subtitle.text = subtitle:CreateFontString(nil, "OVERLAY", "GameFontNormalLarge")
subtitle.text:SetPoint("CENTER")
subtitle.text:SetWidth(680)
subtitle.text:SetTextColor(1, 0.86, 0.55)
subtitle.hint = subtitle:CreateFontString(nil, "OVERLAY", "GameFontHighlightSmall")
subtitle.hint:SetPoint("TOP", subtitle.text, "BOTTOM", 0, -4)
subtitle:RegisterForClicks("AnyUp")
subtitle:SetScript("OnClick", function() Cronica_Decir() end)

local function hideLater(seconds)
	local token = {}
	subtitle.token = token
	C_Timer.After(seconds, function()
		if subtitle.token == token then
			subtitle:Hide()
			pending = nil
		end
	end)
end

local function draw(cat)
	local deck = (CronicaFrases and key and CronicaFrases[key] and CronicaFrases[key][cat]) or GENERIC[cat]
	if not deck or #deck == 0 then return nil end
	-- Baraja sin repetir: se reparte entera antes de volver a mezclar.
	local bag = bags[cat]
	if not bag or #bag == 0 then
		bag = {}
		for i = 1, #deck do bag[i] = deck[i] end
		for i = #bag, 2, -1 do
			local j = math.random(i)
			bag[i], bag[j] = bag[j], bag[i]
		end
		bags[cat] = bag
	end
	return table.remove(bag)
end

local function show(text)
	local mode = db.config.frases
	local name = (char and char.name) or UnitName("player")
	if mode == "voz" then
		pending = { text = text, expires = GetTime() + 15 }
		local keyName = GetBindingKey and GetBindingKey("CRONICA_DECIR")
		subtitle.text:SetText("«" .. text .. "»")
		subtitle.hint:SetText(keyName and ("Pulsa " .. keyName .. " o haz clic para decirlo") or "Haz clic para decirlo (asigna una tecla en Atajos > Crónica)")
		subtitle:Show()
		hideLater(15)
	else
		subtitle.text:SetText("«" .. text .. "»")
		subtitle.hint:SetText("")
		subtitle:Show()
		hideLater(7)
		DEFAULT_CHAT_FRAME:AddMessage(GOLD .. name .. " piensa:|r |cffe9e0cc" .. text .. "|r")
	end
end

-- Puede decir algo según la categoría. extra sustituye a {objeto}.
function Cronica_Frase(cat, extra, force)
	if not db or db.config.frases == "no" then return end
	if not force then
		if math.random() > (CHANCE[cat] or 0.2) then return end
		local wait = db.config.espera or 240
		if cat == "nivel" or cat == "equipo" then wait = math.min(wait, 60) end
		if GetTime() - lastPhrase < wait then return end
	end
	if InCombatLockdown() or UnitAffectingCombat("player") then
		queued = { cat = cat, extra = extra, t = GetTime() }
		return
	end
	local text = draw(cat)
	if not text then return end
	text = text:gsub("{objeto}", extra or "")
	lastPhrase = GetTime()
	show(text)
end

-- Dice en /decir la frase pendiente. Solo funciona desde una tecla o un clic:
-- Blizzard no deja que los addons hablen por su cuenta fuera de las mazmorras.
function Cronica_Decir()
	if not pending or GetTime() > pending.expires then return end
	SendChatMessage(pending.text, "SAY")
	pending = nil
	subtitle:Hide()
end

-- Recargar (para que Crónica reciba lo nuevo sin cerrar sesión).
-- ReloadUI() está reservada a Blizzard: un addon no puede llamarla ni desde un
-- clic. Lo que sí se puede es un botón seguro que ejecuta la macro «/reload»;
-- se usa con una tecla (Opciones → Atajos → Crónica).
local reloadButton = CreateFrame("Button", "CronicaRecargar", UIParent, "SecureActionButtonTemplate")
reloadButton:SetAttribute("type", "macro")
reloadButton:SetAttribute("macrotext", "/reload")
reloadButton:RegisterForClicks("AnyUp", "AnyDown")

BINDING_HEADER_CRONICA = "Crónica"
BINDING_NAME_CRONICA_DECIR = "Decir en voz alta la frase de mi personaje"
_G["BINDING_NAME_CLICK CronicaRecargar:LeftButton"] = "Enviar a Crónica (recarga la interfaz)"

-- ---------------------------------------------------------------------------
-- Eventos
-- ---------------------------------------------------------------------------

local f = CreateFrame("Frame")
local function on(event, fn) f:RegisterEvent(event); f[event] = fn end
f:SetScript("OnEvent", function(self, event, ...) if self[event] then self[event](...) end end)

-- Tiempo jugado: el del propio juego (/played), pedido en silencio.
local playedBase, playedAt, silentPlayed, pendingLevel = nil, 0, 0, nil

local function currentPlayed()
	if playedBase then return playedBase + math.floor(GetTime() - playedAt) end
	if char then return (char.played or 0) + math.floor(GetTime() - sessionStart) end
end

local function requestPlayed()
	if not RequestTimePlayed then return end
	silentPlayed = silentPlayed + 1
	RequestTimePlayed()
	-- Si el juego no llega a mostrarlo, que tu propio /played no se quede sin salir.
	C_Timer.After(5, function() silentPlayed = 0 end)
end

-- Que la respuesta no salga en el chat cuando la pedimos nosotros.
if ChatFrame_DisplayTimePlayed then
	local original = ChatFrame_DisplayTimePlayed
	ChatFrame_DisplayTimePlayed = function(...)
		if silentPlayed > 0 then silentPlayed = silentPlayed - 1; return end
		return original(...)
	end
end

on("TIME_PLAYED_MSG", function(total, thisLevel)
	if not total then return end
	playedBase, playedAt = total, GetTime()
	if char then char.played = total end
	if pendingLevel and thisLevel then
		pendingLevel.played = total - thisLevel
		pendingLevel = nil
	end
end)


on("ADDON_LOADED", function(name)
	if name ~= ADDON then return end
	CronicaDB = CronicaDB or {}
	db = CronicaDB
	db.version = 2
	db.config = db.config or {}
	for k, v in pairs(DEFAULTS) do
		if db.config[k] == nil then db.config[k] = v end
	end
end)

on("PLAYER_LOGIN", function()
	setupCharacter()
end)

on("PLAYER_ENTERING_WORLD", function(isInitialLogin, isReloadingUi)
	sessionStart = GetTime()
	loginGrace = GetTime() + 8
	C_Timer.After(3, requestPlayed)
	if isInitialLogin or (isInitialLogin == nil and not isReloadingUi) then
		add({ type = "login" })
	end
	-- Avisos del cronista (los escribe el programa en CronicaTextos.lua).
	C_Timer.After(6, function()
		if CronicaEstado and CronicaEstado.ok == false and CronicaEstado.mensaje and CronicaEstado.mensaje ~= "" then
			say(RED .. CronicaEstado.mensaje .. "|r")
		end
		local list = CronicaTextos and CronicaTextos[key]
		if type(list) == "table" then
			local newest, count = char.lastStoryT or 0, 0
			for _, s in ipairs(list) do
				if (s.t or 0) > (char.lastStoryT or 0) then count = count + 1; newest = math.max(newest, s.t or 0) end
			end
			if count > 0 then
				say(("%d relato(s) nuevo(s) de %s. Léelos en la app de Crónica."):format(count, char.name or ""))
				char.lastStoryT = newest
			end
		end
	end)
end)

on("PLAYER_LOGOUT", function()
	if char then
		char.played = currentPlayed() or char.played or 0
	end
end)

on("PLAYER_LEVEL_UP", function(level)
	-- Tiempo jugado al subir (para «tiempo en cada nivel»); sin fecha ni hora.
	pendingLevel = add({ type = "level", level = level, played = currentPlayed() })
	C_Timer.After(1, requestPlayed)
	Cronica_Frase("nivel")
end)

local lastZone
on("ZONE_CHANGED_NEW_AREA", function()
	local zone = GetRealZoneText and GetRealZoneText() or ""
	if zone == "" or zone == lastZone then return end
	lastZone = zone
	add({ type = "zone" })
	char.zonesSeen = char.zonesSeen or {}
	if not char.zonesSeen[zone] then
		char.zonesSeen[zone] = true
		Cronica_Frase("zona")
	end
end)

on("QUEST_DETAIL", function()
	onQuestDetail()
end)
on("QUEST_COMPLETE", onQuestComplete)

on("QUEST_ACCEPTED", function(a, b)
	-- Classic: (índice, idMisión). Moderno: (idMisión).
	local id = b or a
	recordAccept(id)
	Cronica_Frase("aceptar")
end)

on("QUEST_TURNED_IN", function(id)
	onTurnedIn(id)
	Cronica_Frase("entregar")
end)

on("QUEST_REMOVED", function(id)
	if not id then return end
	local title = titleFor(id)
	C_Timer.After(1, function()
		-- Si no se ha entregado, es que la has abandonado.
		if not turnedIn[id] then
			acceptedNow[id] = nil -- por si la vuelves a coger
			add({ type = "quest_abandon", id = id, title = title })
		end
	end)
end)

on("PLAYER_EQUIPMENT_CHANGED", function(slot) onEquip(slot) end)

-- ---------------------------------------------------------------------------
-- Reputación: lo ganado con cada facción (para el contexto de los relatos) y
-- cada vez que sube de rango (Amistoso, Honorable…), que es un hito con lore.
-- ---------------------------------------------------------------------------

-- Convierte un texto del juego como «Tu reputación con %s ha aumentado en %d.»
-- en un patrón que devuelve sus partes (respeta %1$s, %2$d de otros idiomas).
local function matcher(fmt)
	if type(fmt) ~= "string" then return function() end end
	local order = {}
	for n in fmt:gmatch("%%(%d)%$") do order[#order + 1] = tonumber(n) end
	local p = fmt:gsub("%%%d%$", "%%")
	p = p:gsub("([%(%)%.%[%]%*%+%-%?%^%$])", "%%%1")
	p = p:gsub("%%s", "(.+)"):gsub("%%d", "(%%d+)")
	p = "^" .. p .. "$"
	return function(msg)
		local c = { msg:match(p) }
		if #c == 0 then return end
		if #order == #c then
			local r = {}
			for i, n in ipairs(order) do r[n] = c[i] end
			c = r
		end
		return (unpack or table.unpack)(c)
	end
end

local repIncreased = matcher(FACTION_STANDING_INCREASED)

-- Recorre las facciones visibles y apunta las que han subido de rango.
local function scanFactions()
	if not char or not GetNumFactions or not GetFactionInfo then return end
	char.standings = char.standings or {}
	for i = 1, GetNumFactions() do
		local name, description, standingID, _, _, _, _, _, isHeader, _, hasRep = GetFactionInfo(i)
		if name and standingID and (not isHeader or hasRep) then
			local prev = char.standings[name]
			if prev and standingID > prev and standingID >= 5 then -- 5 = Amistoso
				add({ type = "standing", faction = name, standingID = standingID,
					standing = _G["FACTION_STANDING_LABEL" .. standingID] or tostring(standingID),
					text = description })
			end
			char.standings[name] = standingID
		end
	end
end

local scanPending = false
local function scanSoon()
	if scanPending then return end
	scanPending = true
	C_Timer.After(1, function() scanPending = false; scanFactions() end)
end

local function onFactionMessage(msg)
	if not msg then return end
	local faction, amount = repIncreased(msg)
	if faction and tonumber(amount) then
		add({ type = "rep", faction = faction, amount = tonumber(amount) })
	end
	scanSoon()
end

on("CHAT_MSG_COMBAT_FACTION_CHANGE", onFactionMessage)
on("UPDATE_FACTION", scanSoon)

on("PLAYER_DEAD", function()
	add({ type = "death" })
	Cronica_Frase("muerte")
end)

on("PLAYER_UPDATE_RESTING", function()
	if IsResting() then Cronica_Frase("descanso") end
end)

on("PLAYER_REGEN_ENABLED", function()
	if queued and GetTime() - queued.t < 30 then
		local q = queued
		queued = nil
		Cronica_Frase(q.cat, q.extra, true)
	end
	queued = nil
end)

-- ---------------------------------------------------------------------------
-- Comandos
-- ---------------------------------------------------------------------------

local MODES = { no = "desactivadas", ["local"] = "solo para ti", voz = "en voz alta (/decir con tecla o clic)" }

SLASH_CRONICA1 = "/cronica"
SLASH_CRONICA2 = "/cr"
SlashCmdList.CRONICA = function(msg)
	local cmd, arg = (msg or ""):lower():match("^(%S*)%s*(.-)$")
	if cmd == "frases" then
		if MODES[arg] then
			db.config.frases = arg
			say("frases " .. MODES[arg] .. ".")
		else
			say("usa: /cronica frases no | local | voz")
		end
	elseif cmd == "espera" then
		local m = tonumber(arg)
		if m and m >= 1 and m <= 60 then
			db.config.espera = m * 60
			say(("como mucho una frase cada %d minuto(s)."):format(m))
		else
			say("usa: /cronica espera <minutos> (1 a 60)")
		end
	elseif cmd == "leer" or cmd == "enviar" or cmd == "boton" then
		say("para enviar lo nuevo a Crónica escribe /reload o usa tu tecla (Opciones → Atajos → Crónica). Al cerrar sesión se envía solo.")
	elseif cmd == "prueba" then
		Cronica_Frase("descanso", nil, true)
	else
		local n = char and #char.events or 0
		say(("v%s · %s · %d suceso(s) registrado(s)."):format(VERSION, key or "?", n))
		say("frases: " .. (MODES[db.config.frases] or "?") .. " · espera: " .. math.floor((db.config.espera or 240) / 60) .. " min")
		if CronicaEstado and CronicaEstado.ok == false then say(RED .. (CronicaEstado.mensaje or "") .. "|r") end
		say("comandos: /cronica frases no|local|voz · espera <min> · prueba · (para enviar a Crónica: /reload o tu tecla)")
	end
end
