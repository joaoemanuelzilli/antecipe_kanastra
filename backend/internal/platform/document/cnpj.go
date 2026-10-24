// Package documento normaliza e valida CNPJ, inclusive o alfanumérico (IN RFB 2.229/2024, desde jul/2026).
package document

import "strings"

var (
	pesos1 = []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	pesos2 = []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
)

// Normalizar tira pontuação e espaços e põe em maiúsculas: "12.abc.345/01de-35" → "12ABC34501DE35".
func Normalizar(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '.' || r == '/' || r == '-' || r == ' ' || r == '\t':
			continue
		case r >= 'a' && r <= 'z':
			b.WriteRune(r - 'a' + 'A')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// CNPJValido confere formato e dígitos verificadores de um CNPJ já normalizado.
// O valor de cada caractere é o código ASCII − 48, o que cobre o CNPJ numérico e o alfanumérico.
func CNPJValido(norm string) bool {
	if len(norm) != 14 {
		return false
	}
	todosIguais := true
	for i := 0; i < 14; i++ {
		c := norm[i]
		if i < 12 && !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z')) {
			return false
		}
		if i >= 12 && !(c >= '0' && c <= '9') {
			return false
		}
		if c != norm[0] {
			todosIguais = false
		}
	}
	if todosIguais {
		return false
	}
	return norm[12] == dv(norm[:12], pesos1) && norm[13] == dv(norm[:13], pesos2)
}

func dv(s string, pesos []int) byte {
	soma := 0
	for i := 0; i < len(s); i++ {
		soma += int(s[i]-'0') * pesos[i]
	}
	resto := soma % 11
	if resto < 2 {
		return '0'
	}
	return byte('0' + 11 - resto)
}

// ComDV completa 12 caracteres com os 2 dígitos verificadores (usado para gerar dados de teste).
func ComDV(base12 string) string {
	b := base12 + string(dv(base12, pesos1))
	return b + string(dv(b, pesos2))
}

// Raiz devolve os 8 primeiros caracteres (a raiz do CNPJ, que identifica a empresa).
func Raiz(norm string) string {
	if len(norm) < 8 {
		return norm
	}
	return norm[:8]
}

// Formatar aplica a máscara XX.XXX.XXX/XXXX-XX; se não tiver 14 caracteres, devolve como veio.
func Formatar(norm string) string {
	if len(norm) != 14 {
		return norm
	}
	return norm[:2] + "." + norm[2:5] + "." + norm[5:8] + "/" + norm[8:12] + "-" + norm[12:]
}
