// Package csvnota lê o CSV de notas no formato pt-BR (";", decimal com vírgula, datas dd/mm/aaaa).
//
// Erro de estrutura (vazio, não-CSV, coluna faltando) derruba o arquivo inteiro.
// Erro dentro de uma linha não derruba nada: a linha volta com ErroFormato preenchido.
package csvnota

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"antecipe/internal/dominio"
)

// Colunas do formato do desafio, na ordem do arquivo de exemplo.
var Colunas = []string{"NOME_FUNDO", "DOC_FUNDO", "NOME_CEDENTE", "DOC_CEDENTE", "NOME_SACADO", "DOC_SACADO", "NU_DOCUMENTO", "TIPO_RECEBIVEL", "VALOR_NOMINAL", "DATA_VENCIMENTO_ORIGINAL", "DATA_AQUISICAO"}

const colunaOpcional = "DATA_AQUISICAO"

// ErroArquivo é um problema que impede ler o arquivo inteiro.
type ErroArquivo struct {
	Codigo   string    // ARQUIVO_VAZIO | CABECALHO_INVALIDO | ARQUIVO_NAO_CSV
	Mensagem string    // texto para mostrar ao usuário
	Detalhes []Detalhe // ex.: cada coluna faltando
}

// Detalhe aponta linha e campo de um problema.
type Detalhe struct {
	Linha    int    `json:"linha"`
	Campo    string `json:"campo"`
	Problema string `json:"problema"`
}

func (e *ErroArquivo) Error() string { return e.Codigo + ": " + e.Mensagem }

const intervaloProgresso = 10000

// Ler interpreta o CSV inteiro. aoAvancar (opcional) recebe o total de linhas lidas a cada 10.000 e no fim.
func Ler(r io.Reader, aoAvancar func(lidas int)) ([]dominio.Nota, error) {
	cr := novoLeitorCSV(r)
	idx, err := lerCabecalho(cr)
	if err != nil {
		return nil, err
	}

	notas := make([]dominio.Nota, 0, 4096)
	for {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		linha, _ := cr.FieldPos(0)
		if err != nil {
			var pe *csv.ParseError
			if errors.As(err, &pe) {
				linha = pe.StartLine
			}
			notas = append(notas, dominio.Nota{Linha: linha, ErroFormato: &dominio.ErroCampo{Campo: "linha", Problema: "não pôde ser lida como CSV"}})
			continue
		}
		if linhaEmBranco(rec) {
			continue
		}
		notas = append(notas, interpretar(rec, idx, linha))
		if aoAvancar != nil && len(notas)%intervaloProgresso == 0 {
			aoAvancar(len(notas))
		}
	}
	if len(notas) == 0 {
		return nil, &ErroArquivo{Codigo: "ARQUIVO_VAZIO", Mensagem: "O arquivo não tem nenhuma nota."}
	}
	if aoAvancar != nil && len(notas)%intervaloProgresso != 0 {
		aoAvancar(len(notas))
	}
	return notas, nil
}

// ValidarCabecalho confere só o começo do arquivo: se é texto e se tem as colunas esperadas.
func ValidarCabecalho(r io.Reader) *ErroArquivo {
	br := bufio.NewReaderSize(r, 64*1024)
	inicio, _ := br.Peek(512)
	if bytes.IndexByte(inicio, 0) >= 0 {
		return &ErroArquivo{Codigo: "ARQUIVO_NAO_CSV", Mensagem: "Esse arquivo não é CSV. Envie um .csv separado por ponto e vírgula."}
	}
	if _, err := lerCabecalho(novoLeitorCSV(br)); err != nil {
		var ea *ErroArquivo
		if errors.As(err, &ea) {
			return ea
		}
		return &ErroArquivo{Codigo: "ARQUIVO_NAO_CSV", Mensagem: "Não conseguimos ler o arquivo como CSV."}
	}
	return nil
}

// ContarLinhas conta as linhas de dados (sem o cabeçalho e sem linhas em branco). Serve para o percentual.
func ContarLinhas(r io.Reader) (int, error) {
	br := bufio.NewReaderSize(r, 256*1024)
	n := 0
	cheia := false // a linha atual já tem conteúdo
	for {
		trecho, err := br.ReadSlice('\n')
		if len(bytes.TrimSpace(trecho)) > 0 {
			cheia = true
		}
		fimDeLinha := len(trecho) > 0 && trecho[len(trecho)-1] == '\n'
		if fimDeLinha || (err != nil && !errors.Is(err, bufio.ErrBufferFull)) {
			if cheia {
				n++
			}
			cheia = false
		}
		if err != nil && !errors.Is(err, bufio.ErrBufferFull) {
			if errors.Is(err, io.EOF) {
				break
			}
			return 0, err
		}
	}
	if n > 0 {
		n-- // cabeçalho
	}
	return n, nil
}

