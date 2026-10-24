package rest

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"antecipe/internal/domain"
	"antecipe/internal/handler/csvfile"
	"antecipe/internal/platform/apperrors"
	"antecipe/internal/platform/format"
	"antecipe/internal/service/batch"
	"antecipe/internal/service/message"
	"antecipe/internal/service/pricing"
)

var (
	errArquivoAusente = apperrors.BadRequest("ARQUIVO_AUSENTE", "Envie o arquivo CSV no campo \"arquivo\" (multipart/form-data).")
	errDataBase       = apperrors.BadRequest("DATA_BASE_INVALIDA", "A data-base deve estar no formato AAAA-MM-DD.")
)

func (h *Handlers) config(w http.ResponseWriter, r *http.Request) {
	a, err := h.d.Lotes.ConfiguracaoAtual(r.Context())
	if err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	p := paraPrecificacao(a.Precificacao)
	p.FonteMetodo = "regras publicadas"
	for _, s := range a.Sobrescritos {
		if s == "metodo" {
			p.FonteMetodo = "config/antecipe.yaml"
		}
	}
	EscreverJSON(w, http.StatusOK, map[string]any{
		"fundo":            map[string]string{"nome": a.Versao.Regras.Fundo.Nome, "cnpj": a.Versao.Regras.Fundo.CNPJ},
		"data_base_padrao": a.DataBase.Format("2006-01-02"),
		"regras_versao":    a.Versao.Numero,
		"precificacao":     p,
		"upload": map[string]any{
			"tamanho_maximo_bytes": int64(a.TamanhoMaxMB) << 20,
			"separador":            ";",
			"colunas":              csvfile.Colunas,
		},
	})
}

// enviarLote lê o multipart em fluxo, grava o arquivo em disco calculando o sha256 e entrega ao serviço.
func (h *Handlers) enviarLote(w http.ResponseWriter, r *http.Request) {
	a, err := h.d.Lotes.ConfiguracaoAtual(r.Context())
	if err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	limite := int64(a.TamanhoMaxMB) << 20
	r.Body = http.MaxBytesReader(w, r.Body, limite+1<<20)
	mr, err := r.MultipartReader()
	if err != nil {
		EscreverErro(w, r, h.d.Log, errArquivoAusente)
		return
	}
	var caminho, hash string
	var dataBase *time.Time
	limpar := func() {
		if caminho != "" {
			os.Remove(caminho)
		}
	}
	for {
		parte, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			limpar()
			EscreverErro(w, r, h.d.Log, erroDeLeitura(err))
			return
		}
		switch parte.FormName() {
		case "data_base":
			v, _ := io.ReadAll(io.LimitReader(parte, 64))
			t, err := time.Parse("2006-01-02", strings.TrimSpace(string(v)))
			if err != nil {
				limpar()
				EscreverErro(w, r, h.d.Log, errDataBase)
				return
			}
			dataBase = &t
		case "arquivo":
			if caminho != "" {
				continue
			}
			f, err := os.CreateTemp(h.d.TmpDir, "lote-*.csv")
			if err != nil {
				EscreverErro(w, r, h.d.Log, apperrors.Internal(err))
				return
			}
			caminho = f.Name()
			sum := sha256.New()
			_, err = io.Copy(io.MultiWriter(f, sum), parte)
			f.Close()
			if err != nil {
				limpar()
				EscreverErro(w, r, h.d.Log, erroDeLeitura(err))
				return
			}
			hash = hex.EncodeToString(sum.Sum(nil))
		}
	}
	if caminho == "" {
		EscreverErro(w, r, h.d.Log, errArquivoAusente)
		return
	}
	if st, err := os.Stat(caminho); err == nil && st.Size() > limite {
		limpar()
		EscreverErro(w, r, h.d.Log, apperrors.TooLarge("ARQUIVO_GRANDE_DEMAIS", "O arquivo passa do limite de "+strconv.Itoa(a.TamanhoMaxMB)+" MB."))
		return
	}
	f, err := os.Open(caminho)
	if err != nil {
		limpar()
		EscreverErro(w, r, h.d.Log, apperrors.Internal(err))
		return
	}
	ea := csvfile.ValidarCabecalho(f)
	f.Close()
	if ea != nil {
		limpar()
		EscreverErro(w, r, h.d.Log, ea.AppError())
		return
	}
	chave := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if len(chave) > 128 {
		limpar()
		EscreverErro(w, r, h.d.Log, apperrors.BadRequest("IDEMPOTENCY_KEY_INVALIDA", "A Idempotency-Key deve ter até 128 caracteres."))
		return
	}
	id, replay, err := h.d.Lotes.Enviar(r.Context(), batch.Envio{CaminhoArquivo: caminho, Hash: hash, IdempotencyKey: chave, DataBase: dataBase})
	if err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	l, _, err := h.d.Lotes.Obter(r.Context(), id)
	status := domain.LoteNaFila
	if err == nil {
		status = l.Status
	}
	w.Header().Set("Location", "/api/lotes/"+id)
	if replay {
		w.Header().Set("Idempotent-Replay", "true")
		EscreverJSON(w, http.StatusOK, map[string]any{"lote_id": id, "status": status, "replay": true})
		return
	}
	EscreverJSON(w, http.StatusAccepted, map[string]any{"lote_id": id, "status": status, "replay": false})
}

