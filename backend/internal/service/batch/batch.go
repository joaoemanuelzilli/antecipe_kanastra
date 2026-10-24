// Package batch é o caso de uso dos lotes: recebe o arquivo, enfileira, processa em etapas com
// progresso em memória e grava o resultado. Depende só das portas do domínio.
package batch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel/metric"

	"antecipe/internal/domain"
	"antecipe/internal/domain/rules"
	"antecipe/internal/platform/apperrors"
	"antecipe/internal/platform/config"
	"antecipe/internal/platform/observability"
	"antecipe/internal/service/engine"
	"antecipe/internal/service/pricing"
)

// Deps são as portas e a configuração que o serviço usa.
type Deps struct {
	Lotes     domain.LoteRepository
	Estoque   domain.EstoqueRepository
	Situacoes domain.SituacaoRepository
	Regras    domain.RegrasRepository
	Leitor    domain.LeitorNotas
	Config    *config.Atual
	Log       *slog.Logger
}

// Progresso de um lote que está na fila ou processando.
type Progresso struct {
	Status      string
	Etapa       string
	Processadas int
	Total       int
	PosicaoFila int
}

type trabalho struct{ id, caminho string }

// Servico de lotes.
type Servico struct {
	d         Deps
	fila      chan trabalho
	mu        sync.Mutex
	progresso map[string]*Progresso
	ordem     []string // ids na fila, em ordem
	wg        sync.WaitGroup

	lotesTotal  metric.Int64Counter
	notasTotal  metric.Int64Counter
	duracaoLote metric.Float64Histogram
	duracaoEtap metric.Float64Histogram
}

// Novo cria o serviço (chame Iniciar para começar a processar).
func Novo(d Deps) *Servico {
	m := observability.Meter()
	s := &Servico{d: d, fila: make(chan trabalho, 1024), progresso: map[string]*Progresso{}}
	s.lotesTotal, _ = m.Int64Counter("antecipe_lotes_total", metric.WithDescription("Lotes processados, por status final"))
	s.notasTotal, _ = m.Int64Counter("antecipe_notas_total", metric.WithDescription("Notas avaliadas, por status"))
	s.duracaoLote, _ = m.Float64Histogram("antecipe_lote_duracao_segundos", metric.WithDescription("Tempo de processamento de um lote"), metric.WithUnit("s"))
	s.duracaoEtap, _ = m.Float64Histogram("antecipe_etapa_duracao_segundos", metric.WithDescription("Tempo de cada etapa do lote"), metric.WithUnit("s"))
	return s
}

// Iniciar marca como erro os lotes interrompidos por uma parada anterior e sobe os workers.
func (s *Servico) Iniciar(ctx context.Context) error {
	n, err := s.d.Lotes.RecuperarInterrompidos(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		s.d.Log.WarnContext(ctx, "lotes interrompidos marcados como erro", slog.Int("lotes", n))
	}
	for i := 0; i < s.d.Config.Obter().Processamento.JobsSimultaneos; i++ {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			for t := range s.fila {
				s.sairDaFila(t.id)
				s.processar(context.WithoutCancel(ctx), t)
			}
		}()
	}
	return nil
}

// Fechar para de aceitar trabalhos e espera os lotes em andamento (até o ctx expirar).
func (s *Servico) Fechar(ctx context.Context) {
	close(s.fila)
	pronto := make(chan struct{})
	go func() { s.wg.Wait(); close(pronto) }()
	select {
	case <-pronto:
	case <-ctx.Done():
	}
}

// Envio é um arquivo já salvo em disco pelo handler. O serviço passa a ser dono do arquivo.
type Envio struct {
	CaminhoArquivo string
	Hash           string
	IdempotencyKey string
	DataBase       *time.Time
}

