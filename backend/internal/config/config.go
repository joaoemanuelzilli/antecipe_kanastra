// Package config lê config/antecipe.yaml: o comportamento ajustável sem mexer no código
// (método de preço e parâmetros de processamento). Segredos ficam no ambiente, nunca aqui.
package config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/shopspring/decimal"
	"go.yaml.in/yaml/v3"

	"antecipe/internal/precificacao"
)

// Precificacao sobrescreve a precificação das regras publicadas. Campo nil = não sobrescreve.
type Precificacao struct {
	Metodo               *string          `yaml:"metodo"`
	TaxaMensalPercentual *decimal.Decimal `yaml:"taxa_mensal_percentual"`
	BaseDias             *int             `yaml:"base_dias"`
}

type Processamento struct {
	DataBasePadrao          string `yaml:"data_base_padrao"`
	SituacaoCadastralPadrao string `yaml:"situacao_cadastral_padrao"`
	JobsSimultaneos         int    `yaml:"jobs_simultaneos"`
	TamanhoMaximoUploadMB   int    `yaml:"tamanho_maximo_upload_mb"`
}

type Config struct {
	Precificacao  Precificacao  `yaml:"precificacao"`
	Processamento Processamento `yaml:"processamento"`

	dataBase time.Time
}

// Carregar lê e valida o arquivo.
func Carregar(caminho string) (Config, error) {
	b, err := os.ReadFile(caminho)
	if err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return Config{}, fmt.Errorf("config: YAML inválido: %w", err)
	}
	if m := c.Precificacao.Metodo; m != nil && !slices.Contains(precificacao.Metodos(), *m) {
		return Config{}, fmt.Errorf("config: precificacao.metodo %q desconhecido (use %s)", *m, strings.Join(precificacao.Metodos(), " ou "))
	}
	if t := c.Precificacao.TaxaMensalPercentual; t != nil && t.IsNegative() {
		return Config{}, errors.New("config: precificacao.taxa_mensal_percentual não pode ser negativa")
	}
	if b := c.Precificacao.BaseDias; b != nil && *b < 1 {
		return Config{}, errors.New("config: precificacao.base_dias deve ser maior que zero")
	}
	p := c.Processamento
	db, err := time.Parse("2006-01-02", p.DataBasePadrao)
	if err != nil {
		return Config{}, errors.New("config: processamento.data_base_padrao é obrigatória no formato AAAA-MM-DD")
	}
	c.dataBase = db
	if p.JobsSimultaneos < 1 {
		return Config{}, errors.New("config: processamento.jobs_simultaneos deve ser maior que zero")
	}
	if p.TamanhoMaximoUploadMB < 1 {
		return Config{}, errors.New("config: processamento.tamanho_maximo_upload_mb deve ser maior que zero")
	}
	if strings.TrimSpace(p.SituacaoCadastralPadrao) == "" {
		return Config{}, errors.New("config: processamento.situacao_cadastral_padrao não pode ficar vazia")
	}
	return c, nil
}

// DataBasePadrao devolve a data-base usada quando o upload não informa uma.
func (c Config) DataBasePadrao() time.Time { return c.dataBase }

// Atual guarda a configuração vigente e a troca quando o arquivo muda.
type Atual struct {
	caminho string
	log     *slog.Logger
	cfg     atomic.Pointer[Config]
	mod     time.Time
}

// NovoAtual carrega o arquivo (erro aqui impede a subida).
func NovoAtual(caminho string, log *slog.Logger) (*Atual, error) {
	c, err := Carregar(caminho)
	if err != nil {
		return nil, err
	}
	a := &Atual{caminho: caminho, log: log}
	a.cfg.Store(&c)
	if st, err := os.Stat(caminho); err == nil {
		a.mod = st.ModTime()
	}
	return a, nil
}

// Obter devolve a configuração vigente.
func (a *Atual) Obter() Config { return *a.cfg.Load() }

// Observar relê o arquivo quando a data de modificação muda. Arquivo inválido mantém a configuração anterior.
func (a *Atual) Observar(ctx context.Context, intervalo time.Duration) {
	t := time.NewTicker(intervalo)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			st, err := os.Stat(a.caminho)
			if err != nil || st.ModTime().Equal(a.mod) {
				continue
			}
			a.mod = st.ModTime()
			c, err := Carregar(a.caminho)
			if err != nil {
				a.log.Warn("config inválida, mantendo a anterior", slog.String("erro", err.Error()))
				continue
			}
			a.cfg.Store(&c)
			metodo := "das regras"
			if c.Precificacao.Metodo != nil {
				metodo = *c.Precificacao.Metodo
			}
			a.log.Info("config recarregada", slog.String("metodo_preco", metodo))
		}
	}
}
