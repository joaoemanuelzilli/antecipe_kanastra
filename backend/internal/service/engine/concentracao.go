package engine

import (
	"math"
	"math/bits"

	"antecipe/internal/domain"
	"antecipe/internal/domain/rules"
	"antecipe/internal/platform/document"
	"antecipe/internal/service/validator"
)

// Concentrar aplica as regras de concentração ativas (na ordem do JSON) às notas candidatas
// (as que passaram em todas as outras regras). A conta é estática e por fundo:
//
//	participação do grupo = (estoque do grupo + candidatas do grupo) ÷ (estoque do fundo + candidatas do fundo)
//
// Se passa do limite, todas as candidatas daquele grupo naquele fundo são recusadas.
// O fundo de cada nota é o DOC_FUNDO dela (ou fundoPadrao, se vier vazio).
// Também é usada na confirmação da cessão, sobre o estoque atual.
func Concentrar(cands []*domain.Resultado, r rules.Regras, estoques map[string]domain.EstoqueFundo, fundoPadrao string) []ErroConfig {
	var erros []ErroConfig
	fundoDe := func(n *domain.Nota) string {
		if f := document.Normalizar(n.DocFundo); f != "" {
			return f
		}
		return fundoPadrao
	}
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
		docDe := func(n *domain.Nota) (string, string) {
			if porSacado {
				return n.DocSacado, n.NomeSacado
			}
			return n.DocCedente, n.NomeCedente
		}

		type chaveGrupo struct{ fundo, grupo string }
		totLote := map[string]int64{}
		grpLote := map[chaveGrupo]int64{}
		var lista []*domain.Resultado
		for _, c := range cands {
			if len(c.Recusas) > 0 {
				continue
			}
			lista = append(lista, c)
			f := fundoDe(c.Nota)
			doc, _ := docDe(c.Nota)
			totLote[f] += c.Nota.ValorCentavos
			grpLote[chaveGrupo{f, grupoDe(doc)}] += c.Nota.ValorCentavos
		}

		// carteira de cada fundo agregada pelo grupo (calculada uma vez por fundo)
		baseGrupo := map[string]map[string]int64{}
		baseDo := func(f string) map[string]int64 {
			if m, ok := baseGrupo[f]; ok {
				return m
			}
			m := map[string]int64{}
			fonte := estoques[f].PorCedente
			if porSacado {
				fonte = estoques[f].PorSacado
			}
			for doc, valor := range fonte {
				m[grupoDe(doc)] += valor
			}
			baseGrupo[f] = m
			return m
		}

		limiteBP := pct.Shift(2).Round(0).IntPart() // 10% → 1000 pontos-base
		limite, _ := pct.Float64()
		barrados := map[chaveGrupo]float64{}
		for k, soma := range grpLote {
			num := baseDo(k.fundo)[k.grupo] + soma
			den := estoques[k.fundo].TotalCentavos + totLote[k.fundo]
			if excede(num, den, limiteBP) {
				barrados[k] = math.Round(float64(num)*10000/float64(den)) / 100
			}
		}
		for _, rs := range lista {
			doc, nome := docDe(rs.Nota)
			k := chaveGrupo{fundoDe(rs.Nota), grupoDe(doc)}
			if p, ok := barrados[k]; ok {
				rs.Recusas = append(rs.Recusas, domain.Recusa{Codigo: v.Codigo, Dados: map[string]any{
					"percentual_resultante": p, "limite": limite, "grupo": k.grupo, "grupo_nome": nome,
				}})
			}
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
