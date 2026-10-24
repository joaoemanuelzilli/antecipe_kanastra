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
			if strings.EqualFold(strings.TrimSpace(rec[0]), "CNPJ") {
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

// ContextoDeEstoque monta as notas já vendidas (todos os fundos) e as somas do fundo das regras.
func ContextoDeEstoque(estoque []domain.Nota, fundoCNPJ string) (map[string]domain.Vendida, domain.EstoqueFundo) {
	fundo := document.Normalizar(fundoCNPJ)
	vendidas := make(map[string]domain.Vendida, len(estoque))
	est := domain.EstoqueFundo{PorSacado: map[string]int64{}, PorCedente: map[string]int64{}}
	for i := range estoque {
		n := &estoque[i]
		if n.ErroFormato != nil {
			continue
		}
		vendidas[domain.Chave(n)] = domain.Vendida{Em: n.Aquisicao, ValorCentavos: n.ValorCentavos}
		if document.Normalizar(n.DocFundo) == fundo {
			est.TotalCentavos += n.ValorCentavos
			est.PorSacado[document.Normalizar(n.DocSacado)] += n.ValorCentavos
			est.PorCedente[document.Normalizar(n.DocCedente)] += n.ValorCentavos
		}
	}
	return vendidas, est
}
