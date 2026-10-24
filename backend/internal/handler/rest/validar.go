package rest

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"antecipe/internal/domain"
	"antecipe/internal/handler/csvfile"
	"antecipe/internal/platform/apperrors"
	"antecipe/internal/service/message"
	"antecipe/internal/service/pricing"
)

const limiteValidar = 10000

type notaEntrada struct {
	NomeFundo              string `json:"nome_fundo"`
	DocFundo               string `json:"doc_fundo"`
	NomeCedente            string `json:"nome_cedente"`
	DocCedente             string `json:"doc_cedente"`
	NomeSacado             string `json:"nome_sacado"`
	DocSacado              string `json:"doc_sacado"`
	NuDocumento            string `json:"nu_documento"`
	TipoRecebivel          string `json:"tipo_recebivel"`
	ValorNominal           string `json:"valor_nominal"`
	DataVencimentoOriginal string `json:"data_vencimento_original"`
	DataAquisicao          string `json:"data_aquisicao"`
}

// validar roda o mesmo motor dos lotes, sem gravar nada. Aceita JSON ou o próprio CSV.
func (h *Handlers) validar(w http.ResponseWriter, r *http.Request) {
	corpo, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 16<<20))
	if err != nil {
		EscreverErro(w, r, h.d.Log, erroDeLeitura(err))
		return
	}
	var dataBase *time.Time
	var csvTexto []byte
	if strings.HasPrefix(r.Header.Get("Content-Type"), "text/csv") {
		csvTexto = corpo
		if v := r.URL.Query().Get("data_base"); v != "" {
			t, err := time.Parse("2006-01-02", v)
			if err != nil {
				EscreverErro(w, r, h.d.Log, errDataBase)
				return
			}
			dataBase = &t
		}
	} else {
		var in struct {
			DataBase string        `json:"data_base"`
			Notas    []notaEntrada `json:"notas"`
		}
		if err := json.Unmarshal(corpo, &in); err != nil {
			EscreverErro(w, r, h.d.Log, apperrors.BadRequest("JSON_INVALIDO", "O corpo deve ser JSON com data_base e notas, ou um CSV (Content-Type: text/csv)."))
			return
		}
		if in.DataBase != "" {
			t, err := time.Parse("2006-01-02", in.DataBase)
			if err != nil {
				EscreverErro(w, r, h.d.Log, errDataBase)
				return
			}
			dataBase = &t
		}
		if len(in.Notas) > limiteValidar {
			EscreverErro(w, r, h.d.Log, apperrors.TooLarge("VALIDACAO_GRANDE_DEMAIS", "Para mais de 10.000 notas, use POST /api/lotes."))
			return
		}
		// reaproveita o mesmo leitor do CSV para as notas em JSON (mesmas regras de formato)
		var b bytes.Buffer
		cw := csv.NewWriter(&b)
		cw.Comma = ';'
		_ = cw.Write(csvfile.Colunas)
		for _, n := range in.Notas {
			_ = cw.Write([]string{n.NomeFundo, n.DocFundo, n.NomeCedente, n.DocCedente, n.NomeSacado, n.DocSacado, n.NuDocumento, n.TipoRecebivel, n.ValorNominal, n.DataVencimentoOriginal, n.DataAquisicao})
		}
		cw.Flush()
		csvTexto = b.Bytes()
	}
	notas, err := csvfile.Leitor{}.Ler(bytes.NewReader(csvTexto), nil)
	if err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	if len(notas) > limiteValidar {
		EscreverErro(w, r, h.d.Log, apperrors.TooLarge("VALIDACAO_GRANDE_DEMAIS", "Para mais de 10.000 notas, use POST /api/lotes."))
		return
	}
	v, err := h.d.Lotes.ValidarAgora(r.Context(), notas, dataBase)
	if err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	calc, _ := pricing.Nova(v.Precificacao)
	saida := make([]notaDTO, 0, len(v.Saida.Resultados))
	for _, res := range v.Saida.Resultados {
		saida = append(saida, paraNotaDeResultado(res, calc))
	}
	EscreverJSON(w, http.StatusOK, map[string]any{
		"data_base": v.DataBase.Format("2006-01-02"), "regras_versao": v.RegrasVersao, "precificacao": paraPrecificacao(v.Precificacao),
		"totais": v.Saida.Totais, "motivos": motivosDe(v.Saida.Motivos), "erros_configuracao": errosOuVazio(v.Saida.ErrosConfig), "notas": saida,
	})
}

func motivosDe(m map[string]int) []motivoDTO {
	out := []motivoDTO{}
	for c, q := range m {
		out = append(out, motivoDTO{Codigo: c, Titulo: message.Titulo(c), Quantidade: q, Acao: message.AcaoResumo(c)})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Quantidade > out[j].Quantidade || (out[i].Quantidade == out[j].Quantidade && out[i].Codigo < out[j].Codigo)
	})
	return out
}

func errosOuVazio(e []domain.ErroConfig) []domain.ErroConfig {
	if e == nil {
		return []domain.ErroConfig{}
	}
	return e
}