func novoLeitorCSV(r io.Reader) *csv.Reader {
	cr := csv.NewReader(novoDecodificador(r))
	cr.Comma = ';'
	cr.LazyQuotes = true
	cr.FieldsPerRecord = -1
	cr.ReuseRecord = true
	return cr
}

func lerCabecalho(cr *csv.Reader) (map[string]int, error) {
	var rec []string
	for {
		r, err := cr.Read()
		if errors.Is(err, io.EOF) {
			return nil, &ErroArquivo{Codigo: "ARQUIVO_VAZIO", Mensagem: "O arquivo está vazio."}
		}
		if err != nil {
			return nil, &ErroArquivo{Codigo: "ARQUIVO_NAO_CSV", Mensagem: "Não conseguimos ler o arquivo como CSV."}
		}
		if !linhaEmBranco(r) {
			rec = r
			break
		}
	}
	idx := make(map[string]int, len(rec))
	for i, nome := range rec {
		idx[strings.ToUpper(strings.TrimSpace(nome))] = i
	}
	var faltando []string
	var detalhes []Detalhe
	for _, c := range Colunas {
		if _, ok := idx[c]; !ok && c != colunaOpcional {
			faltando = append(faltando, c)
			detalhes = append(detalhes, Detalhe{Linha: 1, Campo: c, Problema: "coluna não encontrada"})
		}
	}
	if len(faltando) > 0 {
		msg := "O arquivo não tem as colunas esperadas. Faltou " + faltando[0] + "."
		if len(faltando) > 1 {
			msg = "O arquivo não tem as colunas esperadas. Faltaram " + strings.Join(faltando[:len(faltando)-1], ", ") + " e " + faltando[len(faltando)-1] + "."
		}
		return nil, &ErroArquivo{Codigo: "CABECALHO_INVALIDO", Mensagem: msg, Detalhes: detalhes}
	}
	idx["#total"] = len(rec)
	return idx, nil
}

