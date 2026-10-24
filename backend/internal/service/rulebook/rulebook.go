// Package rulebook é o caso de uso das regras do fundo: o gestor vê, testa (prévia) e publica
// novas versões sem deploy. Os próximos lotes usam a versão publicada.
package rulebook

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	"antecipe/internal/domain"
	"antecipe/internal/domain/rules"
	"antecipe/internal/platform/apperrors"
	"antecipe/internal/platform/config"
	"antecipe/internal/service/engine"
	"antecipe/internal/service/pricing"
)

// Simulador roda o motor sem gravar (o serviço de lotes implementa).
type Simulador interface {
	Simular(ctx context.Context, notas []domain.Nota, r rules.Regras, p domain.Precificacao, dataBase time.Time, ignorarCessao string) (engine.Saida, error)
}

type Servico struct {
	regras domain.RegrasRepository
	lotes  domain.LoteRepository
	sim    Simulador
	cfg    *config.Atual
	log    *slog.Logger
}

func Novo(regras domain.RegrasRepository, lotes domain.LoteRepository, sim Simulador, cfg *config.Atual, log *slog.Logger) *Servico {
	return &Servico{regras: regras, lotes: lotes, sim: sim, cfg: cfg, log: log}
}

// Efetiva é a versão publicada com a precificação que vale de fato (YAML por cima das regras).
type Efetiva struct {
	Versao       domain.VersaoRegras
	Precificacao domain.Precificacao
	Sobrescritos []string
}

func (s *Servico) Publicadas(ctx context.Context) (Efetiva, error) {
	v, err := s.regras.Publicadas(ctx)
	if err != nil {
		return Efetiva{}, err
	}
	p, sob := s.cfg.Obter().PrecificacaoEfetiva(v.Regras.Precificacao)
	return Efetiva{Versao: v, Precificacao: p, Sobrescritos: sob}, nil
}

func (s *Servico) Versoes(ctx context.Context) ([]domain.VersaoRegras, error) {
	return s.regras.Listar(ctx)
}

func (s *Servico) Versao(ctx context.Context, n int) (domain.VersaoRegras, error) {
	return s.regras.Obter(ctx, n)
}

func validar(r rules.Regras) error {
	if errs := rules.ValidarSchema(r, pricing.Metodos()); len(errs) > 0 {
		return apperrors.Validation("REGRAS_INVALIDAS", "Algumas regras têm valores inválidos.").WithDetails(errs)
	}
	return nil
}

// Publicar valida, calcula o resumo das mudanças e grava a nova versão.
func (s *Servico) Publicar(ctx context.Context, r rules.Regras, autor string) (domain.VersaoRegras, error) {
	if err := validar(r); err != nil {
		return domain.VersaoRegras{}, err
	}
	atual, err := s.regras.Publicadas(ctx)
	if err != nil {
		return domain.VersaoRegras{}, err
	}
	resumo := rules.Resumo(atual.Regras, r)
	if len(resumo) == 0 {
		return domain.VersaoRegras{}, apperrors.Validation("SEM_MUDANCAS", "Nada mudou em relação à versão publicada.")
	}
	autor = strings.TrimSpace(autor)
	if autor == "" {
		autor = "gestor"
	}
	if len([]rune(autor)) > 60 {
		autor = string([]rune(autor)[:60])
	}
	v, err := s.regras.Publicar(ctx, r, autor, resumo)
	if err != nil {
		return v, err
	}
	s.log.InfoContext(ctx, "regras publicadas", slog.Int("versao_regras", v.Numero), slog.Int("mudancas", len(resumo)))
	return v, nil
}

// Lado é um dos lados da comparação da prévia.
type Lado struct {
	RegrasVersao int
	Aprovadas    int
	Recusadas    int
	Liquido      int64
}

type MotivoComparado struct {
	Codigo        string
	Antes, Depois int
}

type Previa struct {
	LoteID   string
	Linhas   int
	Antes    Lado
	Depois   Lado
	Motivos  []MotivoComparado
	Mudancas []string
}

// Previa roda o último lote (ou o informado) com as regras propostas, sem publicar.
func (s *Servico) Previa(ctx context.Context, r rules.Regras, loteID string) (Previa, error) {
	if err := validar(r); err != nil {
		return Previa{}, err
	}
	if loteID == "" {
		id, err := s.lotes.UltimoLoteConcluido(ctx)
		if err != nil {
			return Previa{}, err
		}
		loteID = id
	}
	l, err := s.lotes.ObterLote(ctx, loteID)
	if err != nil {
		return Previa{}, err
	}
	if l.Status != domain.LoteConcluido || l.Totais == nil {
		return Previa{}, apperrors.Conflict("LOTE_NAO_CONCLUIDO", "Esse lote ainda está sendo processado.")
	}
	notas, err := s.lotes.NotasOriginais(ctx, loteID)
	if err != nil {
		return Previa{}, err
	}
	atual, err := s.regras.Publicadas(ctx)
	if err != nil {
		return Previa{}, err
	}
	prec, _ := s.cfg.Obter().PrecificacaoEfetiva(r.Precificacao)
	ignorar := ""
	if l.Cessao != nil {
		ignorar = l.Cessao.ID
	}
	saida, err := s.sim.Simular(ctx, notas, r, prec, l.DataBase, ignorar)
	if err != nil {
		return Previa{}, err
	}
	p := Previa{
		LoteID: loteID, Linhas: l.Totais.Linhas,
		Antes:    Lado{RegrasVersao: l.RegrasVersao, Aprovadas: l.Totais.Aprovadas, Recusadas: l.Totais.Recusadas, Liquido: l.Totais.LiquidoCentavos},
		Depois:   Lado{Aprovadas: saida.Totais.Aprovadas, Recusadas: saida.Totais.Recusadas, Liquido: saida.Totais.LiquidoCentavos},
		Mudancas: rules.Resumo(atual.Regras, r),
	}
	codigos := map[string]bool{}
	for c := range l.Motivos {
		codigos[c] = true
	}
	for c := range saida.Motivos {
		codigos[c] = true
	}
	for c := range codigos {
		p.Motivos = append(p.Motivos, MotivoComparado{Codigo: c, Antes: l.Motivos[c], Depois: saida.Motivos[c]})
	}
	sort.Slice(p.Motivos, func(i, j int) bool {
		return p.Motivos[i].Depois > p.Motivos[j].Depois || (p.Motivos[i].Depois == p.Motivos[j].Depois && p.Motivos[i].Codigo < p.Motivos[j].Codigo)
	})
	if p.Mudancas == nil {
		p.Mudancas = []string{}
	}
	return p, nil
}
