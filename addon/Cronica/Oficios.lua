-- Crónica · Banco y oficios: guarda lo que llevas en las bolsas y en el banco,
-- tus profesiones y las recetas que conoces o ves en los instructores, para
-- verlo en el móvil y calcular qué te falta para subir. Solo cantidades y
-- nombres: nada de horas.
--
-- Se guarda en CronicaDB.characters[<clave>].oficios y CronicaDB.nombres
-- (nombres de objetos en tu idioma, compartidos por toda la cuenta).

local ADDON = ...
local db, char

local function charKey()
	local first, surname = UnitName("player")
	if surname == "" then surname = nil end
	local realm = (GetNormalizedRealmName and GetNormalizedRealmName()) or (GetRealmName and GetRealmName()) or "Reino"
	realm = tostring(realm):gsub("[^%w]", "")
	local name = ((first or "?") .. (surname or "")):gsub("[%s%-]", "")
	return name .. "-" .. realm
end

local function itemID(link)
	return link and tonumber(link:match("item:(%d+)")) or nil
end

-- Nombres en tu idioma de los objetos que ve el addon.
local function remember(link, id)
	id = id or itemID(link)
	if not id or not db then return end
	local name = link and link:match("%[(.-)%]")
	if name and name ~= "" then db.nombres[id] = name end
end

-- ---------------------------------------------------------------------------
-- Bolsas y banco (clásico: C_Container en clientes nuevos, funciones sueltas en los viejos)
-- ---------------------------------------------------------------------------

local function slots(bag)
	if C_Container and C_Container.GetContainerNumSlots then return C_Container.GetContainerNumSlots(bag) or 0 end
	return GetContainerNumSlots and GetContainerNumSlots(bag) or 0
end

local function slotItem(bag, slot)
	if C_Container and C_Container.GetContainerItemInfo then
		local info = C_Container.GetContainerItemInfo(bag, slot)
		if info then return info.itemID or itemID(info.hyperlink), info.stackCount or 1, info.hyperlink, info.quality end
		return nil
	end
	if GetContainerItemInfo then
		local _, count, _, quality, _, _, link = GetContainerItemInfo(bag, slot)
		if link then return itemID(link), count or 1, link, quality end
	end
end

local function scan(bags)
	local out, quality = {}, {}
	for _, bag in ipairs(bags) do
		for s = 1, slots(bag) do
			local id, count, link, q = slotItem(bag, s)
			if id then
				out[id] = (out[id] or 0) + (count or 1)
				if q then quality[id] = q end
				remember(link, id)
			end
		end
	end
	return out, quality
end

local BAGS = { 0, 1, 2, 3, 4 }
local BANK = { -1, 5, 6, 7, 8, 9, 10 }

local function saveQuality(q)
	for id, v in pairs(q) do db.calidad[id] = v end
end

local function scanBags()
	if not char then return end
	-- La mochila siempre tiene huecos: si el juego dice 0, aún no ha cargado
	-- (o está cerrando) y no hay que borrar lo que había.
	if slots(0) == 0 then return end
	local items, q = scan(BAGS)
	char.oficios.bolsas = items
	saveQuality(q)
	-- Dinero, para las cuentas de lo que hay que comprar.
	if GetMoney then char.oficios.dinero = GetMoney() end
end

local bankOpen = false
local function scanBank()
	if not char or not bankOpen or slots(-1) == 0 then return end
	local items, q = scan(BANK)
	char.oficios.banco = items
	char.oficios.bancoVisto = true
	saveQuality(q)
end

-- ---------------------------------------------------------------------------
-- Profesiones y recetas
-- ---------------------------------------------------------------------------

-- Nombres de las profesiones (español e inglés), por si el juego no usa los
-- encabezados que esperamos.
local PROFS = {}
for _, n in ipairs({ "Herrería", "Minería", "Herboristería", "Desuello", "Alquimia", "Sastrería", "Peletería",
	"Ingeniería", "Encantamiento", "Cocina", "Primeros auxilios", "Pesca", "Joyería", "Inscripción",
	"Blacksmithing", "Mining", "Herbalism", "Skinning", "Alchemy", "Tailoring", "Leatherworking",
	"Engineering", "Enchanting", "Cooking", "First Aid", "Fishing" }) do PROFS[n] = true end

