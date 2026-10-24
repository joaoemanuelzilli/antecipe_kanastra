package rest

import (
	"sort"
	"time"

	"antecipe/internal/domain"
	"antecipe/internal/platform/document"
	"antecipe/internal/service/message"
	"antecipe/internal/service/pricing"
)

type precificacaoDTO struct {
	Metodo               string  `json:"metodo"`
	TaxaMensalPercentual float64 `json:"taxa_mensal_percentual"`
	BaseDias             int     `json:"base_dias"`
	Descricao            string  `json:"descricao"`
	FonteMetodo          string  `json:"fonte_metodo,omitempty"`
}

func paraPrecificacao(p domain.Precificacao) precificacaoDTO {
	taxa, _ := p.TaxaMensalPercentual.Float64()
	return precificacaoDTO{Metodo: p.Metodo, TaxaMensalPercentual: taxa, BaseDias: p.BaseDias, Descricao: p.Descricao()}
}

type etapaDTO struct {
	Codigo string `json:"codigo"`
	Titulo string `json:"titulo"`
	Status string `json:"status"`
}

var titulosEtapa = map[string]string{
	domain.EtapaLendo:        "Lendo arquivo",
	domain.EtapaValidando:    "Validando regras",
	domain.EtapaConcentracao: "Calculando concentração",
	domain.EtapaPrecificando: "Calculando valores",
	domain.EtapaSalvando:     "Salvando resultado",
}

type progressoDTO struct {
	LinhasProcessadas int     `json:"linhas_processadas"`
	LinhasTotal       int     `json:"linhas_total"`
	Percentual        float64 `json:"percentual"`
}

type motivoDTO struct {
	Codigo     string `json:"codigo"`
	Titulo     string `json:"titulo"`
	Quantidade int    `json:"quantidade"`
	Acao       string `json:"acao"`
}

type cessaoDTO struct {
	ID               string    `json:"id"`
	LoteID           string    `json:"lote_id"`
	ConfirmadaEm     time.Time `json:"confirmada_em"`
	NotasCedidas     int       `json:"notas_cedidas"`
	FaceCentavos     int64     `json:"face_centavos"`
	DescontoCentavos int64     `json:"desconto_centavos"`
	LiquidoCentavos  int64     `json:"liquido_centavos"`
}

func paraCessao(c *domain.Cessao) *cessaoDTO {
	if c == nil {
		return nil
	}
	return &cessaoDTO{ID: c.ID, LoteID: c.LoteID, ConfirmadaEm: c.ConfirmadaEm, NotasCedidas: c.NotasCedidas,
		FaceCentavos: c.FaceCentavos, DescontoCentavos: c.DescontoCentavos, LiquidoCentavos: c.LiquidoCentavos}
}

type loteDTO struct {
	ID                string              `json:"id"`
	Status            string              `json:"status"`
	Etapa             *string             `json:"etapa"`
	Etapas            []etapaDTO          `json:"etapas"`
	Progresso         progressoDTO        `json:"progresso"`
	PosicaoFila       *int                `json:"posicao_fila"`
	DataBase          string              `json:"data_base"`
	RegrasVersao      int                 `json:"regras_versao"`
	Precificacao      precificacaoDTO     `json:"precificacao"`
	CriadoEm          time.Time           `json:"criado_em"`
	IniciadoEm        *time.Time          `json:"iniciado_em"`
	ConcluidoEm       *time.Time          `json:"concluido_em"`
	DuracaoMs         *int                `json:"duracao_ms"`
	Totais            *domain.Totais      `json:"totais"`
	Motivos           []motivoDTO         `json:"motivos"`
	ErrosConfiguracao []domain.ErroConfig `json:"erros_configuracao"`
	Cessao            *cessaoDTO          `json:"cessao"`
	Erro              *domain.ErroLote    `json:"erro"`
}

func paraLote(l domain.Lote, posicao int) loteDTO {
	d := loteDTO{
		ID: l.ID, Status: l.Status, DataBase: l.DataBase.Format("2006-01-02"), RegrasVersao: l.RegrasVersao,
		Precificacao: paraPrecificacao(l.Precificacao), CriadoEm: l.CriadoEm, IniciadoEm: l.IniciadoEm,
		ConcluidoEm: l.ConcluidoEm, DuracaoMs: l.DuracaoMs, Totais: l.Totais, ErrosConfiguracao: l.ErrosConfig,
		Cessao: paraCessao(l.Cessao), Erro: l.Erro, Motivos: []motivoDTO{},
	}
	if d.ErrosConfiguracao == nil {
		d.ErrosConfiguracao = []domain.ErroConfig{}
	}
	if l.Etapa != "" && l.Status == domain.LoteProcessando {
		e := l.Etapa
		d.Etapa = &e
	}
	if posicao > 0 {
		d.PosicaoFila = &posicao
	}
	atual := -1
	for i, e := range domain.Etapas {
		if e == l.Etapa {
			atual = i
		}
	}
	for i, e := range domain.Etapas {
		st := "pendente"
		switch {
		case l.Status == domain.LoteConcluido:
			st = "concluida"
		case l.Status == domain.LoteProcessando && i < atual:
			st = "concluida"
		case l.Status == domain.LoteProcessando && i == atual:
			st = "em_andamento"
		}
		d.Etapas = append(d.Etapas, etapaDTO{Codigo: e, Titulo: titulosEtapa[e], Status: st})
	}
	d.Progresso = progressoDTO{LinhasProcessadas: l.LinhasProcessadas, LinhasTotal: l.LinhasTotal}
	if l.LinhasTotal > 0 {
		d.Progresso.Percentual = float64(int(float64(l.LinhasProcessadas)*1000/float64(l.LinhasTotal))) / 10
	}
	for codigo, q := range l.Motivos {
		d.Motivos = append(d.Motivos, motivoDTO{Codigo: codigo, Titulo: message.Titulo(codigo), Quantidade: q, Acao: message.AcaoResumo(codigo)})
	}
	sort.Slice(d.Motivos, func(i, j int) bool {
		if d.Motivos[i].Quantidade != d.Motivos[j].Quantidade {
			return d.Motivos[i].Quantidade > d.Motivos[j].Quantidade
		}
		return d.Motivos[i].Codigo < d.Motivos[j].Codigo
	})
	return d
}

