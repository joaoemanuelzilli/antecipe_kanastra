CREATE TABLE IF NOT EXISTS estoque (
  id BIGSERIAL PRIMARY KEY,
  chave TEXT NOT NULL UNIQUE,
  fundo_doc TEXT NOT NULL, fundo_nome TEXT NOT NULL,
  cedente_doc TEXT NOT NULL, cedente_nome TEXT NOT NULL,
  sacado_doc TEXT NOT NULL, sacado_nome TEXT NOT NULL,
  numero TEXT NOT NULL, tipo TEXT NOT NULL,
  valor_centavos BIGINT NOT NULL,
  vencimento DATE NOT NULL, aquisicao DATE,
  origem TEXT NOT NULL CHECK (origem IN ('seed', 'cessao')),
  cessao_id UUID
);
CREATE INDEX IF NOT EXISTS estoque_fundo_sacado ON estoque (fundo_doc, sacado_doc);
CREATE INDEX IF NOT EXISTS estoque_fundo_cedente ON estoque (fundo_doc, cedente_doc);

CREATE TABLE IF NOT EXISTS regras_versoes (
  versao INT PRIMARY KEY,
  regras JSONB NOT NULL,
  autor TEXT NOT NULL,
  resumo JSONB NOT NULL DEFAULT '[]',
  publicada_em TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS situacao_cadastral (
  cnpj TEXT PRIMARY KEY,
  situacao TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS lotes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  idempotency_key TEXT NOT NULL UNIQUE,
  hash_arquivo TEXT NOT NULL,
  data_base DATE NOT NULL,
  regras_versao INT NOT NULL REFERENCES regras_versoes (versao),
  precificacao JSONB NOT NULL,
  status TEXT NOT NULL,
  etapa TEXT,
  linhas_total INT NOT NULL DEFAULT 0,
  linhas_processadas INT NOT NULL DEFAULT 0,
  totais JSONB,
  motivos JSONB,
  erros_configuracao JSONB NOT NULL DEFAULT '[]',
  erro JSONB,
  criado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
  iniciado_em TIMESTAMPTZ,
  concluido_em TIMESTAMPTZ,
  duracao_ms INT
);

CREATE TABLE IF NOT EXISTS lote_notas (
  lote_id UUID NOT NULL REFERENCES lotes (id) ON DELETE CASCADE,
  linha INT NOT NULL,
  nome_fundo TEXT, doc_fundo TEXT, nome_cedente TEXT, doc_cedente TEXT, nome_sacado TEXT, doc_sacado TEXT,
  numero TEXT, tipo TEXT, valor_centavos BIGINT, vencimento DATE, aquisicao DATE,
  chave TEXT,
  status SMALLINT NOT NULL,
  motivo_principal TEXT,
  dias INT,
  desconto_centavos BIGINT,
  liquido_centavos BIGINT,
  recusas JSONB NOT NULL DEFAULT '[]',
  erro_formato JSONB,
  PRIMARY KEY (lote_id, linha)
);
CREATE INDEX IF NOT EXISTS lote_notas_filtro ON lote_notas (lote_id, status, motivo_principal);

CREATE TABLE IF NOT EXISTS cessoes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  lote_id UUID NOT NULL UNIQUE REFERENCES lotes (id),
  idempotency_key TEXT UNIQUE,
  confirmada_em TIMESTAMPTZ NOT NULL DEFAULT now(),
  notas_cedidas INT NOT NULL,
  face_centavos BIGINT NOT NULL,
  desconto_centavos BIGINT NOT NULL,
  liquido_centavos BIGINT NOT NULL,
  notas_excluidas JSONB NOT NULL DEFAULT '[]'
);
