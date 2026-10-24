#!/usr/bin/env bash
# Amplia o estoque oficial em N cópias para teste de carga com resposta conhecida.
# A cópia 1 é idêntica ao original; nas outras, NU_DOCUMENTO ganha o sufixo -C<k> (deixa de ser duplicata).
# Como cada fundo cresce na mesma proporção, a concentração não muda: o resultado esperado da linha
# de cada cópia é o do gabarito para a linha original (scripts/comparar-gabarito.py confere).
#
# Uso: scripts/lote-ampliado.sh [VEZES] > amostras/lote_1m.csv     (padrão: 10 → 1.005.540 linhas)
set -euo pipefail
VEZES=${1:-10}
unzip -p "$(dirname "$0")/../data/estoque_antecipe.zip" estoque_antecipe.csv | awk -F';' -v OFS=';' -v vezes="$VEZES" '
  NR == 1 { print; next }
  { linhas[++n] = $0 }
  END {
    for (k = 0; k < vezes; k++)
      for (i = 1; i <= n; i++) {
        if (k == 0) { print linhas[i]; continue }
        split(linhas[i], c, ";")
        c[7] = c[7] "-C" k
        out = c[1]; for (j = 2; j <= 11; j++) out = out OFS c[j]
        print out
      }
  }'
