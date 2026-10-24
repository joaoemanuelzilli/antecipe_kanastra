package csvnota

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"antecipe/internal/dominio"
)

const cab = "NOME_FUNDO;DOC_FUNDO;NOME_CEDENTE;DOC_CEDENTE;NOME_SACADO;DOC_SACADO;NU_DOCUMENTO;TIPO_RECEBIVEL;VALOR_NOMINAL;DATA_VENCIMENTO_ORIGINAL;DATA_AQUISICAO"

func linha(sacado, valor, venc, aq string) string {
	return "ANTECIPE FIDC;92.234.400/0001-80;NLR INDÚSTRIA DE ESTOFADOS LTDA ME;10.062.050/0001-57;" +
		sacado + ";22.144.961/0001-02;725714;Duplicata;" + valor + ";" + venc + ";" + aq
}

func ler(t *testing.T, conteudo string) []Nota {
	t.Helper()
	notas, err := Ler(strings.NewReader(conteudo), nil)
	if err != nil {
		t.Fatalf("Ler: %v", err)
	}
	return notas
}

type Nota = dominio.Nota

func data(a, m, d int) time.Time { return time.Date(a, time.Month(m), d, 0, 0, 0, 0, time.UTC) }

func TestLerValido(t *testing.T) {
	notas := ler(t, cab+"\n"+linha("TECIDOS SANTARÉM LTDA ME", "2278,08", "07/10/2026", "27/04/2026")+"\n")
	if len(notas) != 1 {
		t.Fatalf("len = %d", len(notas))
	}
	n := notas[0]
	if n.Linha != 2 || n.ValorCentavos != 227808 || !n.Vencimento.Equal(data(2026, 10, 7)) || !n.Aquisicao.Equal(data(2026, 4, 27)) {
		t.Fatalf("nota inesperada: %+v", n)
	}
	if n.NomeSacado != "TECIDOS SANTARÉM LTDA ME" || n.Numero != "725714" || n.Tipo != "Duplicata" || n.DocFundo != "92.234.400/0001-80" {
		t.Fatalf("campos inesperados: %+v", n)
	}
	if n.ErroFormato != nil {
		t.Fatalf("erro inesperado: %+v", n.ErroFormato)
	}
}

func TestLerBOMeCRLF(t *testing.T) {
	c := "\xEF\xBB\xBF" + cab + "\r\n" + linha("A", "1,00", "07/10/2026", "") + "\r\n" + linha("B", "2,00", "07/10/2026", "") + "\r\n"
	notas := ler(t, c)
	if len(notas) != 2 || notas[0].NomeFundo != "ANTECIPE FIDC" || notas[1].Aquisicao != (time.Time{}) || notas[1].ValorCentavos != 200 {
		t.Fatalf("notas: %+v", notas)
	}
}

func TestLerAspas(t *testing.T) {
	notas := ler(t, cab+"\n"+linha(`"MÓVEIS; SUL LTDA"`, "1,00", "07/10/2026", ""))
	if notas[0].NomeSacado != "MÓVEIS; SUL LTDA" || notas[0].ErroFormato != nil {
		t.Fatalf("nota: %+v", notas[0])
	}
}

func TestLerMilhar(t *testing.T) {
	notas := ler(t, cab+"\n"+linha("A", "1.234,56", "07/10/2026", "")+"\n"+linha("B", "R$ 10.000,00", "07/10/2026", "")+"\n"+linha("C", "1234", "07/10/2026", ""))
	if notas[0].ValorCentavos != 123456 || notas[1].ValorCentavos != 1000000 || notas[2].ValorCentavos != 123400 {
		t.Fatalf("valores: %d %d %d", notas[0].ValorCentavos, notas[1].ValorCentavos, notas[2].ValorCentavos)
	}
}

