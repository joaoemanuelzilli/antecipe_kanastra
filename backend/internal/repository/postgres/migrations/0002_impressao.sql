-- Impressão digital do envio (hash do arquivo + data-base + versão das regras + preço):
-- impede reaproveitar uma Idempotency-Key com outro arquivo.
ALTER TABLE lotes ADD COLUMN IF NOT EXISTS impressao TEXT NOT NULL DEFAULT '';
