-- Crónica · Modo viaje: juega sin mirar números.
--
--   /cronica viaje   oculta la barra de experiencia y cambia los niveles por un
--                    icono de color (gris, verde, amarillo, naranja, rojo, calavera),
--                    como el color de las misiones. Élites y raros llevan marca aparte.
--   /cronica subida  sube de nivel en silencio: sin sonido ni mensajes de
--                    «has alcanzado el nivel». Te enterarás en el instructor.
--
-- Las dos opciones se guardan en CronicaDB.config (por cuenta) y empiezan apagadas.
-- La columna de luz dorada al subir de nivel es un efecto del propio juego:
-- ningún addon puede quitarla.

local LEVELUP_SOUND = 569593 -- Sound/Spells/LevelUp.ogg

local TEX_DOT = "Interface\\CharacterFrame\\TempPortraitAlphaMask"
local TEX_SKULL = "Interface\\TargetingFrame\\UI-TargetingFrame-Skull"
local TEX_STAR = "Interface\\TargetingFrame\\UI-RaidTargetingIcon_1"
local TEX_DIAMOND = "Interface\\TargetingFrame\\UI-RaidTargetingIcon_3"

-- Palabra que acompaña al icono en la descripción del ratón.
local WORD = { gris = "Trivial", verde = "Fácil", amarillo = "Igualado", naranja = "Difícil", rojo = "Peligroso", calavera = "Imposible" }
local RGB = {
	gris = { 0.62, 0.62, 0.62 }, verde = { 0.25, 0.75, 0.25 }, amarillo = { 1, 0.82, 0 },
	naranja = { 1, 0.5, 0.25 }, rojo = { 1, 0.1, 0.1 }, calavera = { 1, 1, 1 },
}

local cfg -- CronicaDB.config
local GOLD = "|cffd8b061"
local function say(msg) DEFAULT_CHAT_FRAME:AddMessage(GOLD .. "Crónica:|r " .. msg) end
local function viajeOn() return cfg and cfg.viaje end
local function subidaOn() return cfg and cfg.subida end

-- ---------------------------------------------------------------------------
-- Dificultad: el mismo color que el juego usa para las misiones
-- ---------------------------------------------------------------------------

local function difficulty(unit)
	local lvl = UnitLevel(unit)
	if not lvl or lvl <= 0 then return "calavera" end
	local me = UnitLevel("player")
	local diff = lvl - me
	if diff >= 10 then return "calavera" end
	if diff >= 5 then return "rojo" end
	if diff >= 3 then return "naranja" end
	if diff >= -2 then return "amarillo" end
	local green = (GetQuestGreenRange and GetQuestGreenRange()) or 5
	if -diff <= green then return "verde" end
	return "gris"
end

local function badgeTexture(cls)
	if cls == "elite" or cls == "rareelite" then return TEX_STAR end
	if cls == "rare" then return TEX_DIAMOND end
	if cls == "worldboss" then return TEX_SKULL end
end

-- Un icono: disco de color con borde oscuro, calavera si es imposible y marca de élite o raro.
local function makeIcon(parent, size)
	local icon = CreateFrame("Frame", nil, parent)
	icon:SetSize(size, size)
	icon:SetFrameLevel(parent:GetFrameLevel() + 5)
	icon.ring = icon:CreateTexture(nil, "ARTWORK", nil, 1)
	icon.ring:SetTexture(TEX_DOT)
	icon.ring:SetAllPoints()
	icon.ring:SetVertexColor(0, 0, 0, 0.85)
	icon.dot = icon:CreateTexture(nil, "ARTWORK", nil, 2)
	icon.dot:SetTexture(TEX_DOT)
	icon.dot:SetPoint("TOPLEFT", 2, -2)
	icon.dot:SetPoint("BOTTOMRIGHT", -2, 2)
	icon.skull = icon:CreateTexture(nil, "ARTWORK", nil, 3)
	icon.skull:SetTexture(TEX_SKULL)
	icon.skull:SetAllPoints()
	icon.badge = icon:CreateTexture(nil, "OVERLAY")
	icon.badge:SetSize(size * 0.6, size * 0.6)
	icon.badge:SetPoint("CENTER", icon, "TOPRIGHT", -1, -1)
	icon:Hide()
	return icon
end