func TestLerDataInvalida(t *testing.T) {
	notas := ler(t, cab+"\n"+linha("A", "1,00", "31/02/2026", "")+"\n"+linha("B", "1,00", "07/10/2026", ""))
	e := notas[0].ErroFormato
	if e == nil || e.Campo != "DATA_VENCIMENTO_ORIGINAL" || e.Valor != "31/02/2026" || e.Problema != "não é uma data válida (use dd/mm/aaaa)" {
		t.Fatalf("erro: %+v", e)
	}
	if notas[1].ErroFormato != nil || notas[1].Linha != 3 {
		t.Fatalf("linha seguinte: %+v", notas[1])
	}
}

func TestLerValorNaoPositivo(t *testing.T) {
	for _, v := range []string{"0,00", "-5,00"} {
		notas := ler(t, cab+"\n"+linha("A", v, "07/10/2026", ""))
		e := notas[0].ErroFormato
		if e == nil || e.Campo != "VALOR_NOMINAL" || e.Problema != "precisa ser maior que zero" {
			t.Fatalf("valor %s: %+v", v, e)
		}
	}
}

func TestLerValorIlegivel(t *testing.T) {
	for _, v := range []string{"abc", "1,234", "1.2.3,4,5"} {
		notas := ler(t, cab+"\n"+linha("A", v, "07/10/2026", ""))
		e := notas[0].ErroFormato
		if e == nil || e.Campo != "VALOR_NOMINAL" || e.Problema != "não é um valor válido (use 1234,56)" {
			t.Fatalf("valor %s: %+v", v, e)
		}
	}
}

func TestLerColunasErradas(t *testing.T) {
	notas := ler(t, cab+"\nANTECIPE;x;y")
	e := notas[0].ErroFormato
	if e == nil || e.Campo != "linha" || e.Problema != "tem 3 colunas, esperávamos 11" {
		t.Fatalf("erro: %+v", e)
	}
}

func TestLerCampoVazio(t *testing.T) {
	notas := ler(t, cab+"\n"+strings.Replace(linha("A", "1,00", "07/10/2026", ""), "725714", "", 1))
	e := notas[0].ErroFormato
	if e == nil || e.Campo != "NU_DOCUMENTO" || e.Problema != "está vazio" {
		t.Fatalf("erro: %+v", e)
	}
}

func TestLerCabecalhoFaltando(t *testing.T) {
	c := strings.Replace(cab, ";DATA_VENCIMENTO_ORIGINAL", "", 1) + "\n"
	_, err := Ler(strings.NewReader(c), nil)
	var ea *ErroArquivo
	if !errors.As(err, &ea) || ea.Codigo != "CABECALHO_INVALIDO" || len(ea.Detalhes) != 1 || ea.Detalhes[0].Campo != "DATA_VENCIMENTO_ORIGINAL" {
		t.Fatalf("erro: %#v", err)
	}
}

func TestLerVazio(t *testing.T) {
	_, err := Ler(strings.NewReader(""), nil)
	var ea *ErroArquivo
	if !errors.As(err, &ea) || ea.Codigo != "ARQUIVO_VAZIO" {
		t.Fatalf("erro: %#v", err)
	}
}

func TestLerCabecalhoForaDeOrdem(t *testing.T) {
	c := " nu_documento ;NOME_FUNDO;DOC_FUNDO;NOME_CEDENTE;DOC_CEDENTE;NOME_SACADO;DOC_SACADO;TIPO_RECEBIVEL;valor_nominal;DATA_VENCIMENTO_ORIGINAL;OBS;DATA_AQUISICAO\n" +
		"999;ANTECIPE FIDC;92.234.400/0001-80;CED;10.062.050/0001-57;SAC;22.144.961/0001-02;Duplicata;10,50;07/10/2026;qualquer;\n"
	n := ler(t, c)[0]
	if n.Numero != "999" || n.NomeSacado != "SAC" || n.ValorCentavos != 1050 || n.ErroFormato != nil {
		t.Fatalf("nota: %+v", n)
	}
}

