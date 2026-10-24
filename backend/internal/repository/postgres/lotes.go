package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"antecipe/internal/domain"
	"antecipe/internal/platform/apperrors"
)

var _ domain.LoteRepository = (*Repo)(nil)

var errLoteNaoEncontrado = apperrors.NotFound("LOTE_NAO_ENCONTRADO", "Lote não encontrado.")

// CriarLote registra o envio. Se a chave de idempotência já existe, devolve o lote existente (criado=false).
func (r *Repo) CriarLote(ctx context.Context, n domain.NovoLote) (domain.LoteCriado, error) {
	var c domain.LoteCriado
	prec, err := json.Marshal(n.Precificacao)
	if err != nil {
		return c, err
	}
	err = r.Pool.QueryRow(ctx, `
		INSERT INTO lotes (idempotency_key, impressao, hash_arquivo, data_base, regras_versao, precificacao, status, linhas_total)
		VALUES ($1, $2, $3, $4, $5, $6, 'na_fila', $7)
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING id`, n.IdempotencyKey, n.Impressao, n.Hash, n.DataBase, n.RegrasVersao, prec, n.LinhasTotal).Scan(&c.ID)
	if err == nil {
		return domain.LoteCriado{ID: c.ID, Criado: true, Status: domain.LoteNaFila, Impressao: n.Impressao}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return c, err
	}
	err = r.Pool.QueryRow(ctx, `SELECT id, status, impressao FROM lotes WHERE idempotency_key = $1`, n.IdempotencyKey).Scan(&c.ID, &c.Status, &c.Impressao)
	return c, err
}

