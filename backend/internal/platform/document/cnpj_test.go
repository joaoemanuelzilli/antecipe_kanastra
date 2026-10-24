package document

import "testing"

func TestCNPJValido(t *testing.T) {
	casos := []struct {
		entrada string
		valido  bool
	}{
		{"92.234.400/0001-80", true},
		{"12.ABC.345/01DE-35", true}, // alfanumérico (desde jul/2026)
		{"12.abc.345/01de-35", true},
		{"11.122.233/0001-83", true},
		{"77.788.899/0001-80", false}, // DV errado
		{"11.111.111/1111-11", false}, // todos iguais
		{"123", false},
		{"", false},
		{"12.ABC.345/01DE-3A", false}, // DV precisa ser dígito
	}
	for _, c := range casos {
		if got := CNPJValido(Normalizar(c.entrada)); got != c.valido {
			t.Errorf("CNPJValido(%q) = %v, esperado %v", c.entrada, got, c.valido)
		}
	}
}

func TestNormalizar(t *testing.T) {
	if got := Normalizar(" 12.abc.345/01de-35 "); got != "12ABC34501DE35" {
		t.Fatalf("Normalizar = %q", got)
	}
}

func TestRaiz(t *testing.T) {
	if got := Raiz("92234400000180"); got != "92234400" {
		t.Fatalf("Raiz = %q", got)
	}
	if got := Raiz("123"); got != "123" {
		t.Fatalf("Raiz curta = %q", got)
	}
}

func TestFormatar(t *testing.T) {
	if got := Formatar("12ABC34501DE35"); got != "12.ABC.345/01DE-35" {
		t.Fatalf("Formatar = %q", got)
	}
	if got := Formatar("123"); got != "123" {
		t.Fatalf("Formatar curto = %q", got)
	}
}

func TestComDV(t *testing.T) {
	if got := ComDV("111222330001"); got != "11122233000183" {
		t.Fatalf("ComDV = %q", got)
	}
	if got := ComDV("12ABC34501DE"); got != "12ABC34501DE35" {
		t.Fatalf("ComDV alfanumérico = %q", got)
	}
}
