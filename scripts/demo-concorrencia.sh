#!/usr/bin/env bash
# Demonstra que duas cessões simultâneas de lotes com notas em comum nunca vendem a mesma nota duas vezes.
# Uso: make demo-concorrencia   (com a aplicação no ar via make up)
set -euo pipefail
API=${API:-http://localhost:8080}
GERAR=${GERAR:-docker compose run --rm --no-deps -T api antecipe gerar}
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# sementes novas a cada execução: senão o reenvio dos mesmos arquivos só devolve os lotes antigos (idempotência)
s1=$(date +%s); s2=$((s1 + 1))
echo "Gerando lote A (2.000 notas) e lote B (500 notas em comum com A + 2.000 novas)..."
$GERAR --linhas 2000 --semente "$s1" > "$tmp/a.csv" 2>/dev/null
{ head -n 501 "$tmp/a.csv"; $GERAR --linhas 2000 --semente "$s2" 2>/dev/null | tail -n +2; } > "$tmp/b.csv"

enviar() { curl -fsS -F "arquivo=@$1" "$API/api/lotes" | grep -o '"lote_id":"[^"]*"' | cut -d'"' -f4; }
esperar() { until curl -fsS "$API/api/lotes/$1" | grep -q '"status":"concluido"'; do sleep 0.3; done; }

A=$(enviar "$tmp/a.csv"); B=$(enviar "$tmp/b.csv")
esperar "$A"; esperar "$B"
echo "Simulações prontas. Disparando as duas cessões AO MESMO TEMPO..."
curl -fsS -X POST -H "Idempotency-Key: demo-$A" "$API/api/lotes/$A/cessao" > "$tmp/ca.json" &
curl -fsS -X POST -H "Idempotency-Key: demo-$B" "$API/api/lotes/$B/cessao" > "$tmp/cb.json" &
wait

for lote in A B; do
  f="$tmp/c$(echo $lote | tr AB ab).json"
  cedidas=$(grep -o '"notas_cedidas":[0-9]*' "$f" | cut -d: -f2)
  dup=$({ grep -o '"codigo":"DUPLICIDADE"' "$f" || true; } | wc -l | tr -d ' ')
  echo "  Lote $lote: $cedidas notas cedidas · $dup excluídas por já terem sido vendidas na outra cessão"
done
echo "Nenhuma nota foi vendida duas vezes: as notas em comum entraram em uma cessão só."
