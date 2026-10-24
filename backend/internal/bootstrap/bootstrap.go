// Package bootstrap é a raiz de composição: liga adaptadores, serviços e handlers.
// É usado pelo cmd/api e pelos testes ponta a ponta.
package bootstrap

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"antecipe/internal/domain/rules"
	"antecipe/internal/handler/csvfile"
	"antecipe/internal/handler/rest"
	"antecipe/internal/platform/config"
	"antecipe/internal/repository/filesystem"
	"antecipe/internal/repository/postgres"
	"antecipe/internal/service/batch"
)

// Ambiente vem das variáveis de ambiente (segredos e infraestrutura).
type Ambiente struct {
	HTTPAddr    string
	DatabaseURL string
	LogLevel    string
	ConfigPath  string
	DataDir     string
	GestorSenha string
	CORSOrigens []string
	TmpDir      string
	Metricas    http.Handler // handler de /metrics do OpenTelemetry (opcional)
}

// AmbienteDoSistema lê as variáveis de ambiente com os padrões documentados no README.
func AmbienteDoSistema() Ambiente {
	env := func(k, padrao string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return padrao
	}
	var origens []string
	for _, o := range strings.Split(env("CORS_ORIGINS", "http://localhost:5173"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			origens = append(origens, o)
		}
	}
	return Ambiente{
		HTTPAddr: env("HTTP_ADDR", ":8080"), DatabaseURL: os.Getenv("DATABASE_URL"), LogLevel: env("LOG_LEVEL", "info"),
		ConfigPath: env("CONFIG_PATH", "config/antecipe.yaml"), DataDir: env("DATA_DIR", "data"),
		GestorSenha: os.Getenv("GESTOR_SENHA"), CORSOrigens: origens, TmpDir: env("TMP_DIR", os.TempDir()),
	}
}

// App é a aplicação montada.
type App struct {
	Handler http.Handler
	Repo    *postgres.Repo
	Lotes   *batch.Servico
	cancel  context.CancelFunc
}

// Novo abre o banco, aplica migrations, carrega a semente se preciso, sobe os serviços e monta as rotas.
func Novo(ctx context.Context, amb Ambiente, log *slog.Logger) (*App, error) {
	repo, err := postgres.Abrir(ctx, amb.DatabaseURL)
	if err != nil {
		return nil, err
	}
	if err := repo.Migrar(ctx); err != nil {
		repo.Fechar()
		return nil, err
	}
	if err := repo.SemearSeVazio(ctx, carregarSemente(amb.DataDir, log)); err != nil {
		repo.Fechar()
		return nil, err
	}
	cfg, err := config.NovoAtual(amb.ConfigPath, log)
	if err != nil {
		repo.Fechar()
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	go cfg.Observar(ctx, 2*time.Second)

	lotes := batch.Novo(batch.Deps{
		Lotes: repo, Estoque: repo, Situacoes: repo, Regras: repo,
		Leitor: csvfile.Leitor{}, Config: cfg, Log: log,
	})
	if err := lotes.Iniciar(ctx); err != nil {
		cancel()
		repo.Fechar()
		return nil, err
	}
	h := rest.NovoServidor(rest.Deps{
		Lotes: lotes, Saude: repo.Ping, Metricas: amb.Metricas, Log: log,
		GestorSenha: amb.GestorSenha, CORSOrigens: amb.CORSOrigens, TmpDir: amb.TmpDir,
	})
	return &App{Handler: h, Repo: repo, Lotes: lotes, cancel: cancel}, nil
}

// Fechar espera os lotes em andamento e fecha o banco.
func (a *App) Fechar(ctx context.Context) {
	espera, cancelar := context.WithTimeout(ctx, 30*time.Second)
	defer cancelar()
	a.Lotes.Fechar(espera)
	a.cancel()
	a.Repo.Fechar()
}

func carregarSemente(dir string, log *slog.Logger) func() (postgres.Semente, error) {
	return func() (postgres.Semente, error) {
		inicio := time.Now()
		est, err := filesystem.LerEstoqueZip(filepath.Join(dir, "estoque_antecipe.zip"))
		if err != nil {
			return postgres.Semente{}, err
		}
		b, err := os.ReadFile(filepath.Join(dir, "validadores_antecipe.json"))
		if err != nil {
			return postgres.Semente{}, err
		}
		reg, err := rules.Ler(b)
		if err != nil {
			return postgres.Semente{}, err
		}
		sit, err := filesystem.LerSituacoes(filepath.Join(dir, "situacao_cadastral.csv"))
		if err != nil {
			return postgres.Semente{}, err
		}
		log.Info("semente carregada", slog.Int("notas_estoque", len(est)), slog.Int("situacoes", len(sit)), slog.Int64("duracao_ms", time.Since(inicio).Milliseconds()))
		return postgres.Semente{Estoque: est, Regras: reg, Situacoes: sit}, nil
	}
}
