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
	"antecipe/internal/repository/filesystem"
)

var (
	_ domain.EstoqueRepository  = (*Repo)(nil)
	_ domain.SituacaoRepository = (*Repo)(nil)
	_ domain.RegrasRepository   = (*Repo)(nil)
)

// ContextoMotor lê a carteira (estoque) uma vez e monta: as notas vendidas pelo portal (cessões),
// a carteira de cada fundo e as notas por chave. ignorarCessao deixa de fora as notas de uma cessão.
func (r *Repo) ContextoMotor(ctx context.Context, ignorarCessao string) (domain.ContextoEstoque, error) {
	c := domain.ContextoEstoque{Vendidas: map[string]domain.Vendida{}, Estoques: map[string]domain.EstoqueFundo{}, Itens: make(map[string]domain.ItemEstoque, 131072)}
	var ignorar any
	if uuidValido(ignorarCessao) {
		ignorar = ignorarCessao
	}
	rows, err := r.Pool.Query(ctx, `
		SELECT chave, fundo_doc, sacado_doc, cedente_doc, valor_centavos, aquisicao, origem
		FROM estoque WHERE ($1::uuid IS NULL OR cessao_id IS DISTINCT FROM $1::uuid)`, ignorar)
	if err != nil {
		return c, err
	}
	defer rows.Close()
	for rows.Next() {
		var chave, origem string
		var it domain.ItemEstoque
		var aq *time.Time
		if err := rows.Scan(&chave, &it.FundoDoc, &it.SacadoDoc, &it.CedenteDoc, &it.ValorCentavos, &aq, &origem); err != nil {
			return c, err
		}
		filesystem.AcumularEstoque(&c, chave, it)
		if origem == "cessao" {
			v := domain.Vendida{ValorCentavos: it.ValorCentavos}
			if aq != nil {
				v.Em = *aq
			}
			c.Vendidas[chave] = v
		}
	}
	return c, rows.Err()
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