type parteDTO struct {
	Nome      string `json:"nome"`
	Documento string `json:"documento"`
}

type recusaDTO struct {
	Codigo   string         `json:"codigo"`
	Titulo   string         `json:"titulo"`
	Mensagem string         `json:"mensagem"`
	Acao     string         `json:"acao"`
	Dados    map[string]any `json:"dados"`
}

type calculoDTO struct {
	Metodo               string  `json:"metodo"`
	TaxaMensalPercentual float64 `json:"taxa_mensal_percentual"`
	BaseDias             int     `json:"base_dias"`
	Dias                 int     `json:"dias"`
	Texto                string  `json:"texto"`
}

type notaDTO struct {
	Linha            int         `json:"linha"`
	Numero           string      `json:"numero"`
	Tipo             string      `json:"tipo"`
	Cedente          parteDTO    `json:"cedente"`
	Sacado           parteDTO    `json:"sacado"`
	Vencimento       *string     `json:"vencimento"`
	Dias             *int        `json:"dias"`
	Status           string      `json:"status"`
	FaceCentavos     *int64      `json:"face_centavos"`
	DescontoCentavos *int64      `json:"desconto_centavos"`
	LiquidoCentavos  *int64      `json:"liquido_centavos"`
	Recusas          []recusaDTO `json:"recusas"`
	Calculo          *calculoDTO `json:"calculo"`
}

func documentoLegivel(doc string) string {
	n := document.Normalizar(doc)
	if len(n) == 14 {
		return document.Formatar(n)
	}
	return doc
}

func paraRecusas(rs []domain.Recusa) []recusaDTO {
	out := make([]recusaDTO, 0, len(rs))
	for _, r := range rs {
		t := message.Recusa(r.Codigo, r.Dados)
		dados := r.Dados
		if dados == nil {
			dados = map[string]any{}
		}
		out = append(out, recusaDTO{Codigo: r.Codigo, Titulo: t.Titulo, Mensagem: t.Mensagem, Acao: t.Acao, Dados: dados})
	}
	return out
}

func paraNota(n domain.NotaLinha, calc *pricing.Calculadora) notaDTO {
	d := notaDTO{
		Linha: n.Linha, Numero: n.Numero, Tipo: n.Tipo,
		Cedente: parteDTO{n.NomeCedente, documentoLegivel(n.DocCedente)}, Sacado: parteDTO{n.NomeSacado, documentoLegivel(n.DocSacado)},
		Dias: n.Dias, Status: n.Status.String(), FaceCentavos: n.ValorCentavos,
		DescontoCentavos: n.DescontoCentavos, LiquidoCentavos: n.LiquidoCentavos, Recusas: paraRecusas(n.Recusas),
	}
	if n.Vencimento != nil {
		v := n.Vencimento.Format("2006-01-02")
		d.Vencimento = &v
	}
	if n.Status == domain.Aprovada && calc != nil && n.ValorCentavos != nil && n.Dias != nil {
		c := calc.Config()
		taxa, _ := c.TaxaMensalPercentual.Float64()
		d.Calculo = &calculoDTO{Metodo: c.Metodo, TaxaMensalPercentual: taxa, BaseDias: c.BaseDias, Dias: *n.Dias, Texto: calc.Texto(*n.ValorCentavos, *n.Dias)}
	}
	return d
}

// paraNotaDeResultado converte um resultado do motor (validação síncrona) para o mesmo formato.
func paraNotaDeResultado(r domain.Resultado, calc *pricing.Calculadora) notaDTO {
	n := r.Nota
	l := domain.NotaLinha{
		Linha: n.Linha, NomeCedente: n.NomeCedente, DocCedente: n.DocCedente, NomeSacado: n.NomeSacado, DocSacado: n.DocSacado,
		Numero: n.Numero, Tipo: n.Tipo, Status: r.Status, Recusas: r.Recusas,
	}
	if n.ErroFormato == nil {
		v, venc, dias := n.ValorCentavos, n.Vencimento, r.Dias
		l.ValorCentavos, l.Vencimento, l.Dias = &v, &venc, &dias
	}
	if r.Status == domain.Aprovada {
		d, q := r.DescontoCentavos, r.LiquidoCentavos
		l.DescontoCentavos, l.LiquidoCentavos = &d, &q
	}
	return paraNota(l, calc)
}