local function paintIcon(icon, unit)
	if not icon then return end
	if not viajeOn() or not unit or not UnitExists(unit) then icon:Hide(); return end
	local d = difficulty(unit)
	if d == "calavera" then
		icon.dot:Hide(); icon.ring:Hide(); icon.skull:Show()
	else
		local c = RGB[d]
		icon.dot:SetVertexColor(c[1], c[2], c[3])
		icon.dot:Show(); icon.ring:Show(); icon.skull:Hide()
	end
	local badge = badgeTexture(UnitClassification(unit))
	if badge and d ~= "calavera" then icon.badge:SetTexture(badge); icon.badge:Show() else icon.badge:Hide() end
	icon:Show()
end

-- Las texturas y textos del juego no se ocultan, se hacen transparentes:
-- así no peleamos con el juego cada vez que los vuelve a mostrar.
local function fade(obj, hidden)
	if obj and obj.SetAlpha then obj:SetAlpha(hidden and 0 or 1) end
end

-- ---------------------------------------------------------------------------
-- Barra de experiencia
-- ---------------------------------------------------------------------------

local XP_FRAMES = { "MainMenuExpBar", "ExhaustionTick", "ExhaustionLevelFillBar", "StatusTrackingBarManager", "MainStatusTrackingBarContainer" }
local hookedBars = {}

local function updateXPBar()
	local found = false
	for _, name in ipairs(XP_FRAMES) do
		local fr = _G[name]
		if fr then
			found = true
			if not hookedBars[fr] then
				hookedBars[fr] = true
				fr:HookScript("OnShow", function(self) if viajeOn() then self:Hide() end end)
			end
			if viajeOn() then fr:Hide() end
		end
	end
	return found
end

-- ---------------------------------------------------------------------------
-- Marcos de jugador y objetivo
-- ---------------------------------------------------------------------------

local function playerLevelText()
	return _G.PlayerLevelText or (PlayerFrame and PlayerFrame.PlayerFrameContent and PlayerFrame.PlayerFrameContent.PlayerFrameContentMain and PlayerFrame.PlayerFrameContent.PlayerFrameContentMain.PlayerLevelText)
end

local function targetParts()
	local main = TargetFrame and TargetFrame.TargetFrameContent and TargetFrame.TargetFrameContent.TargetFrameContentMain
	local text = _G.TargetFrameTextureFrameLevelText or (main and main.LevelText)
	local skull = _G.TargetFrameTextureFrameHighLevelTexture or (main and main.HighLevelTexture)
	return text, skull
end

local targetIcon

local function updateTarget()
	local text, skull = targetParts()
	if not text then return end
	if not targetIcon then
		targetIcon = makeIcon(TargetFrame, 16)
		targetIcon:SetPoint("CENTER", text, "CENTER", 0, 0)
	end
	fade(text, viajeOn())
	fade(skull, viajeOn())
	paintIcon(targetIcon, "target")
end

local function updatePlayer()
	fade(playerLevelText(), viajeOn())
end

-- Ventana de personaje: «Nivel 12 Humano Guerrero» → «Humano Guerrero».
local function updatePaperDoll()
	local t = _G.CharacterLevelText
	if not t or not viajeOn() then return end
	local race = UnitRace("player") or ""
	local class = UnitClass("player") or ""
	t:SetText((race .. " " .. class):gsub("^%s+", ""))
end

-- ---------------------------------------------------------------------------
-- Placas de nombre
-- ---------------------------------------------------------------------------

local function updatePlate(unit)
	if not C_NamePlate then return end
	local plate = C_NamePlate.GetNamePlateForUnit(unit)
	if not plate or (plate.IsForbidden and plate:IsForbidden()) then return end
	local uf = plate.UnitFrame
	if not uf then return end
	local hide = viajeOn()
	fade(uf.LevelFrame, hide); fade(uf.levelText, hide); fade(uf.LevelText, hide)
	if not UnitCanAttack("player", unit) then
		if plate.cronicaIcon then plate.cronicaIcon:Hide() end
		return
	end
	if not plate.cronicaIcon then
		plate.cronicaIcon = makeIcon(uf, 12)
		local anchor = uf.healthBar or uf.HealthBar or uf
		plate.cronicaIcon:SetPoint("RIGHT", anchor, "LEFT", -2, 0)
	end
	paintIcon(plate.cronicaIcon, unit)