func TestLerWindows1252(t *testing.T) {
	c := cab + "\n" + linha("TECIDOS SANTAR\xC9M LTDA", "1,00", "07/10/2026", "")
	n := ler(t, c)[0]
	if n.NomeSacado != "TECIDOS SANTARÉM LTDA" {
		t.Fatalf("NomeSacado = %q", n.NomeSacado)
	}
	// o nome do cedente (UTF-8 válido no resto da linha não pode quebrar): a linha inteira é cp1252
	if n.ErroFormato != nil {
		t.Fatalf("erro: %+v", n.ErroFormato)
	}
}

func TestLerAquisicaoVazia(t *testing.T) {
	n := ler(t, cab+"\n"+linha("A", "1,00", "07/10/2026", ""))[0]
	if !n.Aquisicao.IsZero() || n.ErroFormato != nil {
		t.Fatalf("nota: %+v", n)
	}
}

func TestLerSemColunaAquisicao(t *testing.T) {
	c := strings.Replace(cab, ";DATA_AQUISICAO", "", 1) + "\n" +
		"ANTECIPE FIDC;92.234.400/0001-80;CED;10.062.050/0001-57;SAC;22.144.961/0001-02;1;Duplicata;1,00;07/10/2026\n"
	n := ler(t, c)[0]
	if n.ErroFormato != nil || n.ValorCentavos != 100 {
		t.Fatalf("nota: %+v", n)
	}
}

func TestLerProgresso(t *testing.T) {
	var b strings.Builder
	b.WriteString(cab + "\n")
	for i := 0; i < 25000; i++ {
		b.WriteString(linha("A", "1,00", "07/10/2026", "") + "\n")
	}
	var chamadas []int
	if _, err := Ler(strings.NewReader(b.String()), func(n int) { chamadas = append(chamadas, n) }); err != nil {
		t.Fatal(err)
	}
	if len(chamadas) < 2 || chamadas[0] != 10000 || chamadas[len(chamadas)-1] != 25000 {
		t.Fatalf("chamadas = %v", chamadas)
	}
}

func TestValidarCabecalho(t *testing.T) {
	if e := ValidarCabecalho(strings.NewReader("\x00\x01PNG")); e == nil || e.Codigo != "ARQUIVO_NAO_CSV" {
		t.Fatalf("binário: %#v", e)
	}
	if e := ValidarCabecalho(strings.NewReader(cab + "\n")); e != nil {
		t.Fatalf("válido: %#v", e)
	}
	if e := ValidarCabecalho(strings.NewReader("a;b;c\n")); e == nil || e.Codigo != "CABECALHO_INVALIDO" {
		t.Fatalf("sem colunas: %#v", e)
	}
	if e := ValidarCabecalho(strings.NewReader("")); e == nil || e.Codigo != "ARQUIVO_VAZIO" {
		t.Fatalf("vazio: %#v", e)
	}
}

func TestContarLinhas(t *testing.T) {
	c := cab + "\n" + linha("A", "1,00", "07/10/2026", "") + "\n" + linha("B", "1,00", "07/10/2026", "") + "\n" + linha("C", "1,00", "07/10/2026", "") + "\n\n"
	n, err := ContarLinhas(strings.NewReader(c))
	if err != nil || n != 3 {
		t.Fatalf("ContarLinhas = %d, %v", n, err)
	}
}

func BenchmarkLer500k(b *testing.B) {
	var sb strings.Builder
	sb.WriteString(cab + "\n")
	for i := 0; i < 500000; i++ {
		fmt.Fprintf(&sb, "ANTECIPE FIDC;92.234.400/0001-80;CEDENTE %d LTDA;10.062.050/0001-57;SACADO %d LTDA;22.144.961/0001-02;G-%08d;Duplicata;%d,%02d;07/10/2026;27/04/2026\n", i%50, i%2000, i, 150+i%50000, i%100)
	}
	conteudo := sb.String()
	b.SetBytes(int64(len(conteudo)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Ler(strings.NewReader(conteudo), nil); err != nil {
			b.Fatal(err)
		}
	}
}
