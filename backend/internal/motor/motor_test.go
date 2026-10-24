package motor

import (
	"encoding/json"
	"math/rand/v2"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"antecipe/internal/dominio"
	"antecipe/internal/precificacao"
	"antecipe/internal/regras"
)

var dataBase = time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)

const sacado = "22.144.961/0001-02"

func regrasOriginais(t testing.TB) regras.Regras {
	t.Helper()
	b, err := os.ReadFile("../../../data/validadores_antecipe.json")
	if err != nil {
		t.Fatal(err)
	}
	r, err := regras.Ler(b)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func contexto(t testing.TB) Contexto {
	t.Helper()
	calc, err := precificacao.Nova(precificacao.Config{Metodo: precificacao.Composto, TaxaMensalPercentual: decimal.NewFromInt(2), BaseDias: 30})
	if err != nil {
		t.Fatal(err)
	}
	return Contexto{
		DataBase: dataBase,
		Situacao: func(string) string { return "ATIVA" },
		Vendidas: map[string]Vendida{},
		Estoque:  EstoqueFundo{TotalCentavos: 1_000_000_000_000, PorSacado: map[string]int64{}, PorCedente: map[string]int64{}},
		Preco:    calc,
	}
}

func nota(linha int, mod func(*dominio.Nota)) dominio.Nota {
	n := dominio.Nota{
		Linha: linha, NomeFundo: "ANTECIPE FIDC", DocFundo: "92.234.400/0001-80",
		NomeCedente: "CEDENTE LTDA", DocCedente: "10.062.050/0001-57",
		NomeSacado: "SACADO LTDA", DocSacado: sacado,
		Numero: "N" + string(rune('A'+linha%26)) + time.Duration(linha).String(), Tipo: "Duplicata",
		ValorCentavos: 1000000, Vencimento: dataBase.AddDate(0, 0, 30),
	}
	if mod != nil {
		mod(&n)
	}
	return n
}

func codigos(r dominio.Resultado) []string {
	var cs []string
	for _, x := range r.Recusas {
		cs = append(cs, x.Codigo)
	}
	return cs
}

func TestAprovadaComposto(t *testing.T) {
	s := Validar([]dominio.Nota{nota(2, nil)}, regrasOriginais(t), contexto(t))
	r := s.Resultados[0]
	if r.Status != dominio.Aprovada || r.DescontoCentavos != 19608 || r.LiquidoCentavos != 980392 || r.Dias != 30 {
		t.Fatalf("resultado = %+v (recusas %v)", r, codigos(r))
	}
	if s.Totais.LiquidoCentavos != 980392 || s.Totais.Aprovadas != 1 || s.Totais.FaceCentavos != 1000000 || s.Totais.Linhas != 1 {
		t.Fatalf("totais = %+v", s.Totais)
	}
}

func TestRegraInativaIgnorada(t *testing.T) {
	s := Validar([]dominio.Nota{nota(2, func(n *dominio.Nota) { n.Vencimento = dataBase.AddDate(0, 0, 5) })}, regrasOriginais(t), contexto(t))
	if s.Resultados[0].Status != dominio.Aprovada {
		t.Fatalf("recusas = %v", codigos(s.Resultados[0]))
	}
}

func TestCodigoDesconhecido(t *testing.T) {
	r := regrasOriginais(t)
	r.Validadores = append(r.Validadores, regras.Validador{Codigo: "REGRA_X", Ativo: true, Parametros: regras.Parametros{}})
	s := Validar([]dominio.Nota{nota(2, func(n *dominio.Nota) { n.Vencimento = dataBase.AddDate(0, 0, 240) })}, r, contexto(t))
	if len(s.ErrosConfig) != 1 || s.ErrosConfig[0].Codigo != "REGRA_X" || s.ErrosConfig[0].Mensagem != "Regra REGRA_X não existe e foi ignorada." {
		t.Fatalf("erros = %+v", s.ErrosConfig)
	}
	if cs := codigos(s.Resultados[0]); len(cs) != 1 || cs[0] != "PRAZO_MAXIMO" {
		t.Fatalf("recusas = %v", cs)
	}
}

func TestParametroInvalidoViraErroConfig(t *testing.T) {
	r := regrasOriginais(t)
	for i := range r.Validadores {
		if r.Validadores[i].Codigo == "PRAZO_MAXIMO" {
			r.Validadores[i].Parametros["prazo_maximo_dias"] = json.RawMessage(`"abc"`)
		}
	}
	s := Validar([]dominio.Nota{nota(2, nil), nota(3, nil)}, r, contexto(t))
	if len(s.ErrosConfig) != 1 || s.ErrosConfig[0].Codigo != "PRAZO_MAXIMO" {
		t.Fatalf("erros = %+v", s.ErrosConfig)
	}
}

func TestFormatoInvalido(t *testing.T) {
	n := dominio.Nota{Linha: 7, ErroFormato: &dominio.ErroCampo{Campo: "VALOR_NOMINAL", Valor: "abc", Problema: "não é um valor válido (use 1234,56)"}}
	s := Validar([]dominio.Nota{n}, regrasOriginais(t), contexto(t))
	r := s.Resultados[0]
	if r.Status != dominio.Recusada || len(r.Recusas) != 1 || r.Recusas[0].Codigo != "FORMATO_INVALIDO" || r.Recusas[0].Dados["linha"] != 7 {
		t.Fatalf("resultado = %+v", r)
	}
	if s.Motivos["FORMATO_INVALIDO"] != 1 {
		t.Fatalf("motivos = %v", s.Motivos)
	}
}

func TestDuplicidadeEstoque(t *testing.T) {
	n := nota(2, nil)
	c := contexto(t)
	c.Vendidas[Chave(&n)] = Vendida{Em: time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC), ValorCentavos: 1000000}
	s := Validar([]dominio.Nota{n}, regrasOriginais(t), c)
	r := s.Resultados[0]
	if r.Status != dominio.Recusada || r.Recusas[0].Codigo != "DUPLICIDADE" || r.Recusas[0].Dados["vendida_em"] != "2026-04-27" {
		t.Fatalf("resultado = %+v", r)
	}
}