// Enviar registra o lote (ou devolve o existente, se for reenvio) e o coloca na fila.
func (s *Servico) Enviar(ctx context.Context, e Envio) (id string, replay bool, err error) {
	enfileirado := false
	defer func() {
		if !enfileirado {
			os.Remove(e.CaminhoArquivo)
		}
	}()
	cfg := s.d.Config.Obter()
	dataBase := cfg.DataBasePadrao()
	if e.DataBase != nil {
		dataBase = *e.DataBase
	}
	versao, err := s.d.Regras.Publicadas(ctx)
	if err != nil {
		return "", false, err
	}
	prec, _ := cfg.PrecificacaoEfetiva(versao.Regras.Precificacao)
	if _, err := pricing.Nova(prec); err != nil {
		return "", false, apperrors.Unavailable("PRECIFICACAO_INVALIDA", "A configuração de preço está inválida. Avise o gestor do fundo.").Wrap(err)
	}
	total, err := s.contar(e.CaminhoArquivo)
	if err != nil {
		return "", false, err
	}
	if total == 0 {
		return "", false, apperrors.Validation("ARQUIVO_VAZIO", "O arquivo não tem nenhuma nota.")
	}
	h := sha256.Sum256([]byte(e.Hash + "|" + dataBase.Format("2006-01-02") + "|" + itoa(versao.Numero) + "|" +
		prec.Metodo + "|" + prec.TaxaMensalPercentual.String() + "|" + itoa(prec.BaseDias)))
	impressao := hex.EncodeToString(h[:])
	chave := e.IdempotencyKey
	if chave == "" {
		chave = impressao
	}
	lc, err := s.d.Lotes.CriarLote(ctx, domain.NovoLote{
		IdempotencyKey: chave, Impressao: impressao, Hash: e.Hash, DataBase: dataBase, RegrasVersao: versao.Numero, Precificacao: prec, LinhasTotal: total,
	})
	if err != nil {
		return "", false, err
	}
	id, criado := lc.ID, lc.Criado
	if !criado && lc.Impressao != impressao {
		return "", false, apperrors.Conflict("IDEMPOTENCY_KEY_REUTILIZADA", "Essa Idempotency-Key já foi usada com outro arquivo ou outra data-base.")
	}
	if !criado && lc.Status == domain.LoteErro {
		reiniciou, err := s.d.Lotes.ReiniciarLote(ctx, id, total, impressao)
		if err != nil {
			return "", false, err
		}
		criado = reiniciou
	}
	if !criado {
		return id, true, nil
	}
	s.enfileirar(id, e.CaminhoArquivo, total)
	enfileirado = true
	s.d.Log.InfoContext(ctx, "lote recebido", slog.String("lote_id", id), slog.Int("linhas", total))
	return id, false, nil
}

func (s *Servico) contar(caminho string) (int, error) {
	f, err := os.Open(caminho)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return s.d.Leitor.ContarLinhas(f)
}

func (s *Servico) enfileirar(id, caminho string, total int) {
	s.mu.Lock()
	s.progresso[id] = &Progresso{Status: domain.LoteNaFila, Total: total}
	s.ordem = append(s.ordem, id)
	s.mu.Unlock()
	s.fila <- trabalho{id: id, caminho: caminho}
}

func (s *Servico) sairDaFila(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, x := range s.ordem {
		if x == id {
			s.ordem = append(s.ordem[:i], s.ordem[i+1:]...)
			break
		}
	}
	if p := s.progresso[id]; p != nil {
		p.Status = domain.LoteProcessando
	}
}

// Progresso devolve o andamento em memória (só existe enquanto o lote está na fila ou processando).
func (s *Servico) Progresso(id string) (Progresso, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.progresso[id]
	if !ok {
		return Progresso{}, false
	}
	c := *p
	if c.Status == domain.LoteNaFila {
		for i, x := range s.ordem {
			if x == id {
				c.PosicaoFila = i + 1
			}
		}
	}
	return c, true
}

func (s *Servico) atualizar(id string, f func(p *Progresso)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p := s.progresso[id]; p != nil {
		f(p)
	}
}

// Obter devolve o lote com o andamento em memória por cima do que está no banco.
func (s *Servico) Obter(ctx context.Context, id string) (domain.Lote, int, error) {
	l, err := s.d.Lotes.ObterLote(ctx, id)
	if err != nil {
		return l, 0, err
	}
	if p, ok := s.Progresso(id); ok && (l.Status == domain.LoteNaFila || l.Status == domain.LoteProcessando) {
		l.Status, l.Etapa, l.LinhasProcessadas, l.LinhasTotal = p.Status, p.Etapa, p.Processadas, p.Total
		return l, p.PosicaoFila, nil
	}
	return l, 0, nil
}

// ListarNotas devolve uma página das notas do lote.
func (s *Servico) ListarNotas(ctx context.Context, id string, f domain.FiltroNotas) ([]domain.NotaLinha, int, error) {
	if _, err := s.d.Lotes.ObterLote(ctx, id); err != nil {
		return nil, 0, err
	}
	return s.d.Lotes.ListarNotas(ctx, id, f)
}

