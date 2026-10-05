"""Simula una sesión de juego con el addon Crónica, sin WoW.

Carga el addon en Lua (con lupa) sobre una imitación mínima de la API del juego,
dispara los eventos de una sesión (misiones, nivel, equipo, caída…) y escribe
CronicaDB en el mismo formato que el juego (SavedVariables/Cronica.lua).

Uso: python3 addon/prueba/simular.py salida.lua
"""
import os
import sys

from lupa import LuaRuntime

HERE = os.path.dirname(os.path.abspath(__file__))
ADDON = os.path.join(HERE, "..", "Cronica")

STUB = r"""
local function newobj()
  local o = {scripts = {}, shown = false, text = ""}
  return setmetatable(o, {__index = function(t, k)
    if k == "SetScript" then return function(self, n, f) self.scripts[n] = f end end
    if k == "GetScript" then return function(self, n) return self.scripts[n] end end
    if k == "CreateFontString" or k == "CreateTexture" then return function() return newobj() end end
    if k == "Show" then return function(self) self.shown = true end end
    if k == "Hide" then return function(self) self.shown = false end end
    if k == "IsShown" then return function(self) return self.shown end end
    if k == "SetText" then return function(self, s) self.text = s end end
    if k == "GetText" then return function(self) return self.text end end
    if k == "RegisterEvent" then return function(self, e) local ev = rawget(self, "events") or {}; rawset(self, "events", ev); ev[e] = true end end
    return function() end
  end})
end
FRAMES = {}
function CreateFrame(kind, name) local f = newobj(); f.name = name; table.insert(FRAMES, f); if name then _G[name] = f end; return f end
UIParent = newobj(); QuestFrame = newobj(); QuestFrameAcceptButton = newobj()
DEFAULT_CHAT_FRAME = { lines = {}, AddMessage = function(self, m) table.insert(self.lines, m) end }
CHAT = DEFAULT_CHAT_FRAME
NOW = os.time() - 10800; CLOCK = 5000
function time() return NOW end
function GetTime() return CLOCK end
TIMERS = {}
C_Timer = { After = function(s, fn) table.insert(TIMERS, {at = CLOCK + s, fn = fn}) end }
function ADVANCE(sec)
  CLOCK = CLOCK + sec; NOW = NOW + sec
  local again = true
  while again do
    again = false
    for i, t in ipairs(TIMERS) do
      if t.at <= CLOCK then table.remove(TIMERS, i); t.fn(); again = true; break end
    end
  end
end
PLAYER = { name = "Nehonar", level = 7, zone = "Bosque de Elwynn", sub = "Villadorada", combat = false }
function UnitName(u) if u == "player" then return PLAYER.name end if u == "npc" or u == "questnpc" then return NPC end end
function UnitLevel() return PLAYER.level end
function UnitRace() return "Humano", "Human" end
function UnitClass() return "Paladín", "PALADIN" end
function GetRealmName() return "Forever Beta" end
function GetNormalizedRealmName() return "ForeverBeta" end
function GetRealZoneText() return PLAYER.zone end
function GetSubZoneText() return PLAYER.sub end
function InCombatLockdown() return PLAYER.combat end
function UnitAffectingCombat() return PLAYER.combat end
function IsResting() return true end
QUEST = {}
function GetQuestID() return QUEST.id or 0 end
function GetTitleText() return QUEST.title end
function GetQuestText() return QUEST.text end
function GetObjectiveText() return QUEST.objectives end
function GetRewardText() return QUEST.reward end
ACCEPTED = {}
function AcceptQuest() table.insert(ACCEPTED, QUEST.id) end
RELOADS = 0
function ReloadUI() RELOADS = RELOADS + 1 end
SAID = {}
function SendChatMessage(msg, chan) table.insert(SAID, chan .. ":" .. msg) end
BINDS = {}
function GetBindingKey(n) return BINDS[n] end
C_QuestLog = { GetTitleForQuestID = function(id) return QUESTS_BY_ID[id] end }
QUESTS_BY_ID = {}
EQUIP = {}
ITEMS = {}
function GetInventoryItemLink(u, slot) return EQUIP[slot] end
function GetItemInfo(link) local it = ITEMS[link]; if not it then return nil end; return it.name, link, it.quality, 10, 1, it.type, it.sub end
C_Item = { GetItemStats = function(link) return ITEMS[link] and ITEMS[link].stats or {} end }
SlashCmdList = {}
math.random = function(n) if n then return 1 end return 0 end  -- siempre «sí», sin azar
"""