// ReiniciarLote volta um lote com erro para a fila. Só um chamador consegue (os outros recebem false).
func (r *Repo) ReiniciarLote(ctx context.Context, id string, linhasTotal int, impressao string) (bool, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck
	tag, err := tx.Exec(ctx, `
		UPDATE lotes SET status = 'na_fila', etapa = NULL, linhas_total = $2, linhas_processadas = 0, totais = NULL,
		       motivos = NULL, erros_configuracao = '[]', erro = NULL, iniciado_em = NULL, concluido_em = NULL, duracao_ms = NULL
		WHERE id = $1 AND status = 'erro' AND impressao = $3`, id, linhasTotal, impressao)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	if _, err := tx.Exec(ctx, `DELETE FROM lote_notas WHERE lote_id = $1`, id); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// MarcarEtapa registra status e etapa (o progresso fino fica em memória no serviço).
func (r *Repo) MarcarEtapa(ctx context.Context, id, status, etapa string) error {
	_, err := r.Pool.Exec(ctx, `UPDATE lotes SET status = $2, etapa = NULLIF($3, ''), iniciado_em = COALESCE(iniciado_em, now()) WHERE id = $1`, id, status, etapa)
	return err
}

// GravarResultados grava as notas avaliadas com COPY (centenas de milhares de linhas em segundos).
func (r *Repo) GravarResultados(ctx context.Context, id string, rs []domain.Resultado, aoAvancar func(int)) error {
	_, err := r.Pool.CopyFrom(ctx, pgx.Identifier{"lote_notas"}, colunasCopia, &fonteResultados{lote: id, rs: rs, aoAvancar: aoAvancar, i: -1})
	if err != nil {
		return fmt.Errorf("gravar resultados: %w", err)
	}
	return nil
}

var colunasCopia = []string{"lote_id", "linha", "nome_fundo", "doc_fundo", "nome_cedente", "doc_cedente", "nome_sacado", "doc_sacado",
	"numero", "tipo", "valor_centavos", "vencimento", "aquisicao", "chave", "status", "motivo_principal", "dias",
	"desconto_centavos", "liquido_centavos", "recusas", "erro_formato"}

type fonteResultados struct {
	lote      string
	rs        []domain.Resultado
	i         int
	err       error
	aoAvancar func(int)
}

func (f *fonteResultados) Next() bool {
	f.i++
	if f.aoAvancar != nil && f.i > 0 && f.i%10000 == 0 {
		f.aoAvancar(f.i)
	}
	return f.i < len(f.rs)
}

func (f *fonteResultados) Values() ([]any, error) {
	rs := &f.rs[f.i]
	n := rs.Nota
	recusas, err := json.Marshal(rs.Recusas)
	if err != nil {
		return nil, err
	}
	if rs.Recusas == nil {
		recusas = []byte("[]")
	}
	var valor, venc, dias, desconto, liquido, motivo, erroFormato any
	if n.ErroFormato == nil {
		valor, venc, dias = n.ValorCentavos, n.Vencimento, rs.Dias
	} else {
		b, _ := json.Marshal(n.ErroFormato)
		erroFormato = b
	}
	if rs.Status == domain.Aprovada {
		desconto, liquido = rs.DescontoCentavos, rs.LiquidoCentavos
	} else if len(rs.Recusas) > 0 {
		motivo = rs.Recusas[0].Codigo
	}
	return []any{f.lote, n.Linha, n.NomeFundo, n.DocFundo, n.NomeCedente, n.DocCedente, n.NomeSacado, n.DocSacado,
		n.Numero, n.Tipo, valor, venc, dataOuNulo(n.Aquisicao), rs.Chave, int16(rs.Status), motivo, dias,
		desconto, liquido, recusas, erroFormato}, nil
}

func (f *fonteResultados) Err() error { return f.err }

// ConcluirLote grava totais e motivos.
func (r *Repo) ConcluirLote(ctx context.Context, id string, res domain.ResumoLote) error {
	tot, _ := json.Marshal(res.Totais)
	mot, _ := json.Marshal(res.Motivos)
	cfg, _ := json.Marshal(res.ErrosConfig)
	if res.ErrosConfig == nil {
		cfg = []byte("[]")
	}
	_, err := r.Pool.Exec(ctx, `
		UPDATE lotes SET status = 'concluido', etapa = NULL, totais = $2, motivos = $3, erros_configuracao = $4,
		       concluido_em = now(), duracao_ms = $5, linhas_processadas = linhas_total
		WHERE id = $1`, id, tot, mot, cfg, res.DuracaoMs)
	return err
}

// FalharLote marca o lote como erro.
func (r *Repo) FalharLote(ctx context.Context, id string, e domain.ErroLote) error {
	b, _ := json.Marshal(e)
	_, err := r.Pool.Exec(ctx, `UPDATE lotes SET status = 'erro', etapa = NULL, erro = $2, concluido_em = now() WHERE id = $1`, id, b)
	return err
}

// RecuperarInterrompidos marca como erro os lotes que estavam na fila ou processando quando a API parou.
func (r *Repo) RecuperarInterrompidos(ctx context.Context) (int, error) {
	tag, err := r.Pool.Exec(ctx, `
		UPDATE lotes SET status = 'erro', etapa = NULL, concluido_em = now(),
		       erro = '{"codigo":"PROCESSAMENTO_INTERROMPIDO","mensagem":"O processamento foi interrompido. Envie o arquivo de novo."}'
		WHERE status IN ('na_fila', 'processando')`)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// ObterLote devolve o lote com a cessão, se houver.
func (r *Repo) ObterLote(ctx context.Context, id string) (domain.Lote, error) {
	var l domain.Lote
	if !uuidValido(id) {
		return l, errLoteNaoEncontrado
	}
	var prec, totais, motivos, errosCfg, erro []byte
	var etapa *string
	var cID *string
	var cEm *time.Time
	var cNotas *int
	var cFace, cDesc, cLiq *int64
	err := r.Pool.QueryRow(ctx, `
		SELECT l.id, l.idempotency_key, l.hash_arquivo, l.data_base, l.regras_versao, l.precificacao, l.status, l.etapa,
		       l.linhas_total, l.linhas_processadas, l.totais, l.motivos, l.erros_configuracao, l.erro,
		       l.criado_em, l.iniciado_em, l.concluido_em, l.duracao_ms,
		       c.id, c.confirmada_em, c.notas_cedidas, c.face_centavos, c.desconto_centavos, c.liquido_centavos
		FROM lotes l LEFT JOIN cessoes c ON c.lote_id = l.id
		WHERE l.id = $1`, id).Scan(
		&l.ID, &l.IdempotencyKey, &l.HashArquivo, &l.DataBase, &l.RegrasVersao, &prec, &l.Status, &etapa,
		&l.LinhasTotal, &l.LinhasProcessadas, &totais, &motivos, &errosCfg, &erro,
		&l.CriadoEm, &l.IniciadoEm, &l.ConcluidoEm, &l.DuracaoMs,
		&cID, &cEm, &cNotas, &cFace, &cDesc, &cLiq)
	if errors.Is(err, pgx.ErrNoRows) {
		return l, errLoteNaoEncontrado
	}
	if err != nil {
		return l, err
	}
	if etapa != nil {
		l.Etapa = *etapa
	}
	_ = json.Unmarshal(prec, &l.Precificacao)
	if totais != nil {
		l.Totais = &domain.Totais{}
		_ = json.Unmarshal(totais, l.Totais)
	}
	if motivos != nil {
		_ = json.Unmarshal(motivos, &l.Motivos)
	}
	_ = json.Unmarshal(errosCfg, &l.ErrosConfig)
	if erro != nil {
		l.Erro = &domain.ErroLote{}
		_ = json.Unmarshal(erro, l.Erro)
	}
	if cID != nil {
		l.Cessao = &domain.Cessao{ID: *cID, LoteID: l.ID, ConfirmadaEm: *cEm, NotasCedidas: *cNotas, FaceCentavos: *cFace, DescontoCentavos: *cDesc, LiquidoCentavos: *cLiq}
	}
	return l, nil
}

const colunasNota = `linha, nome_fundo, doc_fundo, nome_cedente, doc_cedente, nome_sacado, doc_sacado, numero, tipo,
	valor_centavos, vencimento, aquisicao, chave, status, motivo_principal, dias, desconto_centavos, liquido_centavos, recusas, erro_formato`

var ordenacoes = map[string]string{
	"linha": "linha", "vencimento": "vencimento", "dias": "dias", "face": "valor_centavos",
	"liquido": "liquido_centavos", "sacado": "nome_sacado",
}

func filtroSQL(id string, f domain.FiltroNotas) (string, []any) {
	cond := []string{"lote_id = $1"}
	args := []any{id}
	switch f.Status {
	case "aprovada":
		cond = append(cond, "status = 1")
	case "recusada":
		cond = append(cond, "status = 2")
	}
	if f.Motivo != "" {
		args = append(args, f.Motivo)
		cond = append(cond, fmt.Sprintf("motivo_principal = $%d", len(args)))
	}
	if b := strings.TrimSpace(f.Busca); b != "" {
		args = append(args, "%"+b+"%")
		p := fmt.Sprintf("$%d", len(args))
		cond = append(cond, "(numero ILIKE "+p+" OR nome_sacado ILIKE "+p+" OR nome_cedente ILIKE "+p+
			" OR translate(doc_sacado, './- ', '') ILIKE translate("+p+", './- ', '')"+
			" OR translate(doc_cedente, './- ', '') ILIKE translate("+p+", './- ', ''))")
	}
	col, ok := ordenacoes[f.Ordenar]
	if !ok {
		col = "linha"
	}
	dir := "ASC"
	if strings.EqualFold(f.Direcao, "desc") {
		dir = "DESC"
	}
	return " WHERE " + strings.Join(cond, " AND ") + " ORDER BY " + col + " " + dir + " NULLS LAST, linha ASC", args
}

// ListarNotas devolve uma página e o total filtrado.
func (r *Repo) ListarNotas(ctx context.Context, id string, f domain.FiltroNotas) ([]domain.NotaLinha, int, error) {
	if !uuidValido(id) {
		return nil, 0, errLoteNaoEncontrado
	}
	where, args := filtroSQL(id, f)
	args = append(args, f.PorPagina, (f.Pagina-1)*f.PorPagina)
	q := `SELECT ` + colunasNota + `, count(*) OVER () FROM lote_notas` + where +
		fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := r.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var itens []domain.NotaLinha
	total := 0
	for rows.Next() {
		n, err := lerNota(rows, &total)
		if err != nil {
			return nil, 0, err
		}
		itens = append(itens, n)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if len(itens) == 0 {
		// página além do fim: o total ainda precisa vir certo
		cw, ca := filtroSQL(id, f)
		cw = cw[:strings.Index(cw, " ORDER BY")]
		if err := r.Pool.QueryRow(ctx, `SELECT count(*) FROM lote_notas`+cw, ca...).Scan(&total); err != nil {
			return nil, 0, err
		}
	}
	return itens, total, nil
}

// PercorrerNotas lê todas as notas filtradas em fluxo (para o CSV de download).
func (r *Repo) PercorrerNotas(ctx context.Context, id string, f domain.FiltroNotas, fn func(domain.NotaLinha) error) error {
	if !uuidValido(id) {
		return errLoteNaoEncontrado
	}
	where, args := filtroSQL(id, f)
	rows, err := r.Pool.Query(ctx, `SELECT `+colunasNota+` FROM lote_notas`+where, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		n, err := lerNota(rows, nil)
		if err != nil {
			return err
		}
		if err := fn(n); err != nil {
			return err
		}
	}
	return rows.Err()
}

func lerNota(rows pgx.Rows, total *int) (domain.NotaLinha, error) {
	var n domain.NotaLinha
	var nomeFundo, docFundo, nomeCed, docCed, nomeSac, docSac, numero, tipo, chave, motivo *string
	var status int16
	var recusas, erroFormato []byte
	dest := []any{&n.Linha, &nomeFundo, &docFundo, &nomeCed, &docCed, &nomeSac, &docSac, &numero, &tipo,
		&n.ValorCentavos, &n.Vencimento, &n.Aquisicao, &chave, &status, &motivo, &n.Dias, &n.DescontoCentavos, &n.LiquidoCentavos,
		&recusas, &erroFormato}
	if total != nil {
		dest = append(dest, total)
	}
	if err := rows.Scan(dest...); err != nil {
		return n, err
	}
	s := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	n.NomeFundo, n.DocFundo, n.NomeCedente, n.DocCedente = s(nomeFundo), s(docFundo), s(nomeCed), s(docCed)
	n.NomeSacado, n.DocSacado, n.Numero, n.Tipo, n.Chave, n.MotivoPrincipal = s(nomeSac), s(docSac), s(numero), s(tipo), s(chave), s(motivo)
	n.Status = domain.Status(status)
	_ = json.Unmarshal(recusas, &n.Recusas)
	if erroFormato != nil {
		n.ErroFormato = &domain.ErroCampo{}
		_ = json.Unmarshal(erroFormato, n.ErroFormato)
	}
	return n, nil
}

func uuidValido(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
				return false
			}
		}
	}
	return true
}
