package pricing

import (
	"sync"
	"testing"

	"github.com/shopspring/decimal"
)

func calc(t *testing.T, metodo string) *Calculadora {
	t.Helper()
	c, err := Nova(Config{Metodo: metodo, TaxaMensalPercentual: decimal.NewFromInt(2), BaseDias: 30})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestComposto(t *testing.T) {
	c := calc(t, Composto)
	casos := []struct {
		dias              int
		desconto, liquido int64
	}{
		{30, 19608, 980392},
		{60, 38831, 961169},
		{180, 112029, 887971},
	}
	for _, k := range casos {
		d, l := c.Calcular(1000000, k.dias)
		if d != k.desconto || l != k.liquido {
			t.Errorf("composto %d dias = (%d, %d), esperado (%d, %d)", k.dias, d, l, k.desconto, k.liquido)
		}
	}
}

func TestSimples(t *testing.T) {
	c := calc(t, Simples)
	casos := []struct {
		dias              int
		desconto, liquido int64
	}{
		{30, 20000, 980000},
		{60, 40000, 960000},
		{180, 120000, 880000},
	}
	for _, k := range casos {
		d, l := c.Calcular(1000000, k.dias)
		if d != k.desconto || l != k.liquido {
			t.Errorf("simples %d dias = (%d, %d), esperado (%d, %d)", k.dias, d, l, k.desconto, k.liquido)
		}
	}
}

func TestDiasNaoPositivos(t *testing.T) {
	d, l := calc(t, Composto).Calcular(1000000, 0)
	if d != 0 || l != 1000000 {
		t.Fatalf("(%d, %d)", d, l)
	}
}

func TestArredondamentoMeioParaCima(t *testing.T) {
	// 227808 × 2% × 1/30 = 151,872 → 152
	d, _ := calc(t, Simples).Calcular(227808, 1)
	if d != 152 {
		t.Fatalf("desconto = %d", d)
	}
}

func TestMetodoDesconhecido(t *testing.T) {
	if _, err := Nova(Config{Metodo: "X", TaxaMensalPercentual: decimal.NewFromInt(2), BaseDias: 30}); err == nil {
		t.Fatal("método desconhecido deveria falhar")
	}
	if _, err := Nova(Config{Metodo: Composto, TaxaMensalPercentual: decimal.NewFromInt(2), BaseDias: 0}); err == nil {
		t.Fatal("base 0 deveria falhar")
	}
}

func TestTexto(t *testing.T) {
	if got := calc(t, Composto).Texto(1000000, 30); got != "R$ 10.000,00 ÷ (1 + 2%)^(30 ÷ 30) = R$ 9.803,92 → desconto de R$ 196,08" {
		t.Errorf("composto: %q", got)
	}
	if got := calc(t, Simples).Texto(1000000, 30); got != "R$ 10.000,00 × 2% × (30 ÷ 30) = R$ 200,00 de desconto → você recebe R$ 9.800,00" {
		t.Errorf("simples: %q", got)
	}
}

func TestDescricao(t *testing.T) {
	c := Config{Metodo: Composto, TaxaMensalPercentual: decimal.RequireFromString("2.5"), BaseDias: 30}
	if c.Descricao() != "Juros compostos · 2,5% ao mês" {
		t.Errorf("composto: %q", c.Descricao())
	}
	c.Metodo = Simples
	if c.Descricao() != "Juros simples · 2,5% ao mês" {
		t.Errorf("simples: %q", c.Descricao())
	}
}

func TestEfetiva(t *testing.T) {
	base := Config{Metodo: Simples, TaxaMensalPercentual: decimal.NewFromInt(2), BaseDias: 30}
	m := Composto
	got, sobrescritos := Efetiva(base, &m, nil, nil)
	if got.Metodo != Composto || !got.TaxaMensalPercentual.Equal(decimal.NewFromInt(2)) || got.BaseDias != 30 {
		t.Fatalf("efetiva = %+v", got)
	}
	if len(sobrescritos) != 1 || sobrescritos[0] != "metodo" {
		t.Fatalf("sobrescritos = %v", sobrescritos)
	}
	taxa, b := decimal.NewFromInt(3), 360
	got, sobrescritos = Efetiva(base, nil, &taxa, &b)
	if got.Metodo != Simples || !got.TaxaMensalPercentual.Equal(taxa) || got.BaseDias != 360 || len(sobrescritos) != 2 {
		t.Fatalf("efetiva = %+v %v", got, sobrescritos)
	}
}

func TestCalcularConcorrente(t *testing.T) {
	c := calc(t, Composto)
	var wg sync.WaitGroup
	for g := 0; g < 32; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				c.Calcular(1000000, 1+(g*1000+i)%400)
			}
		}(g)
	}
	wg.Wait()
	if _, l := c.Calcular(1000000, 30); l != 980392 {
		t.Fatalf("liquido = %d", l)
	}
}
