// Package export converte resultados para formatos de saída compartilhados (ex.: o do gabarito do desafio).
package export

import "antecipe/internal/domain"

// Gabarito é uma linha no formato de data/resultado_esperado_100k.json.
type Gabarito struct {
	Linha      int      `json:"linha"`
	ID         string   `json:"id"`
	Status     string   `json:"status"` // APROVADO | BARRADO
	BarradoPor []string `json:"barrado_por,omitempty"`
}

// ItemGabarito converte o resultado de uma nota.
func ItemGabarito(rs domain.Resultado) Gabarito {
	return ItemGabaritoDe(rs.Nota.Linha, rs.Nota.Numero, rs.Status, rs.Recusas)
}

// ItemGabaritoDe monta a linha a partir dos campos (usado também com o que vem do banco).
func ItemGabaritoDe(linha int, numero string, status domain.Status, recusas []domain.Recusa) Gabarito {
	g := Gabarito{Linha: linha, ID: numero, Status: "APROVADO"}
	if status == domain.Recusada {
		g.Status = "BARRADO"
		for _, r := range recusas {
			g.BarradoPor = append(g.BarradoPor, r.Codigo)
		}
	}
	return g
}
