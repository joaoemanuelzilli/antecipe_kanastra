package main

import (
	"bytes"
	"strings"
	"testing"
)

const (
	dados     = "../../../data/"
	arqConfig = "../../../config/antecipe.yaml"
)

func TestUsoInvalido(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"voar"}, strings.NewReader(""), &out, &errOut); code != 1 {
		t.Fatalf("código = %d", code)
	}
	if !strings.Contains(errOut.String(), "uso:") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestGerarEValidar(t *testing.T) {
	var lote, errOut bytes.Buffer
	if code := run([]string{"gerar", "--linhas", "500", "--semente", "7"}, nil, &lote, &errOut); code != 0 {
		t.Fatalf("gerar = %d (%s)", code, errOut.String())
	}
	if n := strings.Count(lote.String(), "\n"); n != 501 {
		t.Fatalf("linhas geradas = %d", n)
	}

	var out bytes.Buffer
	errOut.Reset()
	args := []string{"validar", "--regras", dados + "validadores_antecipe.json", "--estoque", dados + "estoque_antecipe.zip",
		"--config", arqConfig, "--situacao", dados + "situacao_cadastral.csv"}
	if code := run(args, &lote, &out, &errOut); code != 0 {
		t.Fatalf("validar = %d (%s)", code, errOut.String())
	}
	linhas := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(linhas) != 501 {
		t.Fatalf("linhas de saída = %d", len(linhas))
	}
	if !strings.HasSuffix(linhas[0], ";STATUS;MOTIVO;O_QUE_FAZER;DIAS;DESCONTO;VALOR_LIQUIDO") {
		t.Fatalf("cabeçalho = %q", linhas[0])
	}
	if !strings.Contains(out.String(), ";APROVADA;") || !strings.Contains(out.String(), ";RECUSADA;") {
		t.Fatal("saída sem APROVADA e RECUSADA")
	}
	// resumo no stderr só com contagens: nada de nomes ou documentos do arquivo
	if !strings.Contains(errOut.String(), "aprovadas") || strings.Contains(errOut.String(), "CLIENTE") || strings.Contains(errOut.String(), "/0001-") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestExemplo(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"exemplo", "--estoque", dados + "estoque_antecipe.zip"}, nil, &out, &errOut); code != 0 {
		t.Fatalf("exemplo = %d (%s)", code, errOut.String())
	}
	if !strings.Contains(out.String(), "LOJA AURORA LTDA") {
		t.Fatal("exemplo sem a LOJA AURORA")
	}
}

func TestValidarNaoInjetaFormula(t *testing.T) {
	entrada := "NOME_FUNDO;DOC_FUNDO;NOME_CEDENTE;DOC_CEDENTE;NOME_SACADO;DOC_SACADO;NU_DOCUMENTO;TIPO_RECEBIVEL;VALOR_NOMINAL;DATA_VENCIMENTO_ORIGINAL;DATA_AQUISICAO\n" +
		"ANTECIPE FIDC;92.234.400/0001-80;CED;10.062.050/0001-57;=HYPERLINK(\"http://x\");22.144.961/0001-02;1;Duplicata;1000,00;21/11/2026;\n"
	var out, errOut bytes.Buffer
	args := []string{"validar", "--regras", dados + "validadores_antecipe.json", "--estoque", dados + "estoque_antecipe.zip",
		"--config", arqConfig, "--situacao", dados + "situacao_cadastral.csv"}
	if code := run(args, strings.NewReader(entrada), &out, &errOut); code != 0 {
		t.Fatalf("validar = %d (%s)", code, errOut.String())
	}
	if strings.Contains(out.String(), ";=HYPERLINK") || !strings.Contains(out.String(), "'=HYPERLINK") {
		t.Fatalf("célula com fórmula não escapada: %s", out.String())
	}
}
