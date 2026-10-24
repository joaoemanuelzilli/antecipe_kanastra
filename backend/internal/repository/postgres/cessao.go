package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"antecipe/internal/domain"
	"antecipe/internal/platform/apperrors"
	"antecipe/internal/platform/document"
)

var _ domain.CessaoRepository = (*Repo)(nil)

var (
	errLoteNaoConcluido = apperrors.Conflict("LOTE_NAO_CONCLUIDO", "O lote ainda está sendo processado. Espere terminar para confirmar.")
	errSemAprovadas     = apperrors.Conflict("SEM_APROVADAS", "Nenhuma nota deste lote foi aprovada. Não há o que ceder.")
	errChaveReutilizada = apperrors.Conflict("IDEMPOTENCY_KEY_REUTILIZADA", "Essa Idempotency-Key já foi usada em outra cessão.")
)

const msgVendidaEmParalelo = "Esta nota foi vendida em outra operação enquanto você revisava o lote."

// Ceder vende as aprovadas do lote numa transação:
//  1. trava o lote (FOR UPDATE); se já foi cedido, devolve a mesma cessão (replay);
//  2. trava o fundo (advisory lock): cessões do mesmo fundo ficam em fila, porque a concentração depende do estoque;
//  3. reverifica a concentração sobre o estoque atual (regra de negócio vem do serviço);
//  4. insere no estoque com ON CONFLICT (chave) DO NOTHING: o que não entrou foi vendido por outra requisição.
func (r *Repo) Ceder(ctx context.Context, loteID, chave string, fundo domain.FundoDestino, reverificar domain.Reverificar) (domain.Cessao, []domain.Excluida, bool, error) {
	var c domain.Cessao
	if !uuidValido(loteID) {
		return c, nil, false, errLoteNaoEncontrado
	}
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return c, nil, false, err
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck

	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM lotes WHERE id = $1 FOR UPDATE`, loteID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c, nil, false, errLoteNaoEncontrado
		}
		return c, nil, false, err
	}
	if status != domain.LoteConcluido {
		return c, nil, false, errLoteNaoConcluido
	}
	if existente, exc, ok, err := cessaoDoLote(ctx, tx, loteID); err != nil || ok {
		return existente, exc, ok, err
	}
	// Sem header, a chave fica NULL: o UNIQUE(lote_id) já garante o replay, e o cliente não
	// consegue ocupar uma chave gerada pelo servidor.
	var chaveOuNulo any
	if chave != "" {
		chaveOuNulo = chave
		var outroLote string
		err = tx.QueryRow(ctx, `SELECT lote_id FROM cessoes WHERE idempotency_key = $1`, chave).Scan(&outroLote)
		if err == nil {
			return c, nil, false, errChaveReutilizada
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return c, nil, false, err
		}
	}

	aprovadas, err := aprovadasDoLote(ctx, tx, loteID)
	if err != nil {
		return c, nil, false, err
	}
	if len(aprovadas) == 0 {
		return c, nil, false, errSemAprovadas
	}
	fundoDoc := document.Normalizar(fundo.CNPJ)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, fundoDoc); err != nil {
		return c, nil, false, err
	}
	est, err := estoqueDoFundo(ctx, tx, fundoDoc)
	if err != nil {
		return c, nil, false, err
	}
	excluidas := reverificar(aprovadas, est)
	fora := map[int]bool{}
	for _, e := range excluidas {
		fora[e.Linha] = true
	}

	if err := tx.QueryRow(ctx, `SELECT gen_random_uuid()`).Scan(&c.ID); err != nil {
		return c, nil, false, err
	}
	if _, err := tx.Exec(ctx, `CREATE TEMP TABLE cessao_tmp (LIKE estoque INCLUDING DEFAULTS) ON COMMIT DROP`); err != nil {
		return c, nil, false, err
	}
	hoje := time.Now().UTC().Truncate(24 * time.Hour)
	var linhas [][]any
	porChave := map[string]*domain.Resultado{}
	for i := range aprovadas {
		rs := &aprovadas[i]
		if fora[rs.Nota.Linha] {
			continue
		}
		n := rs.Nota
		porChave[rs.Chave] = rs
		linhas = append(linhas, []any{rs.Chave, fundoDoc, fundo.Nome, document.Normalizar(n.DocCedente), n.NomeCedente,
			document.Normalizar(n.DocSacado), n.NomeSacado, n.Numero, n.Tipo, n.ValorCentavos, n.Vencimento, hoje, "cessao", c.ID})
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"cessao_tmp"},
		[]string{"chave", "fundo_doc", "fundo_nome", "cedente_doc", "cedente_nome", "sacado_doc", "sacado_nome", "numero", "tipo", "valor_centavos", "vencimento", "aquisicao", "origem", "cessao_id"},
		pgx.CopyFromRows(linhas)); err != nil {
		return c, nil, false, err
	}
	rows, err := tx.Query(ctx, `
		INSERT INTO estoque (chave, fundo_doc, fundo_nome, cedente_doc, cedente_nome, sacado_doc, sacado_nome, numero, tipo, valor_centavos, vencimento, aquisicao, origem, cessao_id)
		SELECT chave, fundo_doc, fundo_nome, cedente_doc, cedente_nome, sacado_doc, sacado_nome, numero, tipo, valor_centavos, vencimento, aquisicao, origem, cessao_id
		FROM cessao_tmp
		ON CONFLICT (chave) DO NOTHING
		RETURNING chave`)
	if err != nil {
		return c, nil, false, err
	}
	entrou := map[string]bool{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			rows.Close()
			return c, nil, false, err
		}
		entrou[k] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return c, nil, false, err
	}
	for k, rs := range porChave {
		if entrou[k] {
			c.NotasCedidas++
			c.FaceCentavos += rs.Nota.ValorCentavos
			c.DescontoCentavos += rs.DescontoCentavos
			c.LiquidoCentavos += rs.LiquidoCentavos
		} else {
			excluidas = append(excluidas, domain.Excluida{Linha: rs.Nota.Linha, Codigo: "DUPLICIDADE", Titulo: "Nota já vendida", Mensagem: msgVendidaEmParalelo})
		}
	}
	if excluidas == nil {
		excluidas = []domain.Excluida{}
	}
	excJS, _ := json.Marshal(excluidas)
	err = tx.QueryRow(ctx, `
		INSERT INTO cessoes (id, lote_id, idempotency_key, notas_cedidas, face_centavos, desconto_centavos, liquido_centavos, notas_excluidas)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING confirmada_em`,
		c.ID, loteID, chaveOuNulo, c.NotasCedidas, c.FaceCentavos, c.DescontoCentavos, c.LiquidoCentavos, excJS).Scan(&c.ConfirmadaEm)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return c, nil, false, errChaveReutilizada
		}
		return c, nil, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return c, nil, false, err
	}
	c.LoteID = loteID
	return c, excluidas, false, nil
}

func cessaoDoLote(ctx context.Context, tx pgx.Tx, loteID string) (domain.Cessao, []domain.Excluida, bool, error) {
	var c domain.Cessao
	var exc []byte
	err := tx.QueryRow(ctx, `SELECT id, confirmada_em, notas_cedidas, face_centavos, desconto_centavos, liquido_centavos, notas_excluidas FROM cessoes WHERE lote_id = $1`, loteID).
		Scan(&c.ID, &c.ConfirmadaEm, &c.NotasCedidas, &c.FaceCentavos, &c.DescontoCentavos, &c.LiquidoCentavos, &exc)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, nil, false, nil
	}
	if err != nil {
		return c, nil, false, err
	}
	c.LoteID = loteID
	var excluidas []domain.Excluida
	_ = json.Unmarshal(exc, &excluidas)
	return c, excluidas, true, nil
}

func aprovadasDoLote(ctx context.Context, tx pgx.Tx, loteID string) ([]domain.Resultado, error) {
	rows, err := tx.Query(ctx, `
		SELECT linha, nome_cedente, doc_cedente, nome_sacado, doc_sacado, numero, tipo, valor_centavos, vencimento, chave, dias, desconto_centavos, liquido_centavos
		FROM lote_notas WHERE lote_id = $1 AND status = 1 ORDER BY linha`, loteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Resultado
	for rows.Next() {
		n := &domain.Nota{}
		rs := domain.Resultado{Nota: n, Status: domain.Aprovada}
		if err := rows.Scan(&n.Linha, &n.NomeCedente, &n.DocCedente, &n.NomeSacado, &n.DocSacado, &n.Numero, &n.Tipo,
			&n.ValorCentavos, &n.Vencimento, &rs.Chave, &rs.Dias, &rs.DescontoCentavos, &rs.LiquidoCentavos); err != nil {
			return nil, err
		}
		out = append(out, rs)
	}
	return out, rows.Err()
}

func estoqueDoFundo(ctx context.Context, tx pgx.Tx, fundoDoc string) (domain.EstoqueFundo, error) {
	est := domain.EstoqueFundo{PorSacado: map[string]int64{}, PorCedente: map[string]int64{}}
	for _, alvo := range []struct {
		coluna  string
		destino map[string]int64
	}{{"sacado_doc", est.PorSacado}, {"cedente_doc", est.PorCedente}} {
		rows, err := tx.Query(ctx, `SELECT `+alvo.coluna+`, sum(valor_centavos)::bigint FROM estoque WHERE fundo_doc = $1 GROUP BY 1`, fundoDoc)
		if err != nil {
			return est, err
		}
		for rows.Next() {
			var doc string
			var soma int64
			if err := rows.Scan(&doc, &soma); err != nil {
				rows.Close()
				return est, err
			}
			alvo.destino[doc] = soma
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return est, err
		}
	}
	for _, v := range est.PorSacado {
		est.TotalCentavos += v
	}
	return est, nil
}