func TestDuplicadaNoArquivo(t *testing.T) {
	a := nota(2, nil)
	b := a
	b.Linha = 3
	s := Validar([]dominio.Nota{a, b}, regrasOriginais(t), contexto(t))
	if s.Resultados[0].Status != dominio.Aprovada {
		t.Fatalf("primeira = %v", codigos(s.Resultados[0]))
	}
	r := s.Resultados[1]
	if r.Recusas[0].Codigo != "DUPLICADA_NO_ARQUIVO" || r.Recusas[0].Dados["linha_original"] != 2 {
		t.Fatalf("segunda = %+v", r.Recusas)
	}
}

func TestParcelasNaoSaoDuplicadas(t *testing.T) {
	a := nota(2, func(n *dominio.Nota) { n.Numero = "1-767495"; n.Vencimento = time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC) })
	b := nota(3, func(n *dominio.Nota) { n.Numero = "1-767495"; n.Vencimento = time.Date(2026, 12, 15, 0, 0, 0, 0, time.UTC) })
	s := Validar([]dominio.Nota{a, b}, regrasOriginais(t), contexto(t))
	for _, r := range s.Resultados {
		if r.Status != dominio.Aprovada {
			t.Fatalf("parcela recusada: %v", codigos(r))
		}
	}
}

func TestDuplicidadeVemPrimeiro(t *testing.T) {
	n := nota(2, func(n *dominio.Nota) { n.Vencimento = dataBase.AddDate(0, 0, 240) })
	c := contexto(t)
	c.Vendidas[Chave(&n)] = Vendida{Em: dataBase, ValorCentavos: n.ValorCentavos}
	cs := codigos(Validar([]dominio.Nota{n}, regrasOriginais(t), c).Resultados[0])
	if len(cs) != 2 || cs[0] != "DUPLICIDADE" || cs[1] != "PRAZO_MAXIMO" {
		t.Fatalf("recusas = %v", cs)
	}
}

func contextoConcentracao(t *testing.T, grupo int64) Contexto {
	c := contexto(t)
	c.Estoque = EstoqueFundo{TotalCentavos: 100_000_000, PorSacado: map[string]int64{"22144961000102": grupo}, PorCedente: map[string]int64{}}
	return c
}

func TestConcentracaoDeterministica(t *testing.T) {
	base := []dominio.Nota{
		nota(2, func(n *dominio.Nota) { n.Numero = "A"; n.ValorCentavos = 500_000; n.Vencimento = dataBase.AddDate(0, 0, 10) }),
		nota(3, func(n *dominio.Nota) { n.Numero = "B"; n.ValorCentavos = 500_000; n.Vencimento = dataBase.AddDate(0, 0, 20) }),
		nota(4, func(n *dominio.Nota) { n.Numero = "C"; n.ValorCentavos = 500_000; n.Vencimento = dataBase.AddDate(0, 0, 30) }),
	}
	ordens := [][]int{{0, 1, 2}, {2, 1, 0}, {1, 2, 0}, {2, 0, 1}, {1, 0, 2}}
	for _, ordem := range ordens {
		entrada := []dominio.Nota{base[ordem[0]], base[ordem[1]], base[ordem[2]]}
		s := Validar(entrada, regrasOriginais(t), contextoConcentracao(t, 9_000_000))
		for _, r := range s.Resultados {
			switch r.Nota.Numero {
			case "A", "B":
				if r.Status != dominio.Aprovada {
					t.Fatalf("ordem %v: %s recusada %v", ordem, r.Nota.Numero, codigos(r))
				}
			case "C":
				if r.Status != dominio.Recusada || r.Recusas[0].Codigo != "CONCENTRACAO_MAXIMA_SACADO" {
					t.Fatalf("ordem %v: C = %v", ordem, codigos(r))
				}
				d := r.Recusas[0].Dados
				if d["percentual_resultante"] != 10.34 || d["limite"] != 10.0 || d["grupo"] != "22144961" || d["grupo_nome"] != "SACADO LTDA" {
					t.Fatalf("dados = %v", d)
				}
			}
		}
	}
}

