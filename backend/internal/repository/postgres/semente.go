package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"

	"antecipe/internal/domain"
	"antecipe/internal/domain/rules"
	"antecipe/internal/platform/document"
)

// Semente são os dados iniciais: o estoque do desafio, a versão 1 das regras e a situação cadastral simulada.
type Semente struct {
	Estoque   []domain.Nota
	Regras    rules.Regras
	Situacoes map[string]string
}

// SemearSeVazio preenche as tabelas vazias. carregar só é chamada se faltar alguma coisa.
func (r *Repo) SemearSeVazio(ctx context.Context, carregar func() (Semente, error)) error {
	conn, err := r.Pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", travaSemente); err != nil {
		return err
	}
	defer conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", travaSemente) //nolint:errcheck

	var temEstoque, temRegras, temSituacao bool
	err = conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM estoque), EXISTS(SELECT 1 FROM regras_versoes), EXISTS(SELECT 1 FROM situacao_cadastral)`).
		Scan(&temEstoque, &temRegras, &temSituacao)
	if err != nil {
		return err
	}
	if temEstoque && temRegras && temSituacao {
		return nil
	}
	s, err := carregar()
	if err != nil {
		return fmt.Errorf("semente: %w", err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck

	if !temEstoque {
		linhas := make([][]any, 0, len(s.Estoque))
		for i := range s.Estoque {
			n := &s.Estoque[i]
			if n.ErroFormato != nil {
				continue
			}
			linhas = append(linhas, []any{domain.Chave(n), document.Normalizar(n.DocFundo), n.NomeFundo,
				document.Normalizar(n.DocCedente), n.NomeCedente, document.Normalizar(n.DocSacado), n.NomeSacado,
				n.Numero, n.Tipo, n.ValorCentavos, n.Vencimento, dataOuNulo(n.Aquisicao), "seed"})
		}
		_, err := tx.CopyFrom(ctx, pgx.Identifier{"estoque"},
			[]string{"chave", "fundo_doc", "fundo_nome", "cedente_doc", "cedente_nome", "sacado_doc", "sacado_nome", "numero", "tipo", "valor_centavos", "vencimento", "aquisicao", "origem"},
			pgx.CopyFromRows(linhas))
		if err != nil {
			return fmt.Errorf("semente do estoque: %w", err)
		}
	}
	if !temRegras {
		js, err := json.Marshal(s.Regras)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO regras_versoes (versao, regras, autor, resumo) VALUES (1, $1, 'semente', '["Versão inicial"]')`, js); err != nil {
			return fmt.Errorf("semente das regras: %w", err)
		}
	}
	if !temSituacao {
		linhas := make([][]any, 0, len(s.Situacoes))
		for cnpj, sit := range s.Situacoes {
			linhas = append(linhas, []any{cnpj, sit})
		}
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"situacao_cadastral"}, []string{"cnpj", "situacao"}, pgx.CopyFromRows(linhas)); err != nil {
			return fmt.Errorf("semente das situações: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// AtualizarSituacoes troca a situação cadastral inteira pela do CSV (roda a cada subida: a fonte oficial manda).
func (r *Repo) AtualizarSituacoes(ctx context.Context, m map[string]string) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck
	if _, err := tx.Exec(ctx, `DELETE FROM situacao_cadastral`); err != nil {
		return err
	}
	linhas := make([][]any, 0, len(m))
	for cnpj, sit := range m {
		linhas = append(linhas, []any{cnpj, sit})
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"situacao_cadastral"}, []string{"cnpj", "situacao"}, pgx.CopyFromRows(linhas)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
