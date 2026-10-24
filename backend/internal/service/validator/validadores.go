// Package validadores tem as regras avaliadas nota a nota. Cada uma é uma função pura:
// recebe a nota, os parâmetros do JSON e o contexto, e diz se passou e com quais dados.
// As regras de concentração dependem do lote inteiro e ficam no motor.
package validator

import (
	"strings"
	"time"

	"antecipe/internal/domain"
	"antecipe/internal/domain/rules"
	"antecipe/internal/platform/document"
)

// Ctx é o contexto comum a todas as notas de um lote.
type Ctx struct {
	DataBase time.Time
	Situacao func(cnpjNorm string) string // já aplica a situação padrão para CNPJ desconhecido
}

// Item é a nota com os valores já calculados pelo motor.
type Item struct {
	Nota       *domain.Nota
	Dias       int
	SacadoNorm string
}

// Func avalia uma regra. err indica parâmetro inválido (erro de configuração, não recusa).
type Func func(it Item, p rules.Parametros, c Ctx) (ok bool, dados map[string]any, err error)

// PorNota mapeia o código do JSON para a função que o executa.
var PorNota = map[string]Func{
	"CNPJ_SACADO_VALIDO":        cnpjSacadoValido,
	"SITUACAO_CADASTRAL_SACADO": situacaoCadastral,
	"TIPO_RECEBIVEL":            tipoRecebivel,
	"VALOR_MINIMO":              valorMinimo,
	"NAO_VENCIDA":               naoVencida,
	"PRAZO_MAXIMO":              prazoMaximo,
	"PRAZO_MINIMO":              prazoMinimo,
}

// DeLote são as regras avaliadas sobre o lote inteiro (no motor).
var DeLote = map[string]bool{"CONCENTRACAO_MAXIMA_SACADO": true, "CONCENTRACAO_MAXIMA_CEDENTE": true}

// Conhecido diz se o motor sabe executar o código.
func Conhecido(codigo string) bool {
	_, ok := PorNota[codigo]
	return ok || DeLote[codigo]
}

const iso = "2006-01-02"

func cnpjSacadoValido(it Item, _ rules.Parametros, _ Ctx) (bool, map[string]any, error) {
	if document.CNPJValido(it.SacadoNorm) {
		return true, nil, nil
	}
	return false, map[string]any{"documento": it.Nota.DocSacado}, nil
}

func situacaoCadastral(it Item, p rules.Parametros, c Ctx) (bool, map[string]any, error) {
	aceitas, err := p.ListaTexto("situacoes_aceitas")
	if err != nil {
		return false, nil, err
	}
	s := c.Situacao(it.SacadoNorm)
	for _, a := range aceitas {
		if strings.EqualFold(strings.TrimSpace(a), s) {
			return true, nil, nil
		}
	}
	return false, map[string]any{"situacao": s, "aceitas": aceitas}, nil
}

func tipoRecebivel(it Item, p rules.Parametros, _ Ctx) (bool, map[string]any, error) {
	aceitos, err := p.ListaTexto("tipos_aceitos")
	if err != nil {
		return false, nil, err
	}
	t := normalizarTexto(it.Nota.Tipo)
	for _, a := range aceitos {
		if normalizarTexto(a) == t {
			return true, nil, nil
		}
	}
	return false, map[string]any{"tipo": it.Nota.Tipo, "aceitos": aceitos}, nil
}

func valorMinimo(it Item, p rules.Parametros, _ Ctx) (bool, map[string]any, error) {
	minimo, err := p.Decimal("valor_minimo")
	if err != nil {
		return false, nil, err
	}
	minimoCentavos := minimo.Shift(2).Round(0).IntPart()
	if it.Nota.ValorCentavos >= minimoCentavos {
		return true, nil, nil
	}
	return false, map[string]any{"valor_centavos": it.Nota.ValorCentavos, "minimo_centavos": minimoCentavos}, nil
}

func naoVencida(it Item, _ rules.Parametros, c Ctx) (bool, map[string]any, error) {
	if it.Nota.Vencimento.After(c.DataBase) {
		return true, nil, nil
	}
	return false, map[string]any{"vencimento": it.Nota.Vencimento.Format(iso), "data_base": c.DataBase.Format(iso)}, nil
}

func prazoMaximo(it Item, p rules.Parametros, _ Ctx) (bool, map[string]any, error) {
	maximo, err := p.Inteiro("prazo_maximo_dias")
	if err != nil {
		return false, nil, err
	}
	if it.Dias <= maximo {
		return true, nil, nil
	}
	antecipavel := it.Nota.Vencimento.AddDate(0, 0, -maximo).Format(iso)
	return false, map[string]any{"dias": it.Dias, "maximo": maximo, "antecipavel_em": antecipavel}, nil
}

func prazoMinimo(it Item, p rules.Parametros, _ Ctx) (bool, map[string]any, error) {
	minimo, err := p.Inteiro("prazo_minimo_dias")
	if err != nil {
		return false, nil, err
	}
	if it.Dias >= minimo {
		return true, nil, nil
	}
	return false, map[string]any{"dias": it.Dias, "minimo": minimo}, nil
}

var semAcento = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c", "ñ", "n",
)

// normalizarTexto compara sem acento, sem diferenciar maiúsculas e sem espaços repetidos.
func normalizarTexto(s string) string {
	return semAcento.Replace(strings.Join(strings.Fields(strings.ToLower(s)), " "))
}
