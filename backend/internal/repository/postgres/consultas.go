package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"antecipe/internal/domain"
	"antecipe/internal/domain/rules"
	"antecipe/internal/platform/apperrors"
	"antecipe/internal/platform/document"
)

var (
	_ domain.EstoqueRepository  = (*Repo)(nil)
	_ domain.SituacaoRepository = (*Repo)(nil)
	_ domain.RegrasRepository   = (*Repo)(nil)
)

// ContextoMotor carrega as chaves já vendidas (todos os fundos) e as somas do fundo, em consultas agrupadas.
func (r *Repo) ContextoMotor(ctx context.Context, fundoCNPJ string) (map[string]domain.Vendida, domain.EstoqueFundo, error) {
	est := domain.EstoqueFundo{PorSacado: map[string]int64{}, PorCedente: map[string]int64{}}
	rows, err := r.Pool.Query(ctx, `SELECT chave, aquisicao, valor_centavos FROM estoque`)
	if err != nil {
		return nil, est, err
	}
	vendidas := make(map[string]domain.Vendida, 131072)
	for rows.Next() {
		var chave string
		var aq *time.Time
		var valor int64
		if err := rows.Scan(&chave, &aq, &valor); err != nil {
			rows.Close()
			return nil, est, err
		}
		v := domain.Vendida{ValorCentavos: valor}
		if aq != nil {
			v.Em = *aq
		}
		vendidas[chave] = v
	}
	if err := rows.Err(); err != nil {
		return nil, est, err
	}

	fundo := document.Normalizar(fundoCNPJ)
	somar := func(coluna string, destino map[string]int64) error {
		rows, err := r.Pool.Query(ctx, `SELECT `+coluna+`, sum(valor_centavos)::bigint FROM estoque WHERE fundo_doc = $1 GROUP BY 1`, fundo)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var doc string
			var soma int64
			if err := rows.Scan(&doc, &soma); err != nil {
				return err
			}
			destino[doc] = soma
		}
		return rows.Err()
	}
	if err := somar("sacado_doc", est.PorSacado); err != nil {
		return nil, est, err
	}
	if err := somar("cedente_doc", est.PorCedente); err != nil {
		return nil, est, err
	}
	for _, v := range est.PorSacado {
		est.TotalCentavos += v
	}
	return vendidas, est, nil
}

// Situacoes devolve a situação cadastral simulada (CNPJ normalizado → situação).
func (r *Repo) Situacoes(ctx context.Context) (map[string]string, error) {
	rows, err := r.Pool.Query(ctx, `SELECT cnpj, situacao FROM situacao_cadastral`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]string{}
	for rows.Next() {
		var c, s string
		if err := rows.Scan(&c, &s); err != nil {
			return nil, err
		}
		m[c] = s
	}
	return m, rows.Err()
}

const colunasVersao = `versao, regras, autor, resumo, publicada_em`

// Publicadas devolve a versão mais recente das regras.
func (r *Repo) Publicadas(ctx context.Context) (domain.VersaoRegras, error) {
	return lerVersao(r.Pool.QueryRow(ctx, `SELECT `+colunasVersao+` FROM regras_versoes ORDER BY versao DESC LIMIT 1`))
}

// Obter devolve uma versão específica.
func (r *Repo) Obter(ctx context.Context, numero int) (domain.VersaoRegras, error) {
	return lerVersao(r.Pool.QueryRow(ctx, `SELECT `+colunasVersao+` FROM regras_versoes WHERE versao = $1`, numero))
}

func lerVersao(row pgx.Row) (domain.VersaoRegras, error) {
	var v domain.VersaoRegras
	var regrasJS, resumoJS []byte
	if err := row.Scan(&v.Numero, &regrasJS, &v.Autor, &resumoJS, &v.PublicadaEm); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return v, apperrors.NotFound("VERSAO_NAO_ENCONTRADA", "Versão das regras não encontrada.")
		}
		return v, err
	}
	reg, err := rules.Ler(regrasJS)
	if err != nil {
		return v, err
	}
	v.Regras = reg
	_ = json.Unmarshal(resumoJS, &v.Resumo)
	return v, nil
}

func dataOuNulo(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
