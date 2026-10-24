// Package assignment é o caso de uso da cessão (a venda das notas aprovadas ao fundo).
package assignment

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"antecipe/internal/domain"
	"antecipe/internal/platform/document"
	"antecipe/internal/platform/observability"
	"antecipe/internal/service/engine"
	"antecipe/internal/service/message"
)

// Servico de cessão.
type Servico struct {
	cessoes domain.CessaoRepository
	lotes   domain.LoteRepository
	regras  domain.RegrasRepository
	log     *slog.Logger
	vendas  metric.Int64Counter
}

func Novo(cessoes domain.CessaoRepository, lotes domain.LoteRepository, regras domain.RegrasRepository, log *slog.Logger) *Servico {
	c, _ := observability.Meter().Int64Counter("antecipe_notas_cedidas_total", metric.WithDescription("Notas vendidas ao fundo em cessões confirmadas"))
	return &Servico{cessoes: cessoes, lotes: lotes, regras: regras, log: log, vendas: c}
}

// Ceder confirma a venda das aprovadas do lote. É idempotente e segura contra confirmações simultâneas.
func (s *Servico) Ceder(ctx context.Context, loteID, chave string) (domain.Cessao, []domain.Excluida, bool, error) {
	ctx, span := observability.Tracer().Start(ctx, "cessao.confirmar", trace.WithAttributes(attribute.String("lote_id", loteID)))
	defer span.End()

	l, err := s.lotes.ObterLote(ctx, loteID)
	if err != nil {
		return domain.Cessao{}, nil, false, err
	}
	v, err := s.regras.Obter(ctx, l.RegrasVersao)
	if err != nil {
		return domain.Cessao{}, nil, false, err
	}
	// Regra de negócio: a concentração é refeita sobre o estoque atual, porque outra cessão
	// pode ter entrado depois da simulação. Roda dentro da transação do repositório.
	reverificar := func(aprovadas []domain.Resultado, est domain.EstoqueFundo) []domain.Excluida {
		cands := make([]*domain.Resultado, len(aprovadas))
		for i := range aprovadas {
			aprovadas[i].Recusas = nil
			cands[i] = &aprovadas[i]
		}
		fundo := document.Normalizar(v.Regras.Fundo.CNPJ)
		for _, c := range cands {
			c.Nota.DocFundo = fundo // a cessão vende ao fundo das regras
		}
		engine.Concentrar(cands, v.Regras, map[string]domain.EstoqueFundo{fundo: est}, fundo)
		var fora []domain.Excluida
		for _, rs := range cands {
			if len(rs.Recusas) > 0 {
				t := message.Recusa(rs.Recusas[0].Codigo, rs.Recusas[0].Dados)
				fora = append(fora, domain.Excluida{Linha: rs.Nota.Linha, Codigo: rs.Recusas[0].Codigo, Titulo: t.Titulo, Mensagem: t.Mensagem})
			}
		}
		return fora
	}
	c, exc, replay, err := s.cessoes.Ceder(ctx, loteID, chave, domain.FundoDestino{Nome: v.Regras.Fundo.Nome, CNPJ: v.Regras.Fundo.CNPJ}, reverificar)
	if err != nil {
		return c, nil, false, err
	}
	if !replay {
		s.vendas.Add(ctx, int64(c.NotasCedidas))
		s.log.InfoContext(ctx, "cessao confirmada", slog.String("lote_id", loteID), slog.Int("notas_cedidas", c.NotasCedidas), slog.Int("excluidas", len(exc)))
	}
	span.SetAttributes(attribute.Int("notas_cedidas", c.NotasCedidas), attribute.Bool("replay", replay))
	return c, exc, replay, nil
}