end

local function updateAllPlates()
	if not C_NamePlate then return end
	for _, plate in ipairs(C_NamePlate.GetNamePlates()) do
		local unit = plate.namePlateUnitToken or (plate.UnitFrame and plate.UnitFrame.unit)
		if unit then updatePlate(unit) end
	end
end

-- ---------------------------------------------------------------------------
-- Descripción del ratón: «Nivel 14 Humanoide» → icono + «Igualado Humanoide»
-- ---------------------------------------------------------------------------

local function esc(s) return (s:gsub("([%^%$%(%)%%%.%[%]%*%+%-%?])", "%%%1")) end

local function tooltipIcon(d)
	if d == "calavera" then return "|T" .. TEX_SKULL .. ":14:14|t" end
	local c = RGB[d]
	return ("|T%s:12:12:0:0:64:64:0:64:0:64:%d:%d:%d|t"):format(TEX_DOT, c[1] * 255, c[2] * 255, c[3] * 255)
end

local function onTooltipUnit(tip)
	if not viajeOn() then return end
	local _, unit = tip:GetUnit()
	if not unit then return end
	local label = esc(LEVEL or "Nivel")
	local d = difficulty(unit)
	local replacement = tooltipIcon(d) .. " " .. WORD[d]
	for i = 2, tip:NumLines() do
		local line = _G[tip:GetName() .. "TextLeft" .. i]
		local txt = line and line:GetText()
		if txt and txt:find("^" .. label) then
			line:SetText((txt:gsub("^" .. label .. " %S+", replacement, 1)))
			break
		end
	end
end

-- ---------------------------------------------------------------------------
-- Mensajes del chat
-- ---------------------------------------------------------------------------

-- Convierte un texto del juego como «Ganas %d puntos» en un patrón que lo reconoce.
local function pattern(fmt)
	if type(fmt) ~= "string" or fmt == "" then return nil end
	local marked = fmt:gsub("%%%d?%$?[sd]", "\001") -- %d, %s, %1$s...
	local p = esc(marked):gsub("\001", ".+")
	return "^" .. p
end

