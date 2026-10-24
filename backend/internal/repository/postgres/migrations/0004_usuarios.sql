-- Contas (empresa e gestor) e sessões. A senha é guardada como bcrypt; o token da sessão só como sha256.
CREATE TABLE IF NOT EXISTS usuarios (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL UNIQUE,
  nome TEXT NOT NULL,
  perfil TEXT NOT NULL CHECK (perfil IN ('empresa', 'gestor')),
  empresa_nome TEXT,
  empresa_cnpj TEXT,
  senha_hash TEXT NOT NULL,
  criado_em TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sessoes (
  token_hash TEXT PRIMARY KEY,
  usuario_id UUID NOT NULL REFERENCES usuarios (id) ON DELETE CASCADE,
  criada_em TIMESTAMPTZ NOT NULL DEFAULT now(),
  expira_em TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS sessoes_expira ON sessoes (expira_em);

-- Lote enviado por uma conta logada (NULL = envio anônimo, como antes)
ALTER TABLE lotes ADD COLUMN IF NOT EXISTS usuario_id UUID REFERENCES usuarios (id);
CREATE INDEX IF NOT EXISTS lotes_usuario ON lotes (usuario_id, criado_em DESC);
