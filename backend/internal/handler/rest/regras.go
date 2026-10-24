package rest

import (
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"antecipe/internal/domain"
	"antecipe/internal/domain/rules"
	"antecipe/internal/platform/apperrors"
	"antecipe/internal/service/message"
)

// exigeGestor protege as rotas do gestor: vale a sessão de uma conta gestor ou, por compatibilidade,
// o header X-Gestor-Senha. A senha nunca é logada.
func (h *Handlers) exigeGestor(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if u := usuarioDe(r); u != nil && u.Perfil == domain.PerfilGestor {
			next(w, r)
			return
		}
		senha := r.Header.Get("X-Gestor-Senha")
		if senha != "" && h.d.GestorSenha != "" && subtle.ConstantTimeCompare([]byte(senha), []byte(h.d.GestorSenha)) == 1 {
			next(w, r)
			return
		}
		if u := usuarioDe(r); u != nil {
			EscreverErro(w, r, h.d.Log, apperrors.New(http.StatusForbidden, "PERMISSAO_NEGADA", "Só o gestor do fundo pode fazer isso."))
			return
		}
		if h.d.GestorSenha == "" && senha != "" {
			EscreverErro(w, r, h.d.Log, apperrors.Unavailable("GESTOR_DESABILITADO", "A área do gestor está desligada (GESTOR_SENHA não configurada)."))
			return
		}
		EscreverErro(w, r, h.d.Log, apperrors.Unauthorized("SENHA_INVALIDA", "Entre com uma conta de gestor (ou envie a senha do gestor)."))
	}
}

func (h *Handlers) verificarGestor(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) obterRegras(w http.ResponseWriter, r *http.Request) {
	e, err := h.d.Regras.Publicadas(r.Context())
	if err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	p := paraPrecificacao(e.Precificacao)
	p.FonteMetodo = "regras publicadas"
	sob := e.Sobrescritos
	if sob == nil {
		sob = []string{}
	}
	for _, s := range sob {
		if s == "metodo" {
			p.FonteMetodo = "config/antecipe.yaml"
		}
	}
	EscreverJSON(w, http.StatusOK, map[string]any{
		"versao": e.Versao.Numero, "publicada_em": e.Versao.PublicadaEm, "autor": e.Versao.Autor, "regras": e.Versao.Regras,
		"catalogo": rules.Catalogo(),
		"precificacao_efetiva": map[string]any{
			"metodo": p.Metodo, "taxa_mensal_percentual": p.TaxaMensalPercentual, "base_dias": p.BaseDias,
			"descricao": p.Descricao, "fonte_metodo": p.FonteMetodo, "sobrescritos_pelo_yaml": sob,
		},
		"catalogo_precificacao": map[string]any{"parametros": rules.CatalogoPrecificacao()},
	})
}

func (h *Handlers) listarVersoes(w http.ResponseWriter, r *http.Request) {
	vs, err := h.d.Regras.Versoes(r.Context())
	if err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	out := make([]map[string]any, 0, len(vs))
	for _, v := range vs {
		out = append(out, map[string]any{"versao": v.Numero, "autor": v.Autor, "publicada_em": v.PublicadaEm, "resumo": v.Resumo})
	}
	EscreverJSON(w, http.StatusOK, map[string]any{"versoes": out})
}

func (h *Handlers) obterVersao(w http.ResponseWriter, r *http.Request) {
	n, err := strconv.Atoi(r.PathValue("versao"))
	if err != nil {
		EscreverErro(w, r, h.d.Log, apperrors.NotFound("VERSAO_NAO_ENCONTRADA", "Versão das regras não encontrada."))
		return
	}
	v, err := h.d.Regras.Versao(r.Context(), n)
	if err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	EscreverJSON(w, http.StatusOK, map[string]any{"versao": v.Numero, "autor": v.Autor, "publicada_em": v.PublicadaEm, "regras": v.Regras})
}

type pedidoRegras struct {
	Regras *rules.Regras `json:"regras"`
	Autor  string        `json:"autor"`
	LoteID string        `json:"lote_id"`
}

func lerPedidoRegras(w http.ResponseWriter, r *http.Request) (pedidoRegras, error) {
	var p pedidoRegras
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		return p, apperrors.TooLarge("CORPO_GRANDE_DEMAIS", "O corpo da requisição é grande demais.")
	}
	if err := json.Unmarshal(b, &p); err != nil || p.Regras == nil {
		return p, apperrors.BadRequest("JSON_INVALIDO", "Envie {\"regras\": {...}} no formato do validadores_antecipe.json.")
	}
	for i := range p.Regras.Validadores {
		if p.Regras.Validadores[i].Parametros == nil {
			p.Regras.Validadores[i].Parametros = rules.Parametros{}
		}
	}
	return p, nil
}

func (h *Handlers) publicarRegras(w http.ResponseWriter, r *http.Request) {
	p, err := lerPedidoRegras(w, r)
	if err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	v, err := h.d.Regras.Publicar(r.Context(), *p.Regras, p.Autor)
	if err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	EscreverJSON(w, http.StatusCreated, map[string]any{"versao": v.Numero, "publicada_em": v.PublicadaEm.Format(time.RFC3339), "resumo": v.Resumo})
}

func (h *Handlers) previa(w http.ResponseWriter, r *http.Request) {
	p, err := lerPedidoRegras(w, r)
	if err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	pv, err := h.d.Regras.Previa(r.Context(), *p.Regras, p.LoteID)
	if err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	motivos := make([]map[string]any, 0, len(pv.Motivos))
	for _, m := range pv.Motivos {
		motivos = append(motivos, map[string]any{"codigo": m.Codigo, "titulo": message.Titulo(m.Codigo), "antes": m.Antes, "depois": m.Depois})
	}
	EscreverJSON(w, http.StatusOK, map[string]any{
		"lote_id": pv.LoteID, "linhas": pv.Linhas,
		"antes":    map[string]any{"regras_versao": pv.Antes.RegrasVersao, "aprovadas": pv.Antes.Aprovadas, "recusadas": pv.Antes.Recusadas, "liquido_centavos": pv.Antes.Liquido},
		"depois":   map[string]any{"aprovadas": pv.Depois.Aprovadas, "recusadas": pv.Depois.Recusadas, "liquido_centavos": pv.Depois.Liquido},
		"motivos":  motivos,
		"mudancas": pv.Mudancas,
	})
}
