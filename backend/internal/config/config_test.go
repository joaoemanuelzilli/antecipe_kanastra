package config

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"antecipe/internal/obs"
)

func TestCarregarRepo(t *testing.T) {
	c, err := Carregar("../../../config/antecipe.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if c.Precificacao.Metodo == nil || *c.Precificacao.Metodo != "DESCONTO_COMPOSTO" {
		t.Fatalf("metodo = %v", c.Precificacao.Metodo)
	}
	if c.Precificacao.TaxaMensalPercentual != nil || c.Precificacao.BaseDias != nil {
		t.Fatal("taxa e base não deveriam estar definidas")
	}
	if !c.DataBasePadrao().Equal(time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("data-base = %v", c.DataBasePadrao())
	}
	p := c.Processamento
	if p.JobsSimultaneos != 2 || p.TamanhoMaximoUploadMB != 300 || p.SituacaoCadastralPadrao != "ATIVA" {
		t.Fatalf("processamento = %+v", p)
	}
}

func escrever(t *testing.T, caminho, conteudo string) {
	t.Helper()
	if err := os.WriteFile(caminho, []byte(conteudo), 0o644); err != nil {
		t.Fatal(err)
	}
}

const base = `
precificacao:
  metodo: %s
processamento:
  data_base_padrao: "2026-10-22"
  situacao_cadastral_padrao: ATIVA
  jobs_simultaneos: 2
  tamanho_maximo_upload_mb: 300
`

func arquivoYAML(metodo string) string { return replace(base, "%s", metodo) }

func replace(s, a, b string) string { return string(bytes.Replace([]byte(s), []byte(a), []byte(b), 1)) }

func TestCarregarMetodoInvalido(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	escrever(t, p, arquivoYAML("MAGICO"))
	if _, err := Carregar(p); err == nil {
		t.Fatal("método inválido deveria falhar")
	}
}

func TestCarregarTaxaSobrescrita(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	escrever(t, p, replace(arquivoYAML("DESCONTO_SIMPLES"), "metodo: DESCONTO_SIMPLES", "metodo: DESCONTO_SIMPLES\n  taxa_mensal_percentual: 2.5\n  base_dias: 360"))
	c, err := Carregar(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.Precificacao.TaxaMensalPercentual == nil || c.Precificacao.TaxaMensalPercentual.String() != "2.5" || *c.Precificacao.BaseDias != 360 {
		t.Fatalf("precificacao = %+v", c.Precificacao)
	}
}

func TestCarregarSemDataBase(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	escrever(t, p, replace(arquivoYAML("DESCONTO_SIMPLES"), `data_base_padrao: "2026-10-22"`, ""))
	if _, err := Carregar(p); err == nil {
		t.Fatal("sem data-base deveria falhar")
	}
}

func TestObservarTroca(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	escrever(t, p, arquivoYAML("DESCONTO_COMPOSTO"))
	a, err := NovoAtual(p, obs.NovoLogger(&bytes.Buffer{}, "error"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	go a.Observar(ctx, 20*time.Millisecond)

	escrever(t, p, arquivoYAML("DESCONTO_SIMPLES"))
	futuro := time.Now().Add(2 * time.Second)
	_ = os.Chtimes(p, futuro, futuro)

	limite := time.Now().Add(2 * time.Second)
	for time.Now().Before(limite) {
		if *a.Obter().Precificacao.Metodo == "DESCONTO_SIMPLES" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("config não foi relida: %s", *a.Obter().Precificacao.Metodo)
}

func TestObservarInvalidoMantem(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	escrever(t, p, arquivoYAML("DESCONTO_SIMPLES"))
	logs := &bufSeguro{}
	a, err := NovoAtual(p, obs.NovoLogger(logs, "info"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	go a.Observar(ctx, 20*time.Millisecond)

	escrever(t, p, ": : lixo")
	futuro := time.Now().Add(2 * time.Second)
	_ = os.Chtimes(p, futuro, futuro)
	time.Sleep(200 * time.Millisecond)

	if *a.Obter().Precificacao.Metodo != "DESCONTO_SIMPLES" {
		t.Fatalf("config trocada por inválida: %s", *a.Obter().Precificacao.Metodo)
	}
	if !strings.Contains(logs.String(), "config inválida, mantendo a anterior") {
		t.Fatalf("log esperado ausente: %s", logs.String())
	}
}

// bufSeguro é um buffer de log que pode ser lido enquanto outra goroutine escreve.
type bufSeguro struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *bufSeguro) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}
func (s *bufSeguro) String() string { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }
