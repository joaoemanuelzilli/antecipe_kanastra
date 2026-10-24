package rest

import (
	"net/http"
	"strings"

	"antecipe/internal/domain"
	"antecipe/internal/platform/apperrors"
)

func (h *Handlers) ceder(w http.ResponseWriter, r *http.Request) {
	chave := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if len(chave) > 128 {
		EscreverErro(w, r, h.d.Log, apperrors.BadRequest("IDEMPOTENCY_KEY_INVALIDA", "A Idempotency-Key deve ter até 128 caracteres."))
		return
	}
	if _, _, err := h.loteVisivel(r, r.PathValue("id")); err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	c, exc, replay, err := h.d.Cessoes.Ceder(r.Context(), r.PathValue("id"), chave)
	if err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	if exc == nil {
		exc = []domain.Excluida{}
	}
	corpo := map[string]any{"cessao": paraCessao(&c), "notas_excluidas": exc}
	if replay {
		w.Header().Set("Idempotent-Replay", "true")
		EscreverJSON(w, http.StatusOK, corpo)
		return
	}
	EscreverJSON(w, http.StatusCreated, corpo)
}
