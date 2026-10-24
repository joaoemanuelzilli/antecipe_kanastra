package domain

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// Métodos de precificação.
const (
	MetodoComposto = "DESCONTO_COMPOSTO"
	MetodoSimples  = "DESCONTO_SIMPLES"
)

// Precificacao é o que define o preço: método, taxa mensal (%) e base em dias.
type Precificacao struct {
	Metodo               string          `json:"metodo"`
	TaxaMensalPercentual decimal.Decimal `json:"taxa_mensal_percentual"`
	BaseDias             int             `json:"base_dias"`
}

// Descricao devolve o texto curto para a tela: "Juros compostos · 2% ao mês".
func (p Precificacao) Descricao() string {
	nome := "Juros compostos"
	if p.Metodo == MetodoSimples {
		nome = "Juros simples"
	}
	return nome + " · " + strings.Replace(p.TaxaMensalPercentual.String(), ".", ",", 1) + "% ao mês"
}

// Totais de um lote (face, desconto e líquido somam só as aprovadas).
type Totais struct {
	Linhas           int   `json:"linhas"`
	Aprovadas        int   `json:"aprovadas"`
	Recusadas        int   `json:"recusadas"`
	FaceCentavos     int64 `json:"face_centavos"`
	DescontoCentavos int64 `json:"desconto_centavos"`
	LiquidoCentavos  int64 `json:"liquido_centavos"`
}

// ErroConfig é uma regra que não pôde ser aplicada (código desconhecido ou parâmetro inválido).
type ErroConfig struct {
	Codigo   string `json:"codigo"`
	Mensagem string `json:"mensagem"`
}

// Status e etapas do processamento de um lote.
const (
	LoteNaFila      = "na_fila"
	LoteProcessando = "processando"
	LoteConcluido   = "concluido"
	LoteErro        = "erro"

	EtapaLendo        = "lendo"
	EtapaValidando    = "validando"
	EtapaConcentracao = "concentracao"
	EtapaPrecificando = "precificando"
	EtapaSalvando     = "salvando"
)

// Etapas na ordem em que acontecem.
var Etapas = []string{EtapaLendo, EtapaValidando, EtapaConcentracao, EtapaPrecificando, EtapaSalvando}

// ErroLote explica por que o processamento falhou.
type ErroLote struct {
	Codigo   string `json:"codigo"`
	Mensagem string `json:"mensagem"`
}

// Lote é um arquivo enviado e o resultado da simulação.
type Lote struct {
	ID                string
	IdempotencyKey    string
	HashArquivo       string
	DataBase          time.Time
	RegrasVersao      int
	Precificacao      Precificacao
	Status            string
	Etapa             string
	LinhasTotal       int
	LinhasProcessadas int
	Totais            *Totais
	Motivos           map[string]int
	ErrosConfig       []ErroConfig
	Erro              *ErroLote
	CriadoEm          time.Time
	IniciadoEm        *time.Time
	ConcluidoEm       *time.Time
	DuracaoMs         *int
	Cessao            *Cessao
}

// Cessao é a venda confirmada das notas aprovadas de um lote.
type Cessao struct {
	ID               string
	LoteID           string
	ConfirmadaEm     time.Time
	NotasCedidas     int
	FaceCentavos     int64
	DescontoCentavos int64
	LiquidoCentavos  int64
}

// NovoLote são os dados para registrar um envio.
type NovoLote struct {
	IdempotencyKey string
	Hash           string
	DataBase       time.Time
	RegrasVersao   int
	Precificacao   Precificacao
	LinhasTotal    int
}

// ResumoLote é o que se grava ao concluir.
type ResumoLote struct {
	Totais      Totais
	Motivos     map[string]int
	ErrosConfig []ErroConfig
	DuracaoMs   int
}

// NotaLinha é uma nota avaliada, como fica guardada (campos ilegíveis vêm nil).
type NotaLinha struct {
	Linha                             int
	NomeFundo, DocFundo               string
	NomeCedente, DocCedente           string
	NomeSacado, DocSacado             string
	Numero, Tipo                      string
	ValorCentavos                     *int64
	Vencimento, Aquisicao             *time.Time
	Chave                             string
	Status                            Status
	MotivoPrincipal                   string
	Dias                              *int
	DescontoCentavos, LiquidoCentavos *int64
	Recusas                           []Recusa
	ErroFormato                       *ErroCampo
}

// FiltroNotas da listagem paginada.
type FiltroNotas struct {
	Status, Motivo, Busca, Ordenar, Direcao string
	Pagina, PorPagina                       int
}

// LoteRepository guarda lotes e os resultados das notas.
type LoteRepository interface {
	CriarLote(ctx context.Context, n NovoLote) (id string, criado bool, status string, err error)
	ReiniciarLote(ctx context.Context, id string, linhasTotal int) (bool, error) // false: outro envio já reiniciou
	MarcarEtapa(ctx context.Context, id, status, etapa string) error
	GravarResultados(ctx context.Context, id string, rs []Resultado, aoAvancar func(int)) error
	ConcluirLote(ctx context.Context, id string, r ResumoLote) error
	FalharLote(ctx context.Context, id string, e ErroLote) error
	ObterLote(ctx context.Context, id string) (Lote, error)
	ListarNotas(ctx context.Context, id string, f FiltroNotas) ([]NotaLinha, int, error)
	PercorrerNotas(ctx context.Context, id string, f FiltroNotas, fn func(NotaLinha) error) error
	RecuperarInterrompidos(ctx context.Context) (int, error)
}

// LeitorNotas lê o arquivo enviado (o adaptador do formato CSV implementa).
type LeitorNotas interface {
	Ler(r io.Reader, aoAvancar func(lidas int)) ([]Nota, error)
	ContarLinhas(r io.Reader) (int, error)
}

// Excluida é uma nota aprovada na simulação que não pôde ser vendida na confirmação.
type Excluida struct {
	Linha    int    `json:"linha"`
	Codigo   string `json:"codigo"`
	Titulo   string `json:"titulo"`
	Mensagem string `json:"mensagem"`
}

// FundoDestino é o fundo que compra as notas na cessão.
type FundoDestino struct {
	Nome, CNPJ string
}

// Reverificar recebe as aprovadas e o estoque atual (dentro da transação) e devolve as que não podem mais ser vendidas.
type Reverificar func(aprovadas []Resultado, estoque EstoqueFundo) []Excluida

// CessaoRepository executa a venda de forma atômica e segura contra concorrência.
type CessaoRepository interface {
	Ceder(ctx context.Context, loteID, idempotencyKey string, fundo FundoDestino, reverificar Reverificar) (c Cessao, excluidas []Excluida, replay bool, err error)
}