local XP_MSGS, LEVEL_MSGS = {}, {}
for _, g in ipairs({ "ERR_QUEST_REWARD_EXP_I", "COMBATLOG_XPGAIN_FIRSTPERSON", "COMBATLOG_XPGAIN_FIRSTPERSON_UNNAMED", "COMBATLOG_XPGAIN_EXHAUSTION1", "ERR_ZONE_EXPLORED_XP" }) do
	local p = pattern(_G[g]); if p then XP_MSGS[#XP_MSGS + 1] = p end
end
for _, g in ipairs({ "LEVEL_UP", "LEVEL_UP_HEALTH", "LEVEL_UP_HEALTH_MANA", "LEVEL_UP_STAT", "LEVEL_UP_CHAR_POINTS", "LEVEL_UP_SKILL_POINTS", "LEVEL_UP_SPELL", "LEVEL_UP_ABILITY" }) do
	local p = pattern(_G[g]); if p then LEVEL_MSGS[#LEVEL_MSGS + 1] = p end
end

local function matchAny(list, msg)
	for _, p in ipairs(list) do if msg:find(p) then return true end end
	return false
end

local function chatFilter(_, event, msg)
	if type(msg) ~= "string" then return false end
	if viajeOn() and (event == "CHAT_MSG_COMBAT_XP_GAIN" or matchAny(XP_MSGS, msg)) then return true end
	if subidaOn() and matchAny(LEVEL_MSGS, msg) then return true end
	return false
end

if ChatFrame_AddMessageEventFilter then
	ChatFrame_AddMessageEventFilter("CHAT_MSG_COMBAT_XP_GAIN", chatFilter)
	ChatFrame_AddMessageEventFilter("CHAT_MSG_SYSTEM", chatFilter)
end

-- ---------------------------------------------------------------------------
-- Subida silenciosa
-- ---------------------------------------------------------------------------

local function updateSound()
	if not MuteSoundFile then return end
	if subidaOn() then MuteSoundFile(LEVELUP_SOUND) else UnmuteSoundFile(LEVELUP_SOUND) end
end

-- El botón de talentos parpadea cuando tienes puntos nuevos: eso también delata la subida.
if type(MicroButtonPulse) == "function" then
	hooksecurefunc("MicroButtonPulse", function(button)
		if subidaOn() and button == _G.TalentMicroButton and MicroButtonPulseStop then MicroButtonPulseStop(button) end
	end)
end

-- ---------------------------------------------------------------------------
-- Eventos
-- ---------------------------------------------------------------------------

local function applyAll()
	updateXPBar(); updatePlayer(); updateTarget(); updateAllPlates(); updatePaperDoll(); updateSound()
end

local f = CreateFrame("Frame")
f:RegisterEvent("PLAYER_LOGIN")
f:RegisterEvent("PLAYER_ENTERING_WORLD")
f:RegisterEvent("PLAYER_TARGET_CHANGED")
f:RegisterEvent("UNIT_LEVEL")
f:RegisterEvent("PLAYER_LEVEL_UP")
f:RegisterEvent("NAME_PLATE_UNIT_ADDED")
pcall(f.RegisterEvent, f, "UNIT_CLASSIFICATION_CHANGED")
f:SetScript("OnEvent", function(_, event, arg1)
	if event == "PLAYER_LOGIN" then
		CronicaDB = CronicaDB or {}
		CronicaDB.config = CronicaDB.config or {}
		cfg = CronicaDB.config
		if cfg.viaje == nil then cfg.viaje = false end
		if cfg.subida == nil then cfg.subida = false end
		if GameTooltip and GameTooltip.HookScript and GameTooltip:HasScript("OnTooltipSetUnit") then
			GameTooltip:HookScript("OnTooltipSetUnit", onTooltipUnit)
		elseif TooltipDataProcessor and Enum and Enum.TooltipDataType then
			TooltipDataProcessor.AddTooltipPostCall(Enum.TooltipDataType.Unit, onTooltipUnit)
		end
		if type(PaperDollFrame_SetLevel) == "function" then hooksecurefunc("PaperDollFrame_SetLevel", updatePaperDoll) end
		if type(TargetFrame_CheckLevel) == "function" then hooksecurefunc("TargetFrame_CheckLevel", updateTarget) end
		if type(PlayerFrame_UpdateLevel) == "function" then hooksecurefunc("PlayerFrame_UpdateLevel", updatePlayer) end
		applyAll()
	elseif event == "PLAYER_ENTERING_WORLD" then
		C_Timer.After(1, applyAll)
	elseif event == "NAME_PLATE_UNIT_ADDED" then
		updatePlate(arg1)
	elseif event == "PLAYER_LEVEL_UP" then
		-- Todo cambia de color cuando subes: refrescar después de que el juego actualice tu nivel.
		C_Timer.After(0.5, applyAll)
	else
		updateTarget()
		if event == "UNIT_LEVEL" and arg1 and arg1:find("^nameplate") then updatePlate(arg1) end
	end
end)

-- ---------------------------------------------------------------------------
-- Comandos: se añaden a /cronica
-- ---------------------------------------------------------------------------

local function parseSwitch(arg, current)
	if arg == "on" or arg == "si" or arg == "sí" then return true end
	if arg == "off" or arg == "no" then return false end
	return not current
end

local previous = SlashCmdList.CRONICA
SlashCmdList.CRONICA = function(msg)
	local cmd, arg = (msg or ""):lower():match("^(%S*)%s*(.-)$")
	if cmd == "viaje" and cfg then
		cfg.viaje = parseSwitch(arg, cfg.viaje)
		applyAll()
		if cfg.viaje then
			say("modo viaje activado: sin barra de experiencia y con colores en vez de niveles.")
		else
			say("modo viaje desactivado. Escribe /reload para que todo vuelva a ser como antes.")
		end
		return
	elseif cmd == "subida" and cfg then
		cfg.subida = parseSwitch(arg, cfg.subida)
		updateSound()
		if cfg.subida then
			say("subida silenciosa activada: sin sonido ni mensajes al subir de nivel. La luz dorada es del juego y no se puede quitar.")
		else
			say("subida silenciosa desactivada.")
		end
		return
	end
	if previous then previous(msg) end
	if cmd == "" and cfg then
		say(("viaje: %s · subida silenciosa: %s · /cronica viaje · /cronica subida"):format(cfg.viaje and "sí" or "no", cfg.subida and "sí" or "no"))
	end
end
