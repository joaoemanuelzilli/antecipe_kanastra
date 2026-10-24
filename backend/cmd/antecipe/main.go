// Comando antecipe: CLI no estilo Unix. Dados entram por stdin e saem por stdout; logs vão para stderr.
//
//	antecipe gerar --linhas 500000 --semente 42 --estoque data/estoque_antecipe.zip > lote.csv
//	antecipe exemplo --estoque data/estoque_antecipe.zip > data/exemplo.csv
//	antecipe validar --regras data/validadores_antecipe.json --estoque data/estoque_antecipe.zip < lote.csv
package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"antecipe/internal/config"
	"antecipe/internal/csvnota"
	"antecipe/internal/dominio"
	"antecipe/internal/fontes"
	"antecipe/internal/formato"
	"antecipe/internal/gerador"
	"antecipe/internal/motor"
	"antecipe/internal/obs"
	"antecipe/internal/precificacao"
	"antecipe/internal/regras"
	"antecipe/internal/textos"
)

const uso = "uso: antecipe <gerar|exemplo|validar> [opções]  (antecipe <comando> -h para ver as opções)"

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, uso)
		return 1
	}
	switch args[0] {
	case "gerar":
		return gerar(args[1:], stdout, stderr)
	case "exemplo":
		return exemplo(args[1:], stdout, stderr)
	case "validar":
		return validar(args[1:], stdin, stdout, stderr)
	}
	fmt.Fprintln(stderr, uso)
	return 1
}

func gerar(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gerar", flag.ContinueOnError)
	fs.SetOutput(stderr)
	linhas := fs.Int("linhas", 100000, "quantidade de notas")
	semente := fs.Int64("semente", 42, "semente (mesma semente, mesmo arquivo)")
	estoque := fs.String("estoque", "", "zip do estoque, para incluir notas já vendidas (opcional)")
	dataBase := fs.String("data-base", "2026-10-22", "data-base AAAA-MM-DD")
	fundoNome := fs.String("fundo-nome", "ANTECIPE FIDC", "nome do fundo")
	fundoCNPJ := fs.String("fundo-cnpj", "92.234.400/0001-80", "CNPJ do fundo")
	if fs.Parse(args) != nil {
		return 1
	}
	db, err := time.Parse("2006-01-02", *dataBase)
	if err != nil {
		fmt.Fprintln(stderr, "data-base inválida:", *dataBase)
		return 1
	}
	var est []dominio.Nota
	if *estoque != "" {
		if est, err = fontes.LerEstoqueZip(*estoque); err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
	}
	err = gerador.Gerar(stdout, gerador.Opcoes{Linhas: *linhas, Semente: *semente, DataBase: db, Estoque: est, FundoNome: *fundoNome, FundoCNPJ: *fundoCNPJ})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	return 0
}

func exemplo(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("exemplo", flag.ContinueOnError)
	fs.SetOutput(stderr)
	estoque := fs.String("estoque", "data/estoque_antecipe.zip", "zip do estoque")
	dataBase := fs.String("data-base", "2026-10-22", "data-base AAAA-MM-DD")
	if fs.Parse(args) != nil {
		return 1
	}
	db, err := time.Parse("2006-01-02", *dataBase)
	if err != nil {
		fmt.Fprintln(stderr, "data-base inválida:", *dataBase)
		return 1
	}
	est, err := fontes.LerEstoqueZip(*estoque)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if err := gerador.Exemplo(stdout, db, est); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	return 0
}

func validar(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("validar", flag.ContinueOnError)
	fs.SetOutput(stderr)
	arqRegras := fs.String("regras", "data/validadores_antecipe.json", "JSON de regras")
	arqEstoque := fs.String("estoque", "data/estoque_antecipe.zip", "zip do estoque")
	arqConfig := fs.String("config", "config/antecipe.yaml", "YAML de configuração")
	arqSituacao := fs.String("situacao", "data/situacao_cadastral.csv", "CSV de situação cadastral")
	dataBase := fs.String("data-base", "", "data-base AAAA-MM-DD (padrão: a do YAML)")
	if fs.Parse(args) != nil {
		return 1
	}
	log := obs.NovoLogger(stderr, "info")
	inicio := time.Now()

	b, err := os.ReadFile(*arqRegras)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	r, err := regras.Ler(b)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	cfg, err := config.Carregar(*arqConfig)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	db := cfg.DataBasePadrao()
	if *dataBase != "" {
		if db, err = time.Parse("2006-01-02", *dataBase); err != nil {
			fmt.Fprintln(stderr, "data-base inválida:", *dataBase)
			return 1
		}
	}
	pc, _ := cfg.PrecificacaoEfetiva(r.Precificacao)
	calc, err := precificacao.Nova(pc)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	est, err := fontes.LerEstoqueZip(*arqEstoque)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	sit, err := fontes.LerSituacoes(*arqSituacao)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	vendidas, ef := fontes.ContextoDeEstoque(est, r.Fundo.CNPJ)

	notas, err := csvnota.Ler(stdin, nil)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	s := motor.Validar(notas, r, motor.Contexto{
		DataBase: db, Situacao: fontes.SituacaoCom(sit, cfg.Processamento.SituacaoCadastralPadrao),
		Vendidas: vendidas, Estoque: ef, Preco: calc,
	})
	if err := escreverResultado(stdout, s); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	log.Info("lote validado",
		slog.Int("linhas", s.Totais.Linhas), slog.Int("aprovadas", s.Totais.Aprovadas), slog.Int("recusadas", s.Totais.Recusadas),
		slog.String("metodo_preco", pc.Metodo), slog.Int64("duracao_ms", time.Since(inicio).Milliseconds()))
	return 0
}

// escreverResultado devolve o CSV de entrada com as colunas do resultado no fim.
func escreverResultado(w io.Writer, s motor.Saida) error {
	bw := bufio.NewWriterSize(w, 1<<20)
	cw := csv.NewWriter(bw)
	cw.Comma = ';'
	cab := append(append([]string{}, csvnota.Colunas...), "STATUS", "MOTIVO", "O_QUE_FAZER", "DIAS", "DESCONTO", "VALOR_LIQUIDO")
	if err := cw.Write(cab); err != nil {
		return err
	}
	for _, rs := range s.Resultados {
		n := rs.Nota
		valor, venc, aq := "", "", ""
		if n.ErroFormato == nil {
			valor, venc = formato.ValorCSV(n.ValorCentavos), formato.Data(n.Vencimento)
			if !n.Aquisicao.IsZero() {
				aq = formato.Data(n.Aquisicao)
			}
		}
		status, motivo, acao, dias, desc, liq := "APROVADA", "", "", "", "", ""
		if n.ErroFormato == nil {
			dias = fmt.Sprint(rs.Dias)
		}
		if rs.Status == dominio.Recusada {
			t := textos.Recusa(rs.Recusas[0].Codigo, rs.Recusas[0].Dados)
			status, motivo, acao = "RECUSADA", t.Titulo+": "+t.Mensagem, t.Acao
		} else {
			desc, liq = formato.ValorCSV(rs.DescontoCentavos), formato.ValorCSV(rs.LiquidoCentavos)
		}
		c := formato.CelulaCSV
		linha := []string{c(n.NomeFundo), c(n.DocFundo), c(n.NomeCedente), c(n.DocCedente), c(n.NomeSacado), c(n.DocSacado), c(n.Numero), c(n.Tipo),
			valor, venc, aq, status, c(motivo), acao, dias, desc, liq}
		if err := cw.Write(linha); err != nil {
			return err
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return err
	}
	return bw.Flush()
}