func TestConcentracaoJaEstourada(t *testing.T) {
	s := Validar([]dominio.Nota{nota(2, func(n *dominio.Nota) { n.ValorCentavos = 500_000 })}, regrasOriginais(t), contextoConcentracao(t, 12_500_000))
	r := s.Resultados[0]
	if r.Status != dominio.Recusada || r.Recusas[0].Codigo != "CONCENTRACAO_MAXIMA_SACADO" {
		t.Fatalf("recusas = %v", codigos(r))
	}
}

func TestAgrupamentoRaiz(t *testing.T) {
	// estoque na filial 0001 com 9,9%; nota da filial 0002 (mesma raiz) passa do limite
	c := contexto(t)
	c.Estoque = EstoqueFundo{TotalCentavos: 100_000_000, PorSacado: map[string]int64{"22144961000102": 9_900_000}, PorCedente: map[string]int64{}}
	n := nota(2, func(n *dominio.Nota) { n.DocSacado = "22.144.961/0002-93"; n.ValorCentavos = 500_000 })
	r := Validar([]dominio.Nota{n}, regrasOriginais(t), c).Resultados[0]
	if r.Status != dominio.Recusada || r.Recusas[0].Codigo != "CONCENTRACAO_MAXIMA_SACADO" {
		t.Fatalf("recusas = %v", codigos(r))
	}
}

func TestProgresso(t *testing.T) {
	c := contexto(t)
	var mu sync.Mutex
	etapas := map[string]bool{}
	c.AoAvancar = func(etapa string, feitas int) {
		mu.Lock()
		etapas[etapa] = true
		mu.Unlock()
	}
	notas := make([]dominio.Nota, 10000)
	for i := range notas {
		notas[i] = nota(i+2, func(n *dominio.Nota) { n.Numero = time.Duration(i).String() })
	}
	Validar(notas, regrasOriginais(t), c)
	for _, e := range []string{"validando", "concentracao", "precificando"} {
		if !etapas[e] {
			t.Errorf("etapa %s não reportada (%v)", e, etapas)
		}
	}
}

// cnpj gera um CNPJ válido a partir de 12 caracteres (só para dados sintéticos de teste).
func cnpj(base string) string {
	calc := func(s string, w []int) byte {
		t := 0
		for i := range s {
			t += int(s[i]-'0') * w[i]
		}
		if r := t % 11; r >= 2 {
			return byte('0' + 11 - r)
		}
		return '0'
	}
	w1 := []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	w2 := append([]int{6}, w1...)
	b := base + string(calc(base, w1))
	return b + string(calc(b, w2))
}

func BenchmarkValidar500k(b *testing.B) {
	rng := rand.New(rand.NewPCG(1, 2))
	sacados := make([]string, 2000)
	for i := range sacados {
		sacados[i] = cnpj(time.Duration(0).String()[:0] + padNum(10_000_000+i, 8) + "0001")
	}
	notas := make([]dominio.Nota, 500_000)
	for i := range notas {
		notas[i] = dominio.Nota{
			Linha: i + 2, NomeSacado: "S", DocSacado: sacados[rng.IntN(len(sacados))], DocCedente: "10.062.050/0001-57",
			Numero: padNum(i, 8), Tipo: "Duplicata", ValorCentavos: int64(15_000 + rng.IntN(5_000_000)),
			Vencimento: dataBase.AddDate(0, 0, 5+rng.IntN(200)),
		}
	}
	r := regrasOriginais(b)
	c := contexto(b)
	c.Estoque = EstoqueFundo{TotalCentavos: 5_307_402_800, PorSacado: map[string]int64{}, PorCedente: map[string]int64{}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Validar(notas, r, c)
	}
}

func padNum(n, largura int) string {
	s := []byte{}
	for n > 0 || len(s) == 0 {
		s = append([]byte{byte('0' + n%10)}, s...)
		n /= 10
	}
	for len(s) < largura {
		s = append([]byte{'0'}, s...)
	}
	return string(s)
}