local function setSkill(name, rank, max)
	if not name or not rank or not max or max <= 1 then return end
	char.oficios.profesiones = char.oficios.profesiones or {}
	char.oficios.profesiones[name] = { nivel = rank, max = max }
end

-- Niveles de las profesiones. Se prueban las dos formas que tiene el juego:
-- la moderna (GetProfessions) y la de Classic (la lista de habilidades).
local function scanSkills()
	if not char then return end
	if GetProfessions and GetProfessionInfo then
		local found = false
		for _, idx in pairs({ GetProfessions() }) do
			local ok, name, _, rank, max = pcall(GetProfessionInfo, idx)
			if ok and name then setSkill(name, rank, max); found = true end
		end
		if found then return end
	end
	if not GetNumSkillLines or not GetSkillLineInfo then return end
	-- Las cabeceras plegadas esconden sus líneas: se despliegan un momento.
	local collapsed = {}
	for i = 1, GetNumSkillLines() do
		local name, isHeader, isExpanded = GetSkillLineInfo(i)
		if isHeader and not isExpanded then collapsed[#collapsed + 1] = name end
	end
	if #collapsed > 0 and ExpandSkillHeader then pcall(ExpandSkillHeader, 0) end
	local header
	for i = 1, GetNumSkillLines() do
		local name, isHeader, _, rank, _, _, max = GetSkillLineInfo(i)
		if isHeader then
			header = name
		elseif name and (PROFS[name] or header == TRADE_SKILLS or header == SECONDARY_SKILLS) then
			setSkill(name, rank, max)
		end
	end
	if #collapsed > 0 and CollapseSkillHeader then
		for i = GetNumSkillLines(), 1, -1 do
			local name, isHeader = GetSkillLineInfo(i)
			for _, c in ipairs(collapsed) do
				if isHeader and name == c then pcall(CollapseSkillHeader, i) end
			end
		end
	end
end

local DIFF = { optimal = "naranja", medium = "amarillo", easy = "verde", trivial = "gris" }
local RELDIFF = { [0] = "naranja", [1] = "amarillo", [2] = "verde", [3] = "gris" }

local function saveRecipes(prof, rank, max, list)
	if not prof or prof == "" or prof == "UNKNOWN" or #list == 0 then return end
	char.oficios.recetas = char.oficios.recetas or {}
	char.oficios.recetas[prof] = { nivel = rank, max = max, lista = list }
	setSkill(prof, rank, max)
end

-- La ventana de una profesión, forma de Classic.
local function scanTradeSkillOld()
	if not GetNumTradeSkills or not GetTradeSkillLine then return false end
	local prof, rank, max = GetTradeSkillLine()
	if not prof or prof == "UNKNOWN" then return false end
	local list = {}
	for i = 1, GetNumTradeSkills() do
		local name, kind = GetTradeSkillInfo(i)
		if name and kind ~= "header" and kind ~= "subheader" then
			local link = GetTradeSkillItemLink and GetTradeSkillItemLink(i)
			local r = { nombre = name, color = DIFF[kind] or kind, id = itemID(link), mats = {} }
			remember(link)
			for j = 1, (GetTradeSkillNumReagents and GetTradeSkillNumReagents(i) or 0) do
				local rname, _, need = GetTradeSkillReagentInfo(i, j)
				local rlink = GetTradeSkillReagentItemLink and GetTradeSkillReagentItemLink(i, j)
				remember(rlink)
				table.insert(r.mats, { id = itemID(rlink), nombre = rname, n = need })
			end
			table.insert(list, r)
		end
	end
	saveRecipes(prof, rank, max, list)
	return #list > 0
end

-- La ventana de una profesión, forma moderna (C_TradeSkillUI).
local function scanTradeSkillNew()
	local T = C_TradeSkillUI
	if not T or not T.GetAllRecipeIDs then return false end
	local prof, rank, max
	if T.GetBaseProfessionInfo then
		local ok, info = pcall(T.GetBaseProfessionInfo)
		if ok and type(info) == "table" then prof, rank, max = info.professionName, info.skillLevel, info.maxSkillLevel end
	end
	if not prof and T.GetTradeSkillLine then
		local ok, _, name, r, m = pcall(T.GetTradeSkillLine)
		if ok then prof, rank, max = name, r, m end
	end
	local ok, ids = pcall(T.GetAllRecipeIDs)
	if not ok or type(ids) ~= "table" then return false end
	local list = {}
	for _, rid in ipairs(ids) do
		local info = T.GetRecipeInfo and T.GetRecipeInfo(rid)
		if info and info.learned ~= false and info.name then
			local color = (type(info.difficulty) == "string" and DIFF[info.difficulty]) or RELDIFF[info.relativeDifficulty]
			local r = { nombre = info.name, color = color, mats = {} }
			local link = T.GetRecipeItemLink and T.GetRecipeItemLink(rid)
			if link then r.id = itemID(link); remember(link) end
			local schem = T.GetRecipeSchematic and select(2, pcall(T.GetRecipeSchematic, rid, false))
			if type(schem) == "table" then
				r.id = r.id or schem.outputItemID
				for _, slot in ipairs(schem.reagentSlotSchematics or {}) do
					local re = slot.reagents and slot.reagents[1]
					if re and re.itemID then table.insert(r.mats, { id = re.itemID, n = slot.quantityRequired or 1 }) end
				end
			elseif T.GetRecipeNumReagents then
				for j = 1, T.GetRecipeNumReagents(rid) or 0 do
					local rname, _, need = T.GetRecipeReagentInfo(rid, j)
					local rlink = T.GetRecipeReagentItemLink and T.GetRecipeReagentItemLink(rid, j)
					remember(rlink)
					table.insert(r.mats, { id = itemID(rlink), nombre = rname, n = need })
				end
			end
			table.insert(list, r)
		end
	end
	saveRecipes(prof, rank, max, list)
	return #list > 0
end

local lastTradeScan = "nada"
local function scanTradeSkill()
	if not char then return end
	local ok, done = pcall(scanTradeSkillOld)
	if ok and done then lastTradeScan = "clásica" return end
	local ok2, done2 = pcall(scanTradeSkillNew)
	if ok2 and done2 then lastTradeScan = "moderna" return end
	lastTradeScan = "sin datos" .. ((not ok and " (error clásica)") or "") .. ((not ok2 and " (error moderna)") or "")
end

-- El instructor: lo que enseña, desde qué nivel y cuánto cuesta.
local function scanTrainer()
	if not char or not GetNumTrainerServices then return end
	local list, prof = {}, nil
	for i = 1, GetNumTrainerServices() do
		local name, sub, kind = GetTrainerServiceInfo(i)
		if name and kind ~= "header" then
			local skill, need = nil, nil
			if GetTrainerServiceSkillReq then skill, need = GetTrainerServiceSkillReq(i) end
			prof = prof or skill
			local link = GetTrainerServiceItemLink and GetTrainerServiceItemLink(i)
			remember(link)
			table.insert(list, {
				nombre = name, rango = sub, estado = kind, id = itemID(link), nivel = need,
				precio = GetTrainerServiceCost and GetTrainerServiceCost(i) or nil,
			})
		end
	end
	if prof and #list > 0 then
		char.oficios.instructor = char.oficios.instructor or {}
		char.oficios.instructor[prof] = list
	end
end

-- ---------------------------------------------------------------------------
-- Nombres en tu idioma de los objetos de la tabla de recetas (los pide el
-- programa en CronicaOficiosIDs): así la app no enseña nombres en inglés.
-- ---------------------------------------------------------------------------

local pendingNames = {}
local function nameOf(id)
	if C_Item and C_Item.GetItemNameByID then return C_Item.GetItemNameByID(id) end
	return GetItemInfo and (GetItemInfo(id)) or nil
end

local function resolveNames()
	if type(CronicaOficiosIDs) ~= "table" then return end
	local asked = 0
	for _, id in ipairs(CronicaOficiosIDs) do
		if not db.nombres[id] then
			local name = nameOf(id)
			if name then
				db.nombres[id] = name
			elseif asked < 40 then
				asked = asked + 1
				pendingNames[id] = true
				if C_Item and C_Item.RequestLoadItemDataByID then pcall(C_Item.RequestLoadItemDataByID, id) end
			end
		end
	end
	-- Lo que falte se pide poco a poco para no cargar el juego.
	if asked > 0 and C_Timer then C_Timer.After(3, resolveNames) end
end

-- ---------------------------------------------------------------------------
-- Eventos
-- ---------------------------------------------------------------------------

local f = CreateFrame("Frame")
local function on(event, fn)
	if pcall(f.RegisterEvent, f, event) then f[event] = fn end
end
f:SetScript("OnEvent", function(self, event, ...) if self[event] then self[event](...) end end)

local bagsSoon = false
local function scanBagsSoon()
	if bagsSoon then return end
	bagsSoon = true
	C_Timer.After(1, function()
		bagsSoon = false
		scanBags()
		scanBank()
	end)
end

on("ADDON_LOADED", function(name)
	if name ~= ADDON then return end
	CronicaDB = CronicaDB or {}
	db = CronicaDB
	db.nombres = db.nombres or {}
	db.calidad = db.calidad or {}
end)

on("PLAYER_LOGIN", function()
	db.characters = db.characters or {}
	local key = charKey()
	db.characters[key] = db.characters[key] or { events = {}, played = 0, seenItems = {} }
	char = db.characters[key]
	char.oficios = char.oficios or {}
	C_Timer.After(5, function() scanBags(); scanSkills(); resolveNames() end)
end)

on("BAG_UPDATE_DELAYED", scanBagsSoon)
on("BAG_UPDATE", scanBagsSoon)
on("PLAYER_MONEY", function() if char and GetMoney then char.oficios.dinero = GetMoney() end end)
on("BANKFRAME_OPENED", function() bankOpen = true; scanBagsSoon() end)
on("BANKFRAME_CLOSED", function() scanBank(); bankOpen = false end)
on("PLAYERBANKSLOTS_CHANGED", scanBagsSoon)
on("SKILL_LINES_CHANGED", function() if char then scanSkills() end end)
on("TRADE_SKILL_SHOW", function() C_Timer.After(0.5, scanTradeSkill) end)
on("TRADE_SKILL_UPDATE", function() if char then scanTradeSkill() end end)
-- En el cliente moderno la lista llega con estos otros avisos.
local tradeSoon = false
local function scanTradeSoon()
	if tradeSoon or not char then return end
	tradeSoon = true
	C_Timer.After(0.5, function() tradeSoon = false; scanTradeSkill() end)
end
on("TRADE_SKILL_LIST_UPDATE", scanTradeSoon)
on("TRADE_SKILL_DATA_SOURCE_CHANGED", scanTradeSoon)
on("NEW_RECIPE_LEARNED", scanTradeSoon)
on("TRAINER_SHOW", function() C_Timer.After(0.5, scanTrainer) end)
on("TRAINER_UPDATE", function() if char then scanTrainer() end end)
on("GET_ITEM_INFO_RECEIVED", function(id, ok)
	if id and pendingNames[id] then
		pendingNames[id] = nil
		local name = ok ~= false and nameOf(id)
		if name then db.nombres[id] = name end
	end
end)
-- Al salir no se vuelve a mirar: el juego ya está vaciando las bolsas y lo
-- guardado durante la partida es lo bueno.

-- /cronica oficios: lo que ha encontrado (para saber qué falla si algo no sale).
local previous = SlashCmdList and SlashCmdList.CRONICA
if SlashCmdList then
	SlashCmdList.CRONICA = function(msg)
		if (msg or ""):lower():match("^%s*oficios") then
			local say = function(t) DEFAULT_CHAT_FRAME:AddMessage("|cffd8b061Crónica:|r " .. t) end
			if not char then say("aún cargando…") return end
			scanSkills(); scanBags(); scanBank()
			if C_TradeSkillUI and C_TradeSkillUI.IsTradeSkillReady and C_TradeSkillUI.IsTradeSkillReady() then scanTradeSkill() end
			local api = {}
			for _, n in ipairs({ "GetProfessions", "GetNumSkillLines", "GetNumTradeSkills", "GetTradeSkillLine" }) do
				api[#api + 1] = n .. (_G[n] and " sí" or " no")
			end
			api[#api + 1] = "C_TradeSkillUI" .. (C_TradeSkillUI and C_TradeSkillUI.GetAllRecipeIDs and " sí" or " no")
			say("funciones del juego: " .. table.concat(api, ", "))
			local n = 0
			for name, s in pairs(char.oficios.profesiones or {}) do say(("%s %d/%d"):format(name, s.nivel or 0, s.max or 0)); n = n + 1 end
			if n == 0 then say("no encuentro profesiones") end
			for name, r in pairs(char.oficios.recetas or {}) do say(("%s: %d recetas guardadas"):format(name, #(r.lista or {}))) end
			say("última ventana de profesión: " .. lastTradeScan .. ". Haz /reload para enviarlo a Crónica.")
			return
		end
		if previous then previous(msg) end
	end
end
