-- Resultado de cada nota no formato do gabarito do desafio (data/resultado_esperado_100k.json):
--   SELECT linha, id, status, barrado_por FROM resultado_gabarito WHERE lote_id = '<id>' ORDER BY linha;
-- barrado_por traz os códigos das regras não cumpridas, na ordem do JSON de regras.
CREATE OR REPLACE VIEW resultado_gabarito AS
SELECT n.lote_id,
       n.linha,
       n.numero AS id,
       CASE n.status WHEN 1 THEN 'APROVADO' ELSE 'BARRADO' END AS status,
       ARRAY(SELECT r.e ->> 'codigo'
             FROM jsonb_array_elements(n.recusas) WITH ORDINALITY AS r(e, i)
             ORDER BY r.i) AS barrado_por
FROM lote_notas n;
