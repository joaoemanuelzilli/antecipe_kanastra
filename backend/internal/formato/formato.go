// Package formato formata valores para exibição em pt-BR.
package formato

import (
	"math"
	"strconv"
	"strings"
	"time"
)

// Reais formata centavos como moeda: 123456 → "R$ 1.234,56".
func Reais(centavos int64) string {
	sinal := ""
	if centavos < 0 {
		sinal = "-"
		centavos = -centavos
	}
	inteiro := centavos / 100
	resto := centavos % 100
	return sinal + "R$ " + milhar(strconv.FormatInt(inteiro, 10)) + "," + dois(resto)
}

// Data formata como dd/mm/aaaa.
func Data(t time.Time) string { return t.Format("02/01/2006") }

// DataISO converte "2026-12-21" em "21/12/2026"; se não for uma data ISO, devolve como veio.
func DataISO(s string) string {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return s
	}
	return Data(t)
}

// Inteiro formata com separador de milhar: 500000 → "500.000".
func Inteiro(n int) string {
	if n < 0 {
		return "-" + milhar(strconv.Itoa(-n))
	}
	return milhar(strconv.Itoa(n))
}

// Percentual formata com vírgula decimal: (12.612, 1) → "12,6%".
func Percentual(f float64, casas int) string {
	p := math.Pow(10, float64(casas))
	f = math.Round(f*p) / p
	return strings.Replace(strconv.FormatFloat(f, 'f', casas, 64), ".", ",", 1) + "%"
}

// Lista junta itens em português: ["A","B","C"] → "A, B e C".
func Lista(xs []string) string {
	switch len(xs) {
	case 0:
		return ""
	case 1:
		return xs[0]
	}
	return strings.Join(xs[:len(xs)-1], ", ") + " e " + xs[len(xs)-1]
}

func milhar(dig string) string {
	if len(dig) <= 3 {
		return dig
	}
	var b strings.Builder
	pre := len(dig) % 3
	if pre > 0 {
		b.WriteString(dig[:pre])
	}
	for i := pre; i < len(dig); i += 3 {
		if b.Len() > 0 {
			b.WriteByte('.')
		}
		b.WriteString(dig[i : i+3])
	}
	return b.String()
}

func dois(n int64) string {
	if n < 10 {
		return "0" + strconv.FormatInt(n, 10)
	}
	return strconv.FormatInt(n, 10)
}
