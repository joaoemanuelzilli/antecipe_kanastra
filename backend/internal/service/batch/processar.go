package batch

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"antecipe/internal/domain"
	"antecipe/internal/platform/apperrors"
	"antecipe/internal/platform/observability"
	"antecipe/internal/service/engine"
)

var errFalhaGenerica = domain.ErroLote{Codigo: "FALHA_PROCESSAMENTO", Mensagem: "Não conseguimos processar o arquivo. Nada foi gravado. Tente de novo."}

// etapaAtiva acompanha a etapa corrente: span filho, tempo e marcação no banco.
type etapaAtiva struct {
	nome   string
	inicio time.Time
	span   trace.Span
}

// processar roda o lote inteiro. Logs e spans levam só metadados (id, etapa, contagens, tempos).
func (s *Servico) processar(ctx context.Context, t trabalho) {
	inicio := time.Now()
	ctx, span := observability.Tracer().Start(ctx, "lote.processar", trace.WithAttributes(attribute.String("lote_id", t.id)))
	defer span.End()
	defer os.Remove(t.caminho)
	defer func() {
		s.mu.Lock()
		delete(s.progresso, t.id)
		s.mu.Unlock()
	}()

	var atual *etapaAtiva
	trocarEtapa := func(nome string) {
		if atual != nil && atual.nome == nome {
			return
		}
		if atual != nil {
			s.encerrarEtapa(ctx, t.id, atual)
		}
		_, sp := observability.Tracer().Start(ctx, "lote."+nome, trace.WithAttributes(attribute.String("lote_id", t.id)))
		atual = &etapaAtiva{nome: nome, inicio: time.Now(), span: sp}
		s.atualizar(t.id, func(p *Progresso) { p.Status, p.Etapa, p.Processadas = domain.LoteProcessando, nome, 0 })
		if err := s.d.Lotes.MarcarEtapa(ctx, t.id, domain.LoteProcessando, nome); err != nil {
			s.d.Log.ErrorContext(ctx, "falha ao marcar etapa", slog.String("lote_id", t.id), slog.String("erro", tipoErro(err)))
		}
	}
	avancar := func(n int) { s.atualizar(t.id, func(p *Progresso) { p.Processadas = n }) }

	falhar := func(e domain.ErroLote, causa error) {
		if atual != nil {
			atual.span.SetStatus(codes.Error, e.Codigo)
			s.encerrarEtapa(ctx, t.id, atual)
		}
		span.SetStatus(codes.Error, e.Codigo)
		if err := s.d.Lotes.FalharLote(context.WithoutCancel(ctx), t.id, e); err != nil {
			s.d.Log.ErrorContext(ctx, "falha ao marcar erro do lote", slog.String("lote_id", t.id), slog.String("erro", tipoErro(err)))
		}
		s.lotesTotal.Add(ctx, 1, metric.WithAttributes(attribute.String("status", domain.LoteErro)))
		s.d.Log.ErrorContext(ctx, "lote com erro", slog.String("lote_id", t.id), slog.String("codigo", e.Codigo), slog.String("erro", tipoErro(causa)))
	}
	defer func() {
		if r := recover(); r != nil {
			falhar(errFalhaGenerica, fmt.Errorf("panic: %T", r))
		}
	}()

	lote, err := s.d.Lotes.ObterLote(ctx, t.id)
	if err != nil {
		falhar(errFalhaGenerica, err)
		return
	}

	// 1. leitura
	trocarEtapa(domain.EtapaLendo)
	f, err := os.Open(t.caminho)
	if err != nil {
		falhar(errFalhaGenerica, err)
		return
	}
	notas, err := s.d.Leitor.Ler(f, avancar)
	f.Close()
	if err != nil {
		if ae := apperrors.From(err); ae.Status < 500 {
			falhar(domain.ErroLote{Codigo: ae.Code, Mensagem: ae.Message}, err)
		} else {
			falhar(errFalhaGenerica, err)
		}
		return
	}
	avancar(len(notas))

	// 2–4. regras, concentração e preço (o motor avisa a troca de etapa)
	versao, err := s.d.Regras.Obter(ctx, lote.RegrasVersao)
	if err != nil {
		falhar(errFalhaGenerica, err)
		return
	}
	trocarEtapa(domain.EtapaValidando)
	ectx, err := s.contextoMotor(ctx, versao, lote.Precificacao, lote.DataBase, "")
	if err != nil {
		falhar(errFalhaGenerica, err)
		return
	}
	ectx.AoAvancar = func(etapa string, n int) {
		trocarEtapa(etapa)
		if etapa == domain.EtapaConcentracao {
			n = len(notas)
		}
		avancar(n)
	}
	saida := engine.Validar(notas, versao.Regras, ectx)

	// 5. gravação
	trocarEtapa(domain.EtapaSalvando)
	if err := s.d.Lotes.GravarResultados(ctx, t.id, saida.Resultados, avancar); err != nil {
		falhar(errFalhaGenerica, err)
		return
	}
	duracao := time.Since(inicio)
	err = s.d.Lotes.ConcluirLote(ctx, t.id, domain.ResumoLote{
		Totais: saida.Totais, Motivos: saida.Motivos, ErrosConfig: saida.ErrosConfig, DuracaoMs: int(duracao.Milliseconds()),
	})
	if err != nil {
		falhar(errFalhaGenerica, err)
		return
	}
	s.encerrarEtapa(ctx, t.id, atual)

	span.SetAttributes(attribute.Int("linhas", saida.Totais.Linhas), attribute.Int("aprovadas", saida.Totais.Aprovadas), attribute.Int("recusadas", saida.Totais.Recusadas))
	s.lotesTotal.Add(ctx, 1, metric.WithAttributes(attribute.String("status", domain.LoteConcluido)))
	s.notasTotal.Add(ctx, int64(saida.Totais.Aprovadas), metric.WithAttributes(attribute.String("status", "aprovada")))
	s.notasTotal.Add(ctx, int64(saida.Totais.Recusadas), metric.WithAttributes(attribute.String("status", "recusada")))
	s.duracaoLote.Record(ctx, duracao.Seconds())
	s.d.Log.InfoContext(ctx, "lote concluido",
		slog.String("lote_id", t.id), slog.Int("linhas", saida.Totais.Linhas), slog.Int("aprovadas", saida.Totais.Aprovadas),
		slog.Int("recusadas", saida.Totais.Recusadas), slog.Int("versao_regras", versao.Numero), slog.Int64("duracao_ms", duracao.Milliseconds()))
}

func (s *Servico) encerrarEtapa(ctx context.Context, id string, e *etapaAtiva) {
	d := time.Since(e.inicio)
	e.span.End()
	s.duracaoEtap.Record(ctx, d.Seconds(), metric.WithAttributes(attribute.String("etapa", e.nome)))
	s.d.Log.InfoContext(ctx, "etapa concluida", slog.String("lote_id", id), slog.String("etapa", e.nome), slog.Int64("duracao_ms", d.Milliseconds()))
}

// tipoErro descreve o erro sem o texto dele (que pode carregar dados do arquivo).
func tipoErro(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%T", err)
}
