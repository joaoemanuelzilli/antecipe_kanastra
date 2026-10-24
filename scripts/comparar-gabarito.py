#!/usr/bin/env python3
"""Compara o resultado.json de um lote com o gabarito oficial (data/resultado_esperado_100k.json).

Funciona com o estoque original e com o lote ampliado por scripts/lote-ampliado.sh: a linha L
corresponde à linha original ((L - 2) % 100554) + 2.

Uso: python3 scripts/comparar-gabarito.py resultado.json [data/resultado_esperado_100k.json]
"""
import json
import sys

nosso = json.load(open(sys.argv[1]))
gabarito = {e["linha"]: e for e in json.load(open(sys.argv[2] if len(sys.argv) > 2 else "data/resultado_esperado_100k.json"))}
n = len(gabarito)
sem = lambda cs: [c for c in cs if c != "SITUACAO_CADASTRAL_SACADO"]
iguais = so_situacao = outras = 0
for item in nosso:
    esp = gabarito[(item["linha"] - 2) % n + 2]
    a, b = item.get("barrado_por", []), esp.get("barrado_por", [])
    if item["status"] == esp["status"] and a == b:
        iguais += 1
    elif sem(a) == sem(b):
        so_situacao += 1
    else:
        outras += 1
        if outras <= 5:
            print(f"  linha {item['linha']}: nosso {item}, gabarito {esp}")
print(f"{len(nosso)} linhas · iguais ao gabarito: {iguais} · só a situação cadastral difere: {so_situacao} · outras diferenças: {outras}")
sys.exit(1 if outras else 0)
