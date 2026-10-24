#!/usr/bin/env bash
# Compara, só com SQL, o resultado de um lote gravado no banco com o gabarito oficial.
# Uso: make conferir-gabarito LOTE=<lote_id>   (ou scripts/conferir-gabarito-sql.sh <lote_id> [gabarito.json])
set -euo pipefail
LOTE=${1:?informe o lote_id}
GAB=${2:-data/resultado_esperado_100k.json}
{
  echo "CREATE TEMP TABLE gab (j jsonb); COPY gab FROM STDIN;"
  grep '^{' "$GAB" | sed 's/,$//'
  echo '\.'
  cat <<SQL
WITH g AS (
  SELECT (j->>'linha')::int AS linha, j->>'id' AS id, j->>'status' AS status,
         ARRAY(SELECT jsonb_array_elements_text(j->'barrado_por')) AS barrado_por
  FROM gab
), cmp AS (
  SELECT g.*, r.status AS nosso_status, r.barrado_por AS nosso_barrado_por, r.id AS nosso_id
  FROM g LEFT JOIN resultado_gabarito r ON r.lote_id = '$LOTE' AND r.linha = g.linha
)
SELECT count(*)                                                                   AS linhas_gabarito,
       count(nosso_status)                                                        AS linhas_no_banco,
       count(*) FILTER (WHERE status = nosso_status AND barrado_por = nosso_barrado_por AND id = nosso_id) AS identicas,
       count(*) FILTER (WHERE NOT (status = nosso_status AND barrado_por = nosso_barrado_por)
                          AND array_remove(barrado_por, 'SITUACAO_CADASTRAL_SACADO')
                            = array_remove(nosso_barrado_por, 'SITUACAO_CADASTRAL_SACADO'))       AS so_situacao_difere,
       count(*) FILTER (WHERE array_remove(barrado_por, 'SITUACAO_CADASTRAL_SACADO')
                          IS DISTINCT FROM array_remove(nosso_barrado_por, 'SITUACAO_CADASTRAL_SACADO')) AS outras_diferencas
FROM cmp;
SELECT 'gabarito' AS fonte, status, count(*) FROM gab_status GROUP BY 1, 2
UNION ALL
SELECT 'banco', status, count(*) FROM resultado_gabarito WHERE lote_id = '$LOTE' GROUP BY 1, 2
ORDER BY 1, 2;
SQL
} | sed 's/FROM gab_status/FROM (SELECT j->>'"'"'status'"'"' AS status FROM gab) s/' \
  | docker compose exec -T postgres psql -U antecipe -d antecipe -v ON_ERROR_STOP=1 -q
