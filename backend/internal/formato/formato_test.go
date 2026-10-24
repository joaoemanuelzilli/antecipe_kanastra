package formato

import (
	"testing"
	"time"
)

func TestReais(t *testing.T) {
	casos := map[int64]string{
		123456:      "R$ 1.234,56",
		980392:      "R$ 9.803,92",
		5:           "R$ 0,05",
		0:           "R$ 0,00",
		-500:        "-R$ 5,00",
		96419753200: "R$ 964.197.532,00",
	}
	for c, esperado := range casos {
		if got := Reais(c); got != esperado {
			t.Errorf("Reais(%d) = %q, esperado %q", c, got, esperado)
		}
	}
}

func TestData(t *testing.T) {
	d := time.Date(2026, 12, 21, 0, 0, 0, 0, time.UTC)
	if got := Data(d); got != "21/12/2026" {
		t.Fatalf("Data = %q", got)
	}
	if got := DataISO("2026-12-21"); got != "21/12/2026" {
		t.Fatalf("DataISO = %q", got)
	}
	if got := DataISO("lixo"); got != "lixo" {
		t.Fatalf("DataISO inválida = %q", got)
	}
}

func TestInteiro(t *testing.T) {
	casos := map[int]string{500000: "500.000", 1204: "1.204", 7: "7", -1234: "-1.234"}
	for n, esperado := range casos {
		if got := Inteiro(n); got != esperado {
			t.Errorf("Inteiro(%d) = %q, esperado %q", n, got, esperado)
		}
	}
}

func TestPercentual(t *testing.T) {
	if got := Percentual(12.612, 1); got != "12,6%" {
		t.Errorf("Percentual = %q", got)
	}
	if got := Percentual(10, 0); got != "10%" {
		t.Errorf("Percentual = %q", got)
	}
	if got := Percentual(2, 0); got != "2%" {
		t.Errorf("Percentual = %q", got)
	}
}

func TestLista(t *testing.T) {
	casos := []struct {
		xs  []string
		out string
	}{
		{[]string{"Duplicata", "Duplicata de Serviço"}, "Duplicata e Duplicata de Serviço"},
		{[]string{"A", "B", "C"}, "A, B e C"},
		{[]string{"ATIVA"}, "ATIVA"},
		{nil, ""},
	}
	for _, c := range casos {
		if got := Lista(c.xs); got != c.out {
			t.Errorf("Lista(%v) = %q, esperado %q", c.xs, got, c.out)
		}
	}
}

func TestValorCSV(t *testing.T) {
	casos := map[int64]string{227808: "2278,08", 5: "0,05", 1000000: "10000,00"}
	for c, esperado := range casos {
		if got := ValorCSV(c); got != esperado {
			t.Errorf("ValorCSV(%d) = %q, esperado %q", c, got, esperado)
		}
	}
}
