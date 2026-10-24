// Package fontes lê os arquivos de dados do desafio (estoque em zip e situação cadastral simulada)
// e monta o contexto que o motor usa. Serve à CLI e à semente do banco.
package filesystem

import (
	"archive/zip"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"antecipe/internal/domain"
	"antecipe/internal/handler/csvfile"
	"antecipe/internal/platform/document"
)

// LerEstoqueZip lê o primeiro .csv de dentro do zip.
func LerEstoqueZip(caminho string) ([]domain.Nota, error) {
	z, err := zip.OpenReader(caminho)
	if err != nil {
		return nil, fmt.Errorf("estoque: %w", err)
	}
	defer z.Close()
	for _, f := range z.File {
		if !strings.HasSuffix(strings.ToLower(f.Name), ".csv") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("estoque: %w", err)
		}
		defer rc.Close()
		return csvfile.Ler(rc, nil)
	}
	return nil, errors.New("estoque: nenhum .csv dentro do zip")
}

// LerSituacoes lê o CSV "CNPJ;SITUACAO". A chave do mapa é o CNPJ normalizado.
func LerSituacoes(caminho string) (map[string]string, error) {
	f, err := os.Open(caminho)
	if err != nil {
		return nil, fmt.Errorf("situações: %w", err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = ';'
	r.FieldsPerRecord = 2
	m := map[string]string{}
	primeira := true
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("situações: %w", err)
		}
		if primeira {
			primeira = false
			if c := strings.ToUpper(strings.TrimSpace(rec[0])); c == "CNPJ" || c == "DOC" {
				continue
			}
		}
		m[document.Normalizar(rec[0])] = strings.ToUpper(strings.TrimSpace(rec[1]))
	}
	return m, nil
}

// SituacaoCom devolve a função de consulta usada pelo motor, com a situação padrão para CNPJ desconhecido.
func SituacaoCom(m map[string]string, padrao string) func(string) string {
	return func(cnpjNorm string) string {
		if s, ok := m[cnpjNorm]; ok {
			return s
		}
		return padrao
	}
}

// ContextoDeEstoque monta a carteira de cada fundo a partir do estoque. Nenhuma nota conta como
// "vendida pelo portal" (isso só existe depois de uma cessão); a venda dupla é barrada na cessão.
func ContextoDeEstoque(estoque []domain.Nota) domain.ContextoEstoque {
	c := domain.ContextoEstoque{Vendidas: map[string]domain.Vendida{}, Estoques: map[string]domain.EstoqueFundo{}, Itens: make(map[string]domain.ItemEstoque, len(estoque))}
	for i := range estoque {
		n := &estoque[i]
		if n.ErroFormato != nil {
			continue
		}
		it := domain.ItemEstoque{FundoDoc: document.Normalizar(n.DocFundo), SacadoDoc: document.Normalizar(n.DocSacado), CedenteDoc: document.Normalizar(n.DocCedente), ValorCentavos: n.ValorCentavos}
		AcumularEstoque(&c, domain.Chave(n), it)
	}
	return c
}

// AcumularEstoque soma uma nota à carteira do fundo dela.
func AcumularEstoque(c *domain.ContextoEstoque, chave string, it domain.ItemEstoque) {
	c.Itens[chave] = it
	e, ok := c.Estoques[it.FundoDoc]
	if !ok {
		e = domain.EstoqueFundo{PorSacado: map[string]int64{}, PorCedente: map[string]int64{}}
	}
	e.TotalCentavos += it.ValorCentavos
	e.PorSacado[it.SacadoDoc] += it.ValorCentavos
	e.PorCedente[it.CedenteDoc] += it.ValorCentavos
	c.Estoques[it.FundoDoc] = e
}
