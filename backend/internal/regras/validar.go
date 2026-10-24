package regras

import (
	"fmt"
	"slices"
	"strings"

	"github.com/shopspring/decimal"

	"antecipe/internal/documento"
)

// ErroValidacao aponta um campo inválido das regras propostas.
type ErroValidacao struct {
	Campo    string `json:"campo"`
	Problema string `json:"problema"`
}

// ValidarSchema confere regras propostas pelo gestor antes de publicar.
func ValidarSchema(r Regras, metodos []string) []ErroValidacao {
	var errs []ErroValidacao
	add := func(campo, problema string) { errs = append(errs, ErroValidacao{Campo: campo, Problema: problema}) }

	if strings.TrimSpace(r.Fundo.Nome) == "" {
		add("fundo.nome", "não pode ficar vazio")
	}
	if !documento.CNPJValido(documento.Normalizar(r.Fundo.CNPJ)) {
		add("fundo.cnpj", "não é um CNPJ válido")
	}

	for i, v := range r.Validadores {
		base := fmt.Sprintf("validadores[%d]", i)
		item, ok := ItemPorCodigo(v.Codigo)
		if !ok {
			add(base+".codigo", "regra desconhecida: "+v.Codigo)
			continue
		}
		for _, pc := range item.Parametros {
			campo := base + ".parametros." + pc.Nome
			if _, presente := v.Parametros[pc.Nome]; !presente {
				if !pc.Opcional {
					add(campo, "parâmetro obrigatório ausente")
				}
				continue
			}
			if problema := validarParametro(v.Parametros, pc); problema != "" {
				add(campo, problema)
			}
		}
	}

	p := r.Precificacao
	if !slices.Contains(metodos, p.Metodo) {
		add("precificacao.metodo", "método desconhecido: "+p.Metodo+" (use "+strings.Join(metodos, " ou ")+")")
	}
	if p.TaxaMensalPercentual.IsNegative() {
		add("precificacao.taxa_mensal_percentual", "não pode ser negativa")
	}
	if p.BaseDias < 1 {
		add("precificacao.base_dias", "deve ser um número inteiro maior que zero")
	}
	return errs
}

func validarParametro(p Parametros, pc ParametroCatalogo) string {
	switch pc.Tipo {
	case "inteiro":
		n, err := p.Inteiro(pc.Nome)
		if err != nil || (pc.Minimo != nil && float64(n) < *pc.Minimo) {
			return "deve ser um número inteiro maior que zero"
		}
	case "decimal":
		d, err := p.Decimal(pc.Nome)
		if err != nil {
			return "deve ser um número"
		}
		if pc.Minimo != nil && d.LessThan(decimal.NewFromFloat(*pc.Minimo)) ||
			pc.Maximo != nil && d.GreaterThan(decimal.NewFromFloat(*pc.Maximo)) {
			if pc.Maximo != nil {
				return fmt.Sprintf("deve estar entre %g e %g", *pc.Minimo, *pc.Maximo)
			}
			return fmt.Sprintf("deve ser maior ou igual a %g", *pc.Minimo)
		}
	case "lista_texto":
		l, err := p.ListaTexto(pc.Nome)
		if err != nil {
			return "deve ser uma lista de textos"
		}
		if len(l) == 0 {
			return "deve ter pelo menos um item"
		}
		for _, x := range l {
			if len(pc.Opcoes) > 0 && !slices.Contains(pc.Opcoes, x) {
				return "valor não aceito: " + x
			}
		}
	case "opcao":
		s, err := p.Texto(pc.Nome)
		if err != nil || !slices.Contains(pc.Opcoes, s) {
			return "deve ser um de: " + strings.Join(pc.Opcoes, ", ")
		}
	}
	return ""
}