// PercorrerNotas lê todas as notas filtradas, em fluxo.
func (s *Servico) PercorrerNotas(ctx context.Context, id string, f domain.FiltroNotas, fn func(domain.NotaLinha) error) error {
	return s.d.Lotes.PercorrerNotas(ctx, id, f, fn)
}

// Atual é a configuração em vigor para novos lotes.
type Atual struct {
	Versao       domain.VersaoRegras
	Precificacao domain.Precificacao
	Sobrescritos []string
	DataBase     time.Time
	TamanhoMaxMB int
}

// ConfiguracaoAtual junta as regras publicadas e o YAML.
func (s *Servico) ConfiguracaoAtual(ctx context.Context) (Atual, error) {
	v, err := s.d.Regras.Publicadas(ctx)
	if err != nil {
		return Atual{}, err
	}
	cfg := s.d.Config.Obter()
	p, sob := cfg.PrecificacaoEfetiva(v.Regras.Precificacao)
	return Atual{Versao: v, Precificacao: p, Sobrescritos: sob, DataBase: cfg.DataBasePadrao(), TamanhoMaxMB: cfg.Processamento.TamanhoMaximoUploadMB}, nil
}

// Validacao é o resultado de uma validação síncrona (sem gravar nada).
type Validacao struct {
	Saida        engine.Saida
	Precificacao domain.Precificacao
	RegrasVersao int
	DataBase     time.Time
}

// ValidarAgora roda o mesmo motor dos lotes sobre notas já lidas, sem persistir.
func (s *Servico) ValidarAgora(ctx context.Context, notas []domain.Nota, dataBase *time.Time) (Validacao, error) {
	atual, err := s.ConfiguracaoAtual(ctx)
	if err != nil {
		return Validacao{}, err
	}
	db := atual.DataBase
	if dataBase != nil {
		db = *dataBase
	}
	ectx, err := s.contextoMotor(ctx, atual.Versao, atual.Precificacao, db, "")
	if err != nil {
		return Validacao{}, err
	}
	return Validacao{Saida: engine.Validar(notas, atual.Versao.Regras, ectx), Precificacao: atual.Precificacao, RegrasVersao: atual.Versao.Numero, DataBase: db}, nil
}

func (s *Servico) contextoMotor(ctx context.Context, v domain.VersaoRegras, p domain.Precificacao, dataBase time.Time, ignorarCessao string) (engine.Contexto, error) {
	calc, err := pricing.Nova(p)
	if err != nil {
		return engine.Contexto{}, apperrors.Unavailable("PRECIFICACAO_INVALIDA", "A configuração de preço está inválida. Avise o gestor do fundo.").Wrap(err)
	}
	est, err := s.d.Estoque.ContextoMotor(ctx, ignorarCessao)
	if err != nil {
		return engine.Contexto{}, err
	}
	sit, err := s.d.Situacoes.Situacoes(ctx)
	if err != nil {
		return engine.Contexto{}, err
	}
	padrao := s.d.Config.Obter().Processamento.SituacaoCadastralPadrao
	return engine.Contexto{
		DataBase: dataBase,
		Situacao: func(cnpj string) string {
			if x, ok := sit[cnpj]; ok {
				return x
			}
			return padrao
		},
		Vendidas: est.Vendidas, Estoques: est.Estoques, Itens: est.Itens, Preco: calc,
	}, nil
}

func itoa(n int) string {
	return string(appendInt(nil, n))
}

func appendInt(b []byte, n int) []byte {
	if n < 0 {
		b = append(b, '-')
		n = -n
	}
	var d [20]byte
	i := len(d)
	for {
		i--
		d[i] = byte('0' + n%10)
		n /= 10
		if n == 0 {
			break
		}
	}
	return append(b, d[i:]...)
}

// Simular roda o motor com regras e preço quaisquer (usado pela prévia do gestor). Não grava nada.
// ignorarCessao deixa de fora as notas que o próprio lote já vendeu.
func (s *Servico) Simular(ctx context.Context, notas []domain.Nota, r rules.Regras, p domain.Precificacao, dataBase time.Time, ignorarCessao string) (engine.Saida, error) {
	ectx, err := s.contextoMotor(ctx, domain.VersaoRegras{Regras: r}, p, dataBase, ignorarCessao)
	if err != nil {
		return engine.Saida{}, err
	}
	return engine.Validar(notas, r, ectx), nil
}
