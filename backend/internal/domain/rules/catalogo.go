package rules

// ParametroCatalogo descreve um parâmetro para a tela do gestor montar o formulário.
type ParametroCatalogo struct {
	Nome     string   `json:"nome"`
	Rotulo   string   `json:"rotulo"`
	Tipo     string   `json:"tipo"` // inteiro | decimal | lista_texto | opcao
	Unidade  string   `json:"unidade,omitempty"`
	Minimo   *float64 `json:"minimo,omitempty"`
	Maximo   *float64 `json:"maximo,omitempty"`
	Opcoes   []string `json:"opcoes,omitempty"`
	Opcional bool     `json:"opcional,omitempty"`
}

// ItemCatalogo descreve uma regra conhecida pelo motor.
type ItemCatalogo struct {
	Codigo     string              `json:"codigo"`
	Titulo     string              `json:"titulo"`
	Descricao  string              `json:"descricao"`
	Parametros []ParametroCatalogo `json:"parametros"`
}

func num(f float64) *float64 { return &f }

var (
	tiposRecebivel = []string{"Duplicata", "Duplicata de Serviço", "Cheque", "CCB", "Nota Comercial"}
	situacoes      = []string{"ATIVA", "SUSPENSA", "INAPTA", "BAIXADA", "NULA"}
	agrupamentos   = []string{"RAIZ_CNPJ", "CNPJ"}
)

// Catalogo lista as 9 regras que o motor sabe executar, na ordem do JSON original.
func Catalogo() []ItemCatalogo {
	return []ItemCatalogo{
		{Codigo: "CNPJ_SACADO_VALIDO", Titulo: "CNPJ do cliente válido", Descricao: "O CNPJ de quem paga a nota precisa ter dígitos verificadores corretos.", Parametros: []ParametroCatalogo{}},
		{Codigo: "SITUACAO_CADASTRAL_SACADO", Titulo: "Situação cadastral do cliente", Descricao: "Só compra notas de clientes com estas situações na Receita.", Parametros: []ParametroCatalogo{
			{Nome: "situacoes_aceitas", Rotulo: "Situações aceitas", Tipo: "lista_texto", Opcoes: situacoes},
		}},
		{Codigo: "TIPO_RECEBIVEL", Titulo: "Tipos de nota aceitos", Descricao: "Só os tipos marcados podem ser comprados.", Parametros: []ParametroCatalogo{
			{Nome: "tipos_aceitos", Rotulo: "Tipos aceitos", Tipo: "lista_texto", Opcoes: tiposRecebivel},
		}},
		{Codigo: "VALOR_MINIMO", Titulo: "Valor mínimo", Descricao: "Menor valor de face que o fundo compra.", Parametros: []ParametroCatalogo{
			{Nome: "valor_minimo", Rotulo: "Valor mínimo", Tipo: "decimal", Unidade: "R$", Minimo: num(0)},
		}},
		{Codigo: "NAO_VENCIDA", Titulo: "Só notas a vencer", Descricao: "Recusa notas com vencimento até a data-base.", Parametros: []ParametroCatalogo{}},
		{Codigo: "PRAZO_MAXIMO", Titulo: "Prazo máximo", Descricao: "Maior número de dias entre a data-base e o vencimento.", Parametros: []ParametroCatalogo{
			{Nome: "prazo_maximo_dias", Rotulo: "Prazo máximo", Tipo: "inteiro", Unidade: "dias", Minimo: num(1)},
		}},
		{Codigo: "CONCENTRACAO_MAXIMA_SACADO", Titulo: "Concentração máxima por cliente (sacado)", Descricao: "Parte máxima da carteira que um mesmo cliente pode ter.", Parametros: []ParametroCatalogo{
			{Nome: "percentual_maximo", Rotulo: "Limite", Tipo: "decimal", Unidade: "%", Minimo: num(0), Maximo: num(100)},
			{Nome: "agrupamento", Rotulo: "Agrupar por", Tipo: "opcao", Opcoes: agrupamentos, Opcional: true},
		}},
		{Codigo: "PRAZO_MINIMO", Titulo: "Prazo mínimo", Descricao: "Menor número de dias entre a data-base e o vencimento.", Parametros: []ParametroCatalogo{
			{Nome: "prazo_minimo_dias", Rotulo: "Prazo mínimo", Tipo: "inteiro", Unidade: "dias", Minimo: num(1)},
		}},
		{Codigo: "CONCENTRACAO_MAXIMA_CEDENTE", Titulo: "Concentração máxima por empresa (cedente)", Descricao: "Parte máxima da carteira que uma mesma empresa pode ter.", Parametros: []ParametroCatalogo{
			{Nome: "percentual_maximo", Rotulo: "Limite", Tipo: "decimal", Unidade: "%", Minimo: num(0), Maximo: num(100)},
			{Nome: "agrupamento", Rotulo: "Agrupar por", Tipo: "opcao", Opcoes: agrupamentos, Opcional: true},
		}},
	}
}

// CatalogoPrecificacao descreve os parâmetros de preço editáveis pelo gestor.
func CatalogoPrecificacao() []ParametroCatalogo {
	return []ParametroCatalogo{
		{Nome: "taxa_mensal_percentual", Rotulo: "Taxa", Tipo: "decimal", Unidade: "% ao mês", Minimo: num(0)},
		{Nome: "base_dias", Rotulo: "Base", Tipo: "inteiro", Unidade: "dias", Minimo: num(1)},
	}
}

// ItemPorCodigo devolve o item do catálogo, ou false se o código for desconhecido.
func ItemPorCodigo(codigo string) (ItemCatalogo, bool) {
	for _, it := range Catalogo() {
		if it.Codigo == codigo {
			return it, true
		}
	}
	return ItemCatalogo{}, false
}
