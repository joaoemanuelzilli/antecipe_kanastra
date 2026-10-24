// Package gerador cria lotes de teste no formato do desafio: grandes (100k–500k linhas, para
// medir desempenho) e o pequeno exemplo de demonstração (uma recusa de cada tipo).
package gerador

import (
	"bufio"
	"fmt"
	"io"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"antecipe/internal/csvnota"
	"antecipe/internal/documento"
	"antecipe/internal/dominio"
)

// Opcoes do lote gerado.
type Opcoes struct {
	Linhas    int
	Semente   int64
	DataBase  time.Time
	Estoque   []dominio.Nota // opcional: permite gerar duplicatas do estoque
	FundoNome string
	FundoCNPJ string
}

const (
	cnpjInvalido    = "77.788.899/0001-80"
	cnpjConcentrado = "58.206.002/0001-11"
	nomeConcentrado = "SERRALHERIA VIOLETA S.A."
)

var situacoesMock = []string{"11.122.233/0001-83", "22.233.344/0001-83", "33.344.455/0001-83"}

type parte struct{ nome, doc string }

// Gerar escreve o cabeçalho e o.Linhas linhas, de forma determinística pela semente.
func Gerar(w io.Writer, o Opcoes) error {
	rng := rand.New(rand.NewPCG(uint64(o.Semente), 0x9e3779b97f4a7c15))
	sacados := make([]parte, 2000)
	for i := range sacados {
		sacados[i] = parte{fmt.Sprintf("CLIENTE %04d LTDA", i+1), documento.Formatar(documento.ComDV(fmt.Sprintf("3%07d0001", i)))}
	}
	cedentes := make([]parte, 50)
	for i := range cedentes {
		cedentes[i] = parte{fmt.Sprintf("EMPRESA %02d INDÚSTRIA LTDA", i+1), documento.Formatar(documento.ComDV(fmt.Sprintf("4%07d0001", i)))}
	}
	// notas do estoque que podem ser reenviadas (duplicidade)
	var doEstoque []*dominio.Nota
	for i := range o.Estoque {
		if o.Estoque[i].ErroFormato == nil {
			doEstoque = append(doEstoque, &o.Estoque[i])
		}
	}

	bw := bufio.NewWriterSize(w, 1<<20)
	bw.WriteString(strings.Join(csvnota.Colunas, ";") + "\n")
	geradas := make([]string, 0, 1024) // para repetições no próprio arquivo
	for i := 0; i < o.Linhas; i++ {
		ced := cedentes[rng.IntN(len(cedentes))]
		sac := sacados[rng.IntN(len(sacados))]
		tipo := "Duplicata"
		if rng.IntN(100) < 15 {
			tipo = "Duplicata de Serviço"
		}
		valor := int64(15_000 + rng.IntN(4_985_001))
		venc := o.DataBase.AddDate(0, 0, 5+rng.IntN(166))
		numero := fmt.Sprintf("G-%08d", i+1)
		vencTxt := venc.Format("02/01/2006")
		valorTxt := centavos(valor)

		switch sorteio := rng.IntN(100); {
		case sorteio < 3:
			vencTxt = o.DataBase.AddDate(0, 0, 181+rng.IntN(180)).Format("02/01/2006")
		case sorteio < 5:
			vencTxt = o.DataBase.AddDate(0, 0, -1-rng.IntN(60)).Format("02/01/2006")
		case sorteio < 7:
			valorTxt = centavos(int64(100 + rng.IntN(9_900)))
		case sorteio < 9:
			tipo = []string{"Cheque", "CCB"}[rng.IntN(2)]
		case sorteio < 11:
			sac.doc = cnpjInvalido
		case sorteio < 13:
			sac.doc = situacoesMock[rng.IntN(len(situacoesMock))]
		case sorteio < 16 && len(doEstoque) > 0:
			bw.WriteString(linhaDeNota(doEstoque[rng.IntN(len(doEstoque))]) + "\n")
			continue
		case sorteio < 18 && len(geradas) > 0:
			bw.WriteString(geradas[rng.IntN(len(geradas))] + "\n")
			continue
		case sorteio < 19:
			if rng.IntN(2) == 0 {
				vencTxt = "31/02/2026"
			} else {
				valorTxt = "abc"
			}
		case sorteio < 20:
			sac = parte{nomeConcentrado, cnpjConcentrado}
		}
		l := strings.Join([]string{o.FundoNome, o.FundoCNPJ, ced.nome, ced.doc, sac.nome, sac.doc, numero, tipo, valorTxt, vencTxt, ""}, ";")
		bw.WriteString(l + "\n")
		if len(geradas) < cap(geradas) {
			geradas = append(geradas, l)
		} else {
			geradas[rng.IntN(len(geradas))] = l
		}
	}
	return bw.Flush()
}

