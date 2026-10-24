// Package motor avalia um lote de notas contra as regras publicadas. É puro: não acessa banco,
// rede nem relógio. A ordem é fixa: formato → regras por nota (em paralelo) → duplicidade →
// concentração (determinística) → preço → totais.
package motor

import (
	"math"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"antecipe/internal/documento"
	"antecipe/internal/dominio"
	"antecipe/internal/precificacao"
	"antecipe/internal/regras"
	"antecipe/internal/validadores"
)

// Vendida é uma nota que já está no estoque (de qualquer fundo) ou numa cessão.
type Vendida struct {
	Em            time.Time
	ValorCentavos int64
}

// EstoqueFundo soma o estoque do fundo das regras, por documento normalizado.
type EstoqueFundo struct {
	TotalCentavos int64
	PorSacado     map[string]int64
	PorCedente    map[string]int64
}

// Contexto traz tudo que o motor precisa saber além das notas e das regras.
type Contexto struct {
	DataBase  time.Time
	Situacao  func(cnpjNorm string) string
	Vendidas  map[string]Vendida
	Estoque   EstoqueFundo
	Preco     *precificacao.Calculadora
	AoAvancar func(etapa string, feitas int) // opcional
}

type Totais struct {
	Linhas           int   `json:"linhas"`
	Aprovadas        int   `json:"aprovadas"`
	Recusadas        int   `json:"recusadas"`
	FaceCentavos     int64 `json:"face_centavos"`
	DescontoCentavos int64 `json:"desconto_centavos"`
	LiquidoCentavos  int64 `json:"liquido_centavos"`
}

type ErroConfig struct {
	Codigo   string `json:"codigo"`
	Mensagem string `json:"mensagem"`
}

// Saida do motor. Resultados[i] corresponde a notas[i].
type Saida struct {
	Resultados  []dominio.Resultado
	Totais      Totais
	Motivos     map[string]int // pelo motivo principal
	ErrosConfig []ErroConfig
}

const (
	tamanhoBloco       = 4096
	intervaloProgresso = 50000
	iso                = "2006-01-02"
)

// Chave identifica a nota para duplicidade: cedente + número + vencimento (o valor fica de fora de propósito).
func Chave(n *dominio.Nota) string {
	return documento.Normalizar(n.DocCedente) + "|" + strings.TrimSpace(n.Numero) + "|" + n.Vencimento.Format(iso)
}

// Dias corridos entre a data-base e o vencimento.
func Dias(vencimento, dataBase time.Time) int {
	return int(math.Round(vencimento.Sub(dataBase).Hours() / 24))
}

type coletorConfig struct {
	mu    sync.Mutex
	visto map[string]bool
	erros []ErroConfig
}

func (c *coletorConfig) add(codigo, msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.visto == nil {
		c.visto = map[string]bool{}
	}
	if c.visto[codigo] {
		return
	}
	c.visto[codigo] = true
	c.erros = append(c.erros, ErroConfig{Codigo: codigo, Mensagem: msg})
}

type regraPorNota struct {
	codigo string
	f      validadores.Func
	p      regras.Parametros
}

// Validar avalia o lote inteiro.
func Validar(notas []dominio.Nota, r regras.Regras, c Contexto) Saida {
	avancar := progresso(c.AoAvancar)
	cfg := &coletorConfig{}
	var porNota []regraPorNota
	for _, v := range r.Validadores {
		if !v.Ativo {
			continue
		}
		if f, ok := validadores.PorNota[v.Codigo]; ok {
			porNota = append(porNota, regraPorNota{v.Codigo, f, v.Parametros})
		} else if !validadores.DeLote[v.Codigo] {
			cfg.add(v.Codigo, "Regra "+v.Codigo+" não existe e foi ignorada.")
		}
	}

	res := make([]dominio.Resultado, len(notas))
	avaliarPorNota(notas, res, porNota, validadores.Ctx{DataBase: c.DataBase, Situacao: c.Situacao}, cfg, avancar)
	marcarDuplicidade(notas, res, c.Vendidas)

	cands := make([]*dominio.Resultado, 0, len(res))
	for i := range res {
		if len(res[i].Recusas) == 0 {
			cands = append(cands, &res[i])
		}
	}
	for _, e := range Concentrar(cands, r, c.Estoque) {
		cfg.add(e.Codigo, e.Mensagem)
	}
	avancar("concentracao", len(cands))

	s := Saida{Resultados: res, Motivos: map[string]int{}}
	s.Totais.Linhas = len(res)
	for i := range res {
		rs := &res[i]
		if len(rs.Recusas) == 0 {
			rs.Status = dominio.Aprovada
			rs.DescontoCentavos, rs.LiquidoCentavos = c.Preco.Calcular(rs.Nota.ValorCentavos, rs.Dias)
			s.Totais.Aprovadas++
			s.Totais.FaceCentavos += rs.Nota.ValorCentavos
			s.Totais.DescontoCentavos += rs.DescontoCentavos
			s.Totais.LiquidoCentavos += rs.LiquidoCentavos
		} else {
			rs.Status = dominio.Recusada
			s.Totais.Recusadas++
			s.Motivos[rs.Recusas[0].Codigo]++
		}
		if (i+1)%intervaloProgresso == 0 {
			avancar("precificando", i+1)
		}
	}
	avancar("precificando", len(res))
	s.ErrosConfig = cfg.erros
	return s
}

