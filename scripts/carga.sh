#!/usr/bin/env bash
# Mede o processamento de lotes de 100 mil, 250 mil e 500 mil linhas pela API.
# Uso: make carga   (com a aplicação no ar via make up)
set -euo pipefail
API=${API:-http://localhost:8080}
GERAR=${GERAR:-docker compose run --rm --no-deps -T api antecipe gerar}
TAMANHOS=${TAMANHOS:-"100000 250000 500000"}
mkdir -p amostras

printf "\n%-10s %-10s %-12s %-12s %-12s\n" "linhas" "MB" "duração" "aprovadas" "recusadas"
for n in $TAMANHOS; do
  arq="amostras/lote_$n.csv"
  [ -s "$arq" ] || $GERAR --linhas "$n" --semente "$n" --estoque data/estoque_antecipe.zip > "$arq"
  id=$(curl -fsS -F "arquivo=@$arq" "$API/api/lotes" | grep -o '"lote_id":"[^"]*"' | cut -d'"' -f4)
  until json=$(curl -fsS "$API/api/lotes/$id") && echo "$json" | grep -qE '"status":"(concluido|erro)"'; do sleep 0.3; done
  ms=$(echo "$json" | grep -o '"duracao_ms":[0-9]*' | cut -d: -f2)
  apr=$(echo "$json" | grep -o '"aprovadas":[0-9]*' | head -1 | cut -d: -f2)
  rec=$(echo "$json" | grep -o '"recusadas":[0-9]*' | head -1 | cut -d: -f2)
  mb=$(du -m "$arq" | cut -f1)
  printf "%-10s %-10s %-12s %-12s %-12s\n" "$n" "$mb" "${ms} ms" "$apr" "$rec"
done
echo