SERIALIZE = r"""
function SERIALIZE(v, indent)
  indent = indent or ""
  local t = type(v)
  if t == "string" then return string.format("%q", v):gsub("\\\n", "\\n") end
  if t == "number" or t == "boolean" then return tostring(v) end
  if t ~= "table" then return "nil" end
  local out = {"{"}
  local n = #v
  local keys = {}
  for k in pairs(v) do if not (type(k) == "number" and k >= 1 and k <= n and k % 1 == 0) then table.insert(keys, k) end end
  table.sort(keys, function(a, b) return tostring(a) < tostring(b) end)
  for _, k in ipairs(keys) do
    local ks = type(k) == "string" and string.format("[%q]", k) or ("[" .. tostring(k) .. "]")
    table.insert(out, indent .. "\t" .. ks .. " = " .. SERIALIZE(v[k], indent .. "\t") .. ",")
  end
  for i = 1, n do table.insert(out, indent .. "\t" .. SERIALIZE(v[i], indent .. "\t") .. ", -- [" .. i .. "]") end
  table.insert(out, indent .. "}")
  return table.concat(out, "\n")
end
"""


def main(out_path):
    lua = LuaRuntime(unpack_returned_tuples=True)
    lua.execute(STUB)
    lua.execute(SERIALIZE)
    for name in ("CronicaTextos.lua", "Cronica.lua"):
        with open(os.path.join(ADDON, name), encoding="utf-8") as f:
            src = f.read()
        lua.execute(src if name != "Cronica.lua" else "return (function(...)\n" + src + "\nend)('Cronica')")
    G = lua.globals()
    ev = lua.eval("function(name, ...) for _, f in ipairs(FRAMES) do local e = rawget(f, 'events'); if e and e[name] and f.scripts.OnEvent then f.scripts.OnEvent(f, name, ...) end end end")
    adv = G.ADVANCE

    def quest(qid, title, npc, text, objectives):
        lua.execute(f"QUESTS_BY_ID[{qid}] = {lua_str(title)}; NPC = {lua_str(npc)}")
        lua.execute(f"QUEST = {{id={qid}, title={lua_str(title)}, text={lua_str(text)}, objectives={lua_str(objectives)}}}")
        ev("QUEST_DETAIL")

    def complete(qid, npc, reward):
        lua.execute(f"NPC = {lua_str(npc)}; QUEST = {{id={qid}, reward={lua_str(reward)}}}")
        ev("QUEST_COMPLETE")
        ev("QUEST_TURNED_IN", qid)
        ev("QUEST_REMOVED", qid)
        ev("QUEST_FINISHED")

    ev("ADDON_LOADED", "Cronica")
    ev("PLAYER_LOGIN")
    ev("PLAYER_ENTERING_WORLD", True, False)
    adv(10)

    # 1) Misión aceptada con el botón «Aceptar y leer en Crónica».
    quest(106, "La joven enamorada", "Maybell Maclure",
          "Mi familia y los Stonefield no se hablan, pero yo quiero a Tommy Joe Stonefield. ¿Le llevarías esta carta? Está junto al río, al sur de la granja de su familia.",
          "Lleva la carta de Maybell a Tommy Joe Stonefield.")
    btn = G.CronicaLeerBoton
    assert btn.shown, "el botón debe aparecer en la ventana de misión"
    btn.scripts.OnClick(btn)
    adv(1)
    assert G.RELOADS == 1, "el botón debe recargar para enviar la misión"
    ev("QUEST_ACCEPTED", 106)  # el servidor confirma: no debe duplicarse
    adv(300)

    # 2) Cadena: entregar a Tommy Joe, que te da la siguiente.
    complete(106, "Tommy Joe Stonefield", "¿Una carta de Maybell? ¡Cielos! Gracias, de verdad.")
    adv(20)
    quest(111, "Hablar con Maybell", "Tommy Joe Stonefield",
          "Tengo que darle una respuesta a Maybell, pero si su familia me ve, se acabó. Llévale mi colgante, ella sabrá lo que significa.",
          "Lleva el colgante de Tommy Joe a Maybell Maclure.")
    ev("QUEST_ACCEPTED", 111)
    lua.execute("PLAYER.level = 8")
    ev("PLAYER_LEVEL_UP", 8)
    adv(400)

    # 3) Equipo azul con atributos, y caída en combate (la frase espera al final del combate).
    lua.execute("""
      EQUIP[16] = "|Hitem:2079:::|h[Espada del viñador]|h"
      ITEMS[EQUIP[16]] = {name="Espada del viñador", quality=3, type="Arma", sub="Espadas de una mano",
        stats={ITEM_MOD_STRENGTH_SHORT=4, ITEM_MOD_STAMINA_SHORT=3, RESISTANCE0_NAME=0}}
    """)
    ev("PLAYER_EQUIPMENT_CHANGED", 16, False)
    adv(5)
    lua.execute("PLAYER.combat = true")
    ev("PLAYER_DEAD")
    lua.execute("PLAYER.combat = false")
    adv(1)

    # 4) Modo «voz»: la frase espera a una tecla y se dice en /decir.
    G.SlashCmdList.CRONICA("frases voz")
    lua.execute('BINDS.CRONICA_DECIR = "F9"')
    G.Cronica_Frase("descanso", None, True)
    assert G.CronicaSubtitulo.shown
    G.Cronica_Decir()
    assert list(G.SAID.values())[0].startswith("SAY:"), "debe decirlo en /decir"
    # Misión abandonada.
    quest(120, "Pañuelos rojos", "Alguacil Dughan", "x", "y")
    ev("QUEST_ACCEPTED", 120)
    ev("QUEST_REMOVED", 120)
    adv(2)

    # 5) Entregar la segunda misión de la cadena y salir.
    adv(600)
    complete(111, "Maybell Maclure", "¡Su colgante! Gracias… nadie debe saberlo.")
    adv(200)

    # 6) Última misión, aceptada con el botón y aún sin hacer: debe aparecer en «Misiones».
    quest(176, "Se busca: Hogger", "Cartel de «Se busca»",
          "SE BUSCA: un gnoll enorme llamado Hogger aterroriza el oeste del Bosque de Elwynn. La guardia de Ventormenta paga una recompensa por su cabeza. Se le ha visto cerca del Bosque Brumoso, al suroeste. Entregad la prueba al alguacil Dughan en Villadorada.",
          "Trae la garra de Hogger al alguacil Dughan, en Villadorada.")
    btn.scripts.OnClick(btn)
    adv(1)
    ev("PLAYER_LOGOUT")

    db = G.CronicaDB
    ch = db.characters["Nehonar-ForeverBeta"]
    types = [ch.events[i].type for i in range(1, len(ch.events) + 1)]
    print("eventos:", types)
    print("chat:", [G.CHAT.lines[i] for i in range(1, len(G.CHAT.lines) + 1)])
    assert types.count("quest_accept") == 4, "una aceptación por misión, sin duplicados"
    assert "quest_abandon" in types and "equip" in types and "death" in types
    with open(out_path, "w", encoding="utf-8") as f:
        f.write("\nCronicaDB = " + G.SERIALIZE(db) + "\n")
    print("escrito", out_path)


def lua_str(s):
    return '"' + s.replace("\\", "\\\\").replace('"', '\\"') + '"'


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else "Cronica.lua")