// progresso serializa as chamadas de AoAvancar (os workers rodam em paralelo).
func progresso(f func(string, int)) func(string, int) {
	if f == nil {
		return func(string, int) {}
	}
	var mu sync.Mutex
	return func(etapa string, n int) {
		mu.Lock()
		defer mu.Unlock()
		f(etapa, n)
	}
}

func avaliarPorNota(notas []dominio.Nota, res []dominio.Resultado, porNota []regraPorNota, vctx validadores.Ctx, cfg *coletorConfig, avancar func(string, int)) {
	blocos := make(chan int, (len(notas)+tamanhoBloco-1)/tamanhoBloco)
	for ini := 0; ini < len(notas); ini += tamanhoBloco {
		blocos <- ini
	}
	close(blocos)

	var feitas atomic.Int64
	var wg sync.WaitGroup
	for w := 0; w < runtime.GOMAXPROCS(0); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ini := range blocos {
				fim := min(ini+tamanhoBloco, len(notas))
				for i := ini; i < fim; i++ {
					avaliarNota(&notas[i], &res[i], porNota, vctx, cfg)
				}
				avancar("validando", int(feitas.Add(int64(fim-ini))))
			}
		}()
	}
	wg.Wait()
}

func avaliarNota(n *dominio.Nota, rs *dominio.Resultado, porNota []regraPorNota, vctx validadores.Ctx, cfg *coletorConfig) {
	rs.Nota = n
	if e := n.ErroFormato; e != nil {
		rs.Recusas = []dominio.Recusa{{Codigo: "FORMATO_INVALIDO", Dados: map[string]any{
			"campo": e.Campo, "valor": e.Valor, "problema": e.Problema, "linha": n.Linha,
		}}}
		return
	}
	rs.Dias = Dias(n.Vencimento, vctx.DataBase)
	rs.Chave = Chave(n)
	it := validadores.Item{Nota: n, Dias: rs.Dias, SacadoNorm: documento.Normalizar(n.DocSacado)}
	for _, rg := range porNota {
		ok, dados, err := rg.f(it, rg.p, vctx)
		if err != nil {
			cfg.add(rg.codigo, "Regra "+rg.codigo+" tem parâmetro inválido e foi ignorada.")
			continue
		}
		if !ok {
			rs.Recusas = append(rs.Recusas, dominio.Recusa{Codigo: rg.codigo, Dados: dados})
		}
	}
}

// marcarDuplicidade roda em ordem de linha: a primeira ocorrência no arquivo é avaliada normalmente.
// Recusas de integridade vão para a frente (viram o motivo principal).
func marcarDuplicidade(notas []dominio.Nota, res []dominio.Resultado, vendidas map[string]Vendida) {
	vistas := make(map[string]int, len(notas))
	for i := range res {
		n := &notas[i]
		if n.ErroFormato != nil {
			continue
		}
		k := res[i].Chave
		var rec *dominio.Recusa
		if v, ok := vendidas[k]; ok {
			em := ""
			if !v.Em.IsZero() {
				em = v.Em.Format(iso)
			}
			rec = &dominio.Recusa{Codigo: "DUPLICIDADE", Dados: map[string]any{
				"vendida_em": em, "valor_original_centavos": v.ValorCentavos, "valor_centavos": n.ValorCentavos,
				"numero": n.Numero, "vencimento": n.Vencimento.Format(iso),
			}}
		} else if linha, ok := vistas[k]; ok {
			rec = &dominio.Recusa{Codigo: "DUPLICADA_NO_ARQUIVO", Dados: map[string]any{"linha_original": linha}}
		} else {
			vistas[k] = n.Linha
		}
		if rec != nil {
			res[i].Recusas = append([]dominio.Recusa{*rec}, res[i].Recusas...)
		}
	}
}
