package engine

import (
	"math"
	"math/bits"
	"sort"

	"antecipe/internal/domain"
	"antecipe/internal/domain/rules"
	"antecipe/internal/platform/document"
	"antecipe/internal/service/validator"
)

// Concentrar aplica as regras de concentração ativas (na ordem do JSON) sobre os candidatos
// sem recusa, em ordem determinística (vencimento, número, linha). Marca as recusas no lugar.
// Também é usada na confirmação da cessão, sobre o estoque atual.
func Concentrar(cands []*domain.Resultado, r rules.Regras, est domain.EstoqueFundo) []ErroConfig {
	var erros []ErroConfig
	for _, v := range r.Validadores {
		if !v.Ativo || !validator.DeLote[v.Codigo] {
			continue
		}
		pct, err := v.Parametros.Decimal("percentual_maximo")
		if err != nil {
			erros = append(erros, ErroConfig{Codigo: v.Codigo, Mensagem: "Regra " + v.Codigo + " tem parâmetro inválido e foi ignorada."})
			continue
		}
		agrupamento := "CNPJ"
		if _, presente := v.Parametros["agrupamento"]; presente {
			agrupamento, err = v.Parametros.Texto("agrupamento")
			if err != nil || (agrupamento != "RAIZ_CNPJ" && agrupamento != "CNPJ") {
				erros = append(erros, ErroConfig{Codigo: v.Codigo, Mensagem: "Regra " + v.Codigo + " tem parâmetro inválido e foi ignorada."})
				continue
			}
		}
		porSacado := v.Codigo == "CONCENTRACAO_MAXIMA_SACADO"
		grupoDe := func(doc string) string {
			n := document.Normalizar(doc)
			if agrupamento == "RAIZ_CNPJ" {
				return document.Raiz(n)
			}
			return n
		}
		fonte := est.PorCedente
		if porSacado {
			fonte = est.PorSacado
		}
		base := make(map[string]int64, len(fonte))
		for doc, valor := range fonte {
			base[grupoDe(doc)] += valor
		}

		lista := make([]*domain.Resultado, 0, len(cands))
		for _, c := range cands {
			if len(c.Recusas) == 0 {
				lista = append(lista, c)
			}
		}
		sort.SliceStable(lista, func(i, j int) bool {
			a, b := lista[i].Nota, lista[j].Nota
			if !a.Vencimento.Equal(b.Vencimento) {
				return a.Vencimento.Before(b.Vencimento)
			}
			if a.Numero != b.Numero {
				return a.Numero < b.Numero
			}
			return a.Linha < b.Linha
		})

		limiteBP := pct.Shift(2).Round(0).IntPart() // 10% → 1000 pontos-base
		limite, _ := pct.Float64()
		var aceitoTotal int64
		aceitoGrupo := map[string]int64{}
		for _, rs := range lista {
			doc, nome := rs.Nota.DocCedente, rs.Nota.NomeCedente
			if porSacado {
				doc, nome = rs.Nota.DocSacado, rs.Nota.NomeSacado
			}
			g := grupoDe(doc)
			face := rs.Nota.ValorCentavos
			num := base[g] + aceitoGrupo[g] + face
			den := est.TotalCentavos + aceitoTotal + face
			if excede(num, den, limiteBP) {
				rs.Recusas = append(rs.Recusas, domain.Recusa{Codigo: v.Codigo, Dados: map[string]any{
					"percentual_resultante": math.Round(float64(num)*10000/float64(den)) / 100,
					"limite":                limite,
					"grupo":                 g,
					"grupo_nome":            nome,
				}})
				continue
			}
			aceitoGrupo[g] += face
			aceitoTotal += face
		}
	}
	return erros
}

// excede compara num/den > limiteBP/10000 em 128 bits: sem estouro mesmo com valores enormes.
func excede(num, den, limiteBP int64) bool {
	if num < 0 || den <= 0 || limiteBP < 0 {
		return true // entrada impossível: falha fechada (recusa)
	}
	ah, al := bits.Mul64(uint64(num), 10000)
	bh, bl := bits.Mul64(uint64(limiteBP), uint64(den))
	return ah > bh || (ah == bh && al > bl)
}
