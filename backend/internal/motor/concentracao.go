package motor

import (
	"math"
	"sort"

	"antecipe/internal/documento"
	"antecipe/internal/dominio"
	"antecipe/internal/regras"
	"antecipe/internal/validadores"
)

// Concentrar aplica as regras de concentração ativas (na ordem do JSON) sobre os candidatos
// sem recusa, em ordem determinística (vencimento, número, linha). Marca as recusas no lugar.
// Também é usada na confirmação da cessão, sobre o estoque atual.
func Concentrar(cands []*dominio.Resultado, r regras.Regras, est EstoqueFundo) []ErroConfig {
	var erros []ErroConfig
	for _, v := range r.Validadores {
		if !v.Ativo || !validadores.DeLote[v.Codigo] {
			continue
		}
		pct, err := v.Parametros.Decimal("percentual_maximo")
		if err != nil {
			erros = append(erros, ErroConfig{v.Codigo, "Regra " + v.Codigo + " tem parâmetro inválido e foi ignorada."})
			continue
		}
		agrupamento := "CNPJ"
		if _, presente := v.Parametros["agrupamento"]; presente {
			agrupamento, err = v.Parametros.Texto("agrupamento")
			if err != nil || (agrupamento != "RAIZ_CNPJ" && agrupamento != "CNPJ") {
				erros = append(erros, ErroConfig{v.Codigo, "Regra " + v.Codigo + " tem parâmetro inválido e foi ignorada."})
				continue
			}
		}
		porSacado := v.Codigo == "CONCENTRACAO_MAXIMA_SACADO"
		grupoDe := func(doc string) string {
			n := documento.Normalizar(doc)
			if agrupamento == "RAIZ_CNPJ" {
				return documento.Raiz(n)
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

		lista := make([]*dominio.Resultado, 0, len(cands))
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
			if num*10000 > limiteBP*den {
				rs.Recusas = append(rs.Recusas, dominio.Recusa{Codigo: v.Codigo, Dados: map[string]any{
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
