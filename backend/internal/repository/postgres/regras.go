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
)

const travaRegras = 424244

// Publicar grava uma nova versão (número = maior + 1), sob trava para não haver duas versões iguais.
func (r *Repo) Publicar(ctx context.Context, reg rules.Regras, autor string, resumo []string) (domain.VersaoRegras, error) {
	js, err := json.Marshal(reg)
	if err != nil {
		return domain.VersaoRegras{}, err
	}
	rs, _ := json.Marshal(resumo)
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return domain.VersaoRegras{}, err
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, travaRegras); err != nil {
		return domain.VersaoRegras{}, err
	}
	v := domain.VersaoRegras{Regras: reg, Autor: autor, Resumo: resumo}
	err = tx.QueryRow(ctx, `
		INSERT INTO regras_versoes (versao, regras, autor, resumo)
		SELECT COALESCE(max(versao), 0) + 1, $1, $2, $3 FROM regras_versoes
		RETURNING versao, publicada_em`, js, autor, rs).Scan(&v.Numero, &v.PublicadaEm)
	if err != nil {
		return v, err
	}
	return v, tx.Commit(ctx)
}

// Listar devolve todas as versões, da mais nova para a mais antiga.
func (r *Repo) Listar(ctx context.Context) ([]domain.VersaoRegras, error) {
	rows, err := r.Pool.Query(ctx, `SELECT `+colunasVersao+` FROM regras_versoes ORDER BY versao DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.VersaoRegras
	for rows.Next() {
		v, err := lerVersao(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// NotasOriginais reconstrói as notas lidas de um lote (para a prévia rodar o motor de novo).
func (r *Repo) NotasOriginais(ctx context.Context, id string) ([]domain.Nota, error) {
	if !uuidValido(id) {
		return nil, errLoteNaoEncontrado
	}
	rows, err := r.Pool.Query(ctx, `
		SELECT linha, nome_fundo, doc_fundo, nome_cedente, doc_cedente, nome_sacado, doc_sacado, numero, tipo,
		       valor_centavos, vencimento, aquisicao, erro_formato
		FROM lote_notas WHERE lote_id = $1 ORDER BY linha`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Nota
	for rows.Next() {
		var n domain.Nota
		var valor *int64
		var venc, aq *time.Time
		var erro []byte
		var nf, df, nc, dc, ns, ds, num, tipo *string
		if err := rows.Scan(&n.Linha, &nf, &df, &nc, &dc, &ns, &ds, &num, &tipo, &valor, &venc, &aq, &erro); err != nil {
			return nil, err
		}
		s := func(p *string) string {
			if p == nil {
				return ""
			}
			return *p
		}
		n.NomeFundo, n.DocFundo, n.NomeCedente, n.DocCedente = s(nf), s(df), s(nc), s(dc)
		n.NomeSacado, n.DocSacado, n.Numero, n.Tipo = s(ns), s(ds), s(num), s(tipo)
		if valor != nil {
			n.ValorCentavos = *valor
		}
		if venc != nil {
			n.Vencimento = *venc
		}
		if aq != nil {
			n.Aquisicao = *aq
		}
		if erro != nil {
			n.ErroFormato = &domain.ErroCampo{}
			_ = json.Unmarshal(erro, n.ErroFormato)
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// UltimoLoteConcluido devolve o id do lote concluído mais recente.
func (r *Repo) UltimoLoteConcluido(ctx context.Context) (string, error) {
	var id string
	err := r.Pool.QueryRow(ctx, `SELECT id FROM lotes WHERE status = 'concluido' ORDER BY concluido_em DESC LIMIT 1`).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", apperrors.Conflict("SEM_LOTE", "Envie um lote para ver o impacto.")
	}
	return id, err
}