// Exemplo escreve o lote de demonstração: 12 aprovadas e uma recusa de cada tipo.
func Exemplo(w io.Writer, dataBase time.Time, estoque []dominio.Nota) error {
	const fundo, fundoDoc = "ANTECIPE FIDC", "92.234.400/0001-80"
	ced := parte{"NLR INDÚSTRIA DE ESTOFADOS LTDA ME", "10.062.050/0001-57"}
	d := func(dias int) string { return dataBase.AddDate(0, 0, dias).Format("02/01/2006") }
	l := func(sac parte, numero, tipo, valor, venc string) string {
		return strings.Join([]string{fundo, fundoDoc, ced.nome, ced.doc, sac.nome, sac.doc, numero, tipo, valor, venc, ""}, ";")
	}
	cnpj := func(i int) string { return documento.Formatar(documento.ComDV(fmt.Sprintf("5%07d0001", i))) }
	aurora := parte{"LOJA AURORA LTDA", "55.566.677/0001-83"}
	linhaAurora := l(aurora, "900001", "Duplicata", "10.000,00", d(30))

	linhas := []string{
		linhaAurora,
		l(parte{"MERCADO SOL LTDA", cnpj(1)}, "900002", "Duplicata", "4.500,00", d(45)),
		l(parte{"CAFÉ BRISA LTDA", cnpj(2)}, "900003", "Duplicata de Serviço", "2.300,00", d(60)),
		l(parte{"PADARIA CENTRAL LTDA", cnpj(3)}, "900004", "Duplicata", "1.250,90", d(15)),
		l(parte{"FARMÁCIA VIDA LTDA", cnpj(4)}, "900005", "Duplicata", "7.800,00", d(90)),
		l(parte{"AUTO PEÇAS RIO LTDA", cnpj(5)}, "900006", "Duplicata", "15.000,00", d(120)),
		l(parte{"PAPELARIA ESTRELA LTDA", cnpj(6)}, "900007", "Duplicata de Serviço", "980,00", d(20)),
		l(parte{"HORTIFRUTI VERDE LTDA", cnpj(7)}, "900008", "Duplicata", "3.333,33", d(75)),
		l(parte{"ÓTICA VISÃO LTDA", cnpj(8)}, "900009", "Duplicata", "6.120,00", d(150)),
		l(parte{"PET SHOP AMIGO LTDA", cnpj(9)}, "900010", "Duplicata", "540,00", d(10)),
		l(parte{"LIVRARIA SABER LTDA", cnpj(10)}, "900011", "Duplicata de Serviço", "12.400,00", d(180)),
		l(parte{"MÓVEIS PRIMAVERA LTDA", "66.677.788/0001-83"}, "900012", "Duplicata", "8.000,00", d(100)),
		// uma recusa de cada tipo
		l(parte{"COMERCIAL ALFA LTDA", cnpjInvalido}, "900013", "Duplicata", "3.000,00", d(40)),
		l(parte{"DISTRIBUIDORA BETA LTDA", "11.122.233/0001-83"}, "900014", "Duplicata", "3.000,00", d(40)),
		l(parte{"ATACADO GAMA LTDA", "22.233.344/0001-83"}, "900015", "Duplicata", "2.000,00", d(35)),
		l(parte{"CONFECÇÕES DELTA LTDA", cnpj(11)}, "900016", "Cheque", "2.500,00", d(30)),
		l(parte{"BAZAR ÉPSILON LTDA", cnpj(12)}, "900017", "Duplicata", "80,00", d(30)),
		l(parte{"LANCHONETE ZETA LTDA", cnpj(13)}, "900018", "Duplicata", "1.900,00", d(-10)),
		l(parte{"CONSTRUTORA ETA LTDA", cnpj(14)}, "900019", "Duplicata", "9.000,00", d(240)),
		l(parte{nomeConcentrado, cnpjConcentrado}, "900020", "Duplicata", "5.000,00", d(60)),
		linhaAurora,
		l(parte{"ARMAZÉM TETA LTDA", cnpj(15)}, "900021", "Duplicata", "1.000,00", "31/02/2026"),
	}
	if n := notaVendavel(estoque, dataBase); n != nil {
		linhas = append(linhas, linhaDeNota(n))
	}

	bw := bufio.NewWriter(w)
	bw.WriteString(strings.Join(csvnota.Colunas, ";") + "\n")
	for _, x := range linhas {
		bw.WriteString(x + "\n")
	}
	return bw.Flush()
}

// notaVendavel escolhe uma nota do ANTECIPE no estoque que, fora a duplicidade, seria aprovada.
func notaVendavel(estoque []dominio.Nota, dataBase time.Time) *dominio.Nota {
	for i := range estoque {
		n := &estoque[i]
		dias := n.Vencimento.Sub(dataBase).Hours() / 24
		if n.ErroFormato == nil && documento.Normalizar(n.DocFundo) == "92234400000180" &&
			dias >= 30 && dias <= 90 && n.ValorCentavos >= 100_000 && documento.Raiz(documento.Normalizar(n.DocSacado)) != "58206002" {
			return n
		}
	}
	return nil
}

func linhaDeNota(n *dominio.Nota) string {
	aq := ""
	if !n.Aquisicao.IsZero() {
		aq = n.Aquisicao.Format("02/01/2006")
	}
	return strings.Join([]string{n.NomeFundo, n.DocFundo, n.NomeCedente, n.DocCedente, n.NomeSacado, n.DocSacado,
		n.Numero, n.Tipo, centavos(n.ValorCentavos), n.Vencimento.Format("02/01/2006"), aq}, ";")
}

// centavos formata como no CSV do desafio: 227808 → "2278,08".
func centavos(c int64) string {
	s := strconv.FormatInt(c%100, 10)
	if len(s) == 1 {
		s = "0" + s
	}
	return strconv.FormatInt(c/100, 10) + "," + s
}
