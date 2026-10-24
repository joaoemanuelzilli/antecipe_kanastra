// Package fontes lê os arquivos de dados do desafio (estoque em zip e situação cadastral simulada)
// e monta o contexto que o motor usa. Serve à CLI e à semente do banco.
package fontes

import (
	"archive/zip"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"antecipe/internal/csvnota"
	"antecipe/internal/documento"
	"antecipe/internal/dominio"
	"antecipe/internal/motor"
)

// LerEstoqueZip lê o primeiro .csv de dentro do zip.
func LerEstoqueZip(caminho string) ([]dominio.Nota, error) {
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
		return csvnota.Ler(rc, nil)
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
		m[documento.Normalizar(rec[0])] = strings.ToUpper(strings.TrimSpace(rec[1]))
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
func ContextoDeEstoque(estoque []dominio.Nota, fundoCNPJ string) (map[string]motor.Vendida, motor.EstoqueFundo) {
	fundo := documento.Normalizar(fundoCNPJ)
	vendidas := make(map[string]motor.Vendida, len(estoque))
	est := motor.EstoqueFundo{PorSacado: map[string]int64{}, PorCedente: map[string]int64{}}
	for i := range estoque {
		n := &estoque[i]
		if n.ErroFormato != nil {
			continue
		}
		vendidas[motor.Chave(n)] = motor.Vendida{Em: n.Aquisicao, ValorCentavos: n.ValorCentavos}
		if documento.Normalizar(n.DocFundo) == fundo {
			est.TotalCentavos += n.ValorCentavos
			est.PorSacado[documento.Normalizar(n.DocSacado)] += n.ValorCentavos
			est.PorCedente[documento.Normalizar(n.DocCedente)] += n.ValorCentavos
		}
	}
	return vendidas, est
}
