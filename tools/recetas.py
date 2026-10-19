#!/usr/bin/env python3
"""Genera docs/oficios/recetas.json (Herrería y Minería de Classic) a partir del
paquete npm «wow-classic-items» (MIT, datos de Wowhead y la API de Blizzard).

Uso: python3 tools/recetas.py <ruta a wow-classic-items/data/json/data.json>

Los cuatro umbrales de color (naranja/amarillo/verde/gris) no vienen en los
datos: se estiman (amarillo = nivel + 20, verde + 40, gris + 60) y el addon los
corrige con lo que ve en el juego.
"""
import json, sys

PROFS = {"Blacksmithing": "Herrería", "Mining": "Minería"}
MAX_SKILL = 300          # Classic
TBC_FROM = 23400         # los objetos de Burning Crusade empiezan por aquí
TBC_IDS = set(range(22572, 22579))  # «Mota de…» (Burning Crusade)

src = json.load(open(sys.argv[1]))
items = {it["itemId"]: it for it in src}
QUALITY = {"Poor": 0, "Common": 1, "Uncommon": 2, "Rare": 3, "Epic": 4, "Legendary": 5}

recipes, used = [], set()
for it in src:
    for cb in it.get("createdBy") or []:
        prof = cb.get("category")
        if prof not in PROFS:
            continue
        req = cb.get("requiredSkill") or 1
        regs = cb.get("reagents") or []
        ids = [it["itemId"]] + [r["itemId"] for r in regs] + list(cb.get("recipes") or [])
        if req > MAX_SKILL or any(i >= TBC_FROM or i in TBC_IDS for i in ids):
            continue
        learn = []
        for rid in cb.get("recipes") or []:
            ri = items.get(rid)
            if not ri:
                continue
            s = ri.get("source") or {}
            learn.append({k: v for k, v in {
                "id": rid, "fuente": s.get("category"), "precio": s.get("cost") or ri.get("vendorPrice"),
                "prob": s.get("dropChance"), "faccion": s.get("faction"),
            }.items() if v is not None})
        amt = cb.get("amount") or [1, 1]
        recipes.append({
            "id": it["itemId"], "oficio": PROFS[prof], "nivel": req,
            "colores": [req, req + 20, req + 40, req + 60],
            "hace": (amt[0] + amt[1]) / 2,
            "mats": [{"id": r["itemId"], "n": r["amount"]} for r in regs],
            "aprender": learn or "instructor",
        })
        used.update(ids)

recipes.sort(key=lambda r: (r["oficio"], r["nivel"], r["id"]))
out_items = {}
for i in sorted(used):
    it = items.get(i)
    if not it:
        continue
    out_items[str(i)] = {k: v for k, v in {
        "en": it["name"], "q": QUALITY.get(it.get("quality"), 1),
        "venta": it.get("sellPrice") or None, "compra": it.get("vendorPrice") or None,
        "tipo": it.get("subclass"),
    }.items() if v is not None}
json.dump({"fuente": "wow-classic-items 2.0.1 (MIT), datos de Wowhead; colores estimados",
           "recetas": recipes, "objetos": out_items},
          open("docs/oficios/recetas.json", "w"), ensure_ascii=False, separators=(",", ":"))
print(len(recipes), "recetas,", len(out_items), "objetos")