func erroDeLeitura(err error) error {
	var mbe *http.MaxBytesError
	if errors.As(err, &mbe) {
		return apperrors.TooLarge("ARQUIVO_GRANDE_DEMAIS", "O arquivo passa do limite aceito.")
	}
	return apperrors.BadRequest("UPLOAD_INVALIDO", "Não conseguimos ler o envio. Tente de novo.").Wrap(err)
}

func (h *Handlers) obterLote(w http.ResponseWriter, r *http.Request) {
	l, pos, err := h.d.Lotes.Obter(r.Context(), r.PathValue("id"))
	if err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	EscreverJSON(w, http.StatusOK, paraLote(l, pos))
}

func filtroDe(r *http.Request) domain.FiltroNotas {
	q := r.URL.Query()
	f := domain.FiltroNotas{Status: q.Get("status"), Motivo: q.Get("motivo"), Busca: q.Get("busca"), Ordenar: q.Get("ordenar"), Direcao: q.Get("direcao"), Pagina: 1, PorPagina: 50}
	if p, err := strconv.Atoi(q.Get("pagina")); err == nil && p > 0 {
		f.Pagina = p
	}
	if p, err := strconv.Atoi(q.Get("por_pagina")); err == nil && p > 0 {
		f.PorPagina = min(p, 200)
	}
	return f
}

func (h *Handlers) listarNotas(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	l, _, err := h.d.Lotes.Obter(r.Context(), id)
	if err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	f := filtroDe(r)
	itens, total, err := h.d.Lotes.ListarNotas(r.Context(), id, f)
	if err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	calc, _ := pricing.Nova(l.Precificacao)
	out := make([]notaDTO, 0, len(itens))
	for _, n := range itens {
		out = append(out, paraNota(n, calc))
	}
	paginas := (total + f.PorPagina - 1) / f.PorPagina
	EscreverJSON(w, http.StatusOK, map[string]any{"pagina": f.Pagina, "por_pagina": f.PorPagina, "total": total, "total_paginas": paginas, "itens": out})
}

// baixarNotas devolve o resultado em CSV, em fluxo, com as células de texto neutralizadas contra fórmulas.
func (h *Handlers) baixarNotas(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, _, err := h.d.Lotes.Obter(r.Context(), id); err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="lote-`+id+`.csv"`)
	_, _ = w.Write([]byte("\xEF\xBB\xBF")) // BOM: o Excel abre os acentos certos
	cw := csv.NewWriter(w)
	cw.Comma = ';'
	_ = cw.Write(append(append([]string{}, csvfile.Colunas...), "STATUS", "MOTIVO", "O_QUE_FAZER", "DIAS", "DESCONTO", "VALOR_LIQUIDO"))
	c := format.CelulaCSV
	err := h.d.Lotes.PercorrerNotas(r.Context(), id, filtroDe(r), func(n domain.NotaLinha) error {
		valor, venc, aq, dias, desc, liq := "", "", "", "", "", ""
		if n.ValorCentavos != nil {
			valor = format.ValorCSV(*n.ValorCentavos)
		}
		if n.Vencimento != nil {
			venc = format.Data(*n.Vencimento)
		}
		if n.Aquisicao != nil {
			aq = format.Data(*n.Aquisicao)
		}
		if n.Dias != nil {
			dias = strconv.Itoa(*n.Dias)
		}
		status, motivo, acao := "APROVADA", "", ""
		if n.Status == domain.Recusada && len(n.Recusas) > 0 {
			t := message.Recusa(n.Recusas[0].Codigo, n.Recusas[0].Dados)
			status, motivo, acao = "RECUSADA", t.Titulo+": "+t.Mensagem, t.Acao
		} else if n.DescontoCentavos != nil && n.LiquidoCentavos != nil {
			desc, liq = format.ValorCSV(*n.DescontoCentavos), format.ValorCSV(*n.LiquidoCentavos)
		}
		return cw.Write([]string{c(n.NomeFundo), c(n.DocFundo), c(n.NomeCedente), c(n.DocCedente), c(n.NomeSacado), c(n.DocSacado),
			c(n.Numero), c(n.Tipo), valor, venc, aq, status, c(motivo), acao, dias, desc, liq})
	})
	cw.Flush()
	if err != nil {
		h.d.Log.ErrorContext(r.Context(), "falha no download do csv", "lote_id", id)
	}
}