func linhaEmBranco(rec []string) bool {
	for _, c := range rec {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func interpretar(rec []string, idx map[string]int, linha int) dominio.Nota {
	n := dominio.Nota{Linha: linha}
	total := idx["#total"]
	if len(rec) != total && !sobraVazia(rec, total) {
		n.ErroFormato = &dominio.ErroCampo{Campo: "linha", Problema: "tem " + strconv.Itoa(len(rec)) + " colunas, esperávamos " + strconv.Itoa(total)}
		return n
	}
	campo := func(nome string) string {
		i, ok := idx[nome]
		if !ok || i >= len(rec) {
			return ""
		}
		return strings.TrimSpace(rec[i])
	}
	n.NomeFundo, n.DocFundo = campo("NOME_FUNDO"), campo("DOC_FUNDO")
	n.NomeCedente, n.DocCedente = campo("NOME_CEDENTE"), campo("DOC_CEDENTE")
	n.NomeSacado, n.DocSacado = campo("NOME_SACADO"), campo("DOC_SACADO")
	n.Numero, n.Tipo = campo("NU_DOCUMENTO"), campo("TIPO_RECEBIVEL")

	falhar := func(c, v, p string) dominio.Nota {
		n.ErroFormato = &dominio.ErroCampo{Campo: c, Valor: v, Problema: p}
		return n
	}
	for _, c := range Colunas {
		if c != colunaOpcional && campo(c) == "" {
			return falhar(c, "", "está vazio")
		}
	}
	valor := campo("VALOR_NOMINAL")
	centavos, ok := parseValor(valor)
	if !ok {
		return falhar("VALOR_NOMINAL", valor, "não é um valor válido (use 1234,56)")
	}
	if centavos <= 0 {
		return falhar("VALOR_NOMINAL", valor, "precisa ser maior que zero")
	}
	n.ValorCentavos = centavos

	venc := campo("DATA_VENCIMENTO_ORIGINAL")
	t, ok := parseData(venc)
	if !ok {
		return falhar("DATA_VENCIMENTO_ORIGINAL", venc, "não é uma data válida (use dd/mm/aaaa)")
	}
	n.Vencimento = t

	if aq := campo(colunaOpcional); aq != "" {
		t, ok := parseData(aq)
		if !ok {
			return falhar(colunaOpcional, aq, "não é uma data válida (use dd/mm/aaaa)")
		}
		n.Aquisicao = t
	}
	return n
}

// sobraVazia aceita ";" extras no fim da linha (colunas a mais, todas vazias).
func sobraVazia(rec []string, total int) bool {
	if len(rec) < total {
		return false
	}
	for _, c := range rec[total:] {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

// parseValor aceita "1234,56", "1.234,56", "1234", "R$ 10.000,00" e "1234.56". Devolve centavos.
func parseValor(s string) (int64, bool) {
	s = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "R$"))
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	if s == "" {
		return 0, false
	}
	inteiro, frac := s, ""
	switch strings.Count(s, ",") {
	case 0:
		// "1234.56" (ponto decimal) quando o ponto não forma grupo de milhar
		if i := strings.LastIndexByte(s, '.'); i >= 0 && strings.Count(s, ".") == 1 && len(s)-i-1 <= 2 {
			inteiro, frac = s[:i], s[i+1:]
			if frac == "" {
				return 0, false
			}
		}
	case 1:
		i := strings.IndexByte(s, ',')
		inteiro, frac = s[:i], s[i+1:]
		if frac == "" {
			return 0, false
		}
	default:
		return 0, false
	}
	if len(frac) > 2 || !soDigitos(frac) {
		return 0, false
	}
	if strings.Contains(inteiro, ".") {
		grupos := strings.Split(inteiro, ".")
		if len(grupos[0]) == 0 || len(grupos[0]) > 3 {
			return 0, false
		}
		for _, g := range grupos[1:] {
			if len(g) != 3 {
				return 0, false
			}
		}
		inteiro = strings.Join(grupos, "")
	}
	if inteiro == "" || !soDigitos(inteiro) || len(inteiro) > 15 {
		return 0, false
	}
	v, err := strconv.ParseInt(inteiro, 10, 64)
	if err != nil {
		return 0, false
	}
	for len(frac) < 2 {
		frac += "0"
	}
	f, _ := strconv.ParseInt(frac, 10, 64)
	c := v*100 + f
	if neg {
		c = -c
	}
	return c, true
}

func soDigitos(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func parseData(s string) (time.Time, bool) {
	for _, layout := range []string{"02/01/2006", "2/1/2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// decodificador entrega o arquivo em UTF-8: tira o BOM e converte de Windows-1252 as linhas
// que não forem UTF-8 válido (CSV salvo pelo Excel no Brasil costuma vir assim).
type decodificador struct {
	r       *bufio.Reader
	buf     []byte
	scratch []byte
	err     error
	inicio  bool
}

var bom = []byte{0xEF, 0xBB, 0xBF}

func novoDecodificador(r io.Reader) *decodificador {
	return &decodificador{r: bufio.NewReaderSize(r, 256*1024), inicio: true}
}

func (d *decodificador) Read(p []byte) (int, error) {
	for len(d.buf) == 0 {
		if d.err != nil {
			return 0, d.err
		}
		linha, err := d.r.ReadSlice('\n')
		if errors.Is(err, bufio.ErrBufferFull) {
			d.scratch = append(d.scratch[:0], linha...)
			for errors.Is(err, bufio.ErrBufferFull) {
				linha, err = d.r.ReadSlice('\n')
				d.scratch = append(d.scratch, linha...)
			}
			linha = d.scratch
		}
		if d.inicio {
			linha = bytes.TrimPrefix(linha, bom)
			d.inicio = false
		}
		if !utf8.Valid(linha) {
			linha = cp1252ParaUTF8(linha)
		}
		d.buf = linha
		d.err = err
	}
	n := copy(p, d.buf)
	d.buf = d.buf[n:]
	return n, nil
}

var cp1252 = [32]rune{
	0x20AC, 0x81, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021, 0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0x8D, 0x017D, 0x8F,
	0x90, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014, 0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0x9D, 0x017E, 0x0178,
}

func cp1252ParaUTF8(b []byte) []byte {
	out := make([]byte, 0, len(b)+len(b)/4)
	for _, c := range b {
		switch {
		case c < 0x80:
			out = append(out, c)
		case c < 0xA0:
			out = utf8.AppendRune(out, cp1252[c-0x80])
		default:
			out = utf8.AppendRune(out, rune(c))
		}
	}
	return out
}
