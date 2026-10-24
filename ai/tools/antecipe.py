"""
Ferramentas do Antecipe
=======================

Consultam a API do Antecipe (backend Go) e devolvem JSON enxuto para o modelo. Só leem: nada aqui
envia lote, confirma cessão ou muda regra.

- Os textos de recusa (título, mensagem e ação) vêm prontos da API: o assistente diz o mesmo que a tela.
- Valores saem em reais e datas em dd/mm/aaaa: o modelo não converte nada.
- O lote padrão é o que está aberto na tela; o front o manda em `dependencies.lote_id`.
- Logs: só a rota (sem valores), o status e a duração. Nenhum dado do CSV vai para o log.
- A API só mostra um lote de conta à dona e aos gestores, então as consultas vão com a sessão de quem
  pergunta (`SESSAO_DO_USUARIO`, preenchida por requisição em app/main.py). Ela não entra no contexto do
  modelo nem no log.
"""

import json
import logging
import re
import time
from contextvars import ContextVar
from typing import Any

import httpx
from agno.run import RunContext
from agno.tools import Toolkit

from tools.formato import data, data_hora, duracao, inteiro, reais

log = logging.getLogger("antecipe.ai.ferramentas")

_UUID = re.compile(r"^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")
_CODIGO = re.compile(r"^[A-Z_]{3,60}$")
POR_PAGINA = 10

# sessão (cookie antecipe_sessao) de quem fez a pergunta; o nginx a repassa em X-Antecipe-Sessao
SESSAO_DO_USUARIO: ContextVar[str] = ContextVar("sessao_do_usuario", default="")

_STATUS_LOTE = {
    "na_fila": "Na fila: o lote ainda não começou a ser processado.",
    "processando": "Processando: o lote está passando pelas etapas agora.",
    "concluido": "Concluído: todas as etapas terminaram e o resultado está pronto.",
    "erro": "Com erro: o processamento não terminou.",
}


class ErroConsulta(Exception):
    """Falha que o modelo deve repassar ao usuário. A mensagem já está pronta para isso."""


def _json(valor: Any) -> str:
    return json.dumps(valor, ensure_ascii=False)


def _id_do_lote(run_context: RunContext | None, lote_id: str) -> str:
    lid = (lote_id or "").strip()
    if not lid and run_context is not None and run_context.dependencies:
        lid = str(run_context.dependencies.get("lote_id") or "").strip()
    if not lid:
        raise ErroConsulta("Nenhum lote foi informado e não há lote aberto na tela. Peça ao usuário o id do lote.")
    if not _UUID.match(lid):
        raise ErroConsulta("Esse id não tem o formato de um lote (UUID, ex.: 8f1c2a9e-0d4b-4c51-9a77-2f3e6b1d0c42).")
    return lid.lower()


def _mensagem_de_erro(r: httpx.Response) -> str:
    try:
        erro = r.json().get("erro") or {}
    except ValueError:
        erro = {}
    if erro.get("codigo") == "LOTE_NAO_ENCONTRADO":
        return "Não encontrei esse lote. Confira o id."
    return erro.get("mensagem") or f"A API do Antecipe recusou a consulta (status {r.status_code})."


class AntecipeTools(Toolkit):
    def __init__(self, api_url: str, timeout_s: float = 15.0, transport: httpx.AsyncBaseTransport | None = None):
        self.api_url = api_url.rstrip("/")
        self._timeout_s = timeout_s
        self._transport = transport  # os testes injetam um httpx.MockTransport
        super().__init__(
            name="antecipe",
            tools=[self.consultar_lote, self.buscar_notas, self.consultar_regras, self.consultar_formato_do_arquivo],
        )

    async def _get(self, rota: str, caminho: str, params: dict[str, Any] | None = None) -> Any:
        """GET na API. `rota` é o padrão sem valores (vai para o log); `caminho` é o endereço real."""
        inicio = time.perf_counter()
        status = 0
        sessao = SESSAO_DO_USUARIO.get()
        headers = {"Authorization": f"Bearer {sessao}"} if sessao else None
        try:
            async with httpx.AsyncClient(
                base_url=self.api_url, timeout=self._timeout_s, transport=self._transport
            ) as c:
                r = await c.get(caminho, params=params, headers=headers)
            status = r.status_code
        except httpx.HTTPError as e:
            raise ErroConsulta("A API do Antecipe não respondeu agora. Tente de novo em instantes.") from e
        finally:
            ms = round((time.perf_counter() - inicio) * 1000)
            log.info("consulta_api", extra={"campos": {"rota": rota, "status": status, "duracao_ms": ms}})
        if status >= 400:
            raise ErroConsulta(_mensagem_de_erro(r))
        return r.json()

    async def consultar_lote(self, run_context: RunContext, lote_id: str = "") -> str:
        """Mostra como foi (ou como está) o processamento de um lote: status, etapa atual, andamento,
        tempo total, quantas notas foram aprovadas e recusadas, o valor a receber, os motivos de recusa
        mais comuns e se a venda (cessão) já foi confirmada.

        Args:
            lote_id: id do lote (UUID). Deixe vazio para usar o lote aberto na tela do usuário.
        """
        try:
            lid = _id_do_lote(run_context, lote_id)
            lote = await self._get("GET /api/lotes/{id}", f"/api/lotes/{lid}")
        except ErroConsulta as e:
            return _json({"erro": str(e)})
        return _json(resumo_do_lote(lote))

    async def buscar_notas(
        self,
        run_context: RunContext,
        busca: str = "",
        status: str = "",
        motivo: str = "",
        pagina: int = 1,
        lote_id: str = "",
    ) -> str:
        """Procura notas (documentos) de um lote e explica cada uma: aprovada, com o cálculo do valor,
        ou recusada, com o motivo e o que fazer. Use para "por que a nota X foi recusada?", "quais notas
        do cliente Y entraram?" ou "mostre as recusadas por prazo".

        Args:
            busca: número da nota, nome ou CNPJ do cliente (sacado) ou da empresa (cedente). Vazio = todas.
            status: "aprovada" ou "recusada". Vazio = as duas.
            motivo: código do motivo principal da recusa, ex.: PRAZO_MAXIMO, DUPLICIDADE, CONCENTRACAO_MAXIMA_SACADO.
            pagina: página do resultado (10 notas por página), começando em 1.
            lote_id: id do lote (UUID). Deixe vazio para usar o lote aberto na tela do usuário.
        """
        try:
            lid = _id_do_lote(run_context, lote_id)
            params: dict[str, Any] = {"pagina": max(1, int(pagina or 1)), "por_pagina": POR_PAGINA}
            if busca.strip():
                params["busca"] = busca.strip()[:100]
            if status.strip():
                if status.strip().lower() not in ("aprovada", "recusada"):
                    raise ErroConsulta('O filtro de status aceita só "aprovada" ou "recusada".')
                params["status"] = status.strip().lower()
            if motivo.strip():
                if not _CODIGO.match(motivo.strip().upper()):
                    raise ErroConsulta("Código de motivo inválido. Use um código como PRAZO_MAXIMO.")
                params["motivo"] = motivo.strip().upper()
            pagina_api = await self._get("GET /api/lotes/{id}/notas", f"/api/lotes/{lid}/notas", params)
        except ErroConsulta as e:
            return _json({"erro": str(e)})
        return _json(resumo_das_notas(pagina_api))

    async def consultar_regras(self) -> str:
        """Mostra as regras do fundo em vigor (a versão publicada): quais estão ligadas, seus limites
        (prazo máximo, valor mínimo, concentração etc.) e como o preço é calculado.
        """
        try:
            regras = await self._get("GET /api/regras", "/api/regras")
        except ErroConsulta as e:
            return _json({"erro": str(e)})
        return _json(resumo_das_regras(regras))

    async def consultar_formato_do_arquivo(self) -> str:
        """Mostra como o arquivo CSV deve ser montado (colunas, separador, tamanho máximo), a
        data-base padrão e para que serve a coluna opcional DATA_AQUISICAO. Use para dúvidas sobre como
        preparar ou corrigir o arquivo.
        """
        try:
            cfg = await self._get("GET /api/config", "/api/config")
        except ErroConsulta as e:
            return _json({"erro": str(e)})
        upload = cfg.get("upload") or {}
        return _json(
            {
                "fundo": (cfg.get("fundo") or {}).get("nome"),
                "data_base_padrao": data(cfg.get("data_base_padrao")),
                "preco": (cfg.get("precificacao") or {}).get("descricao"),
                "separador": upload.get("separador"),
                "colunas": upload.get("colunas"),
                # Contrato v1.4: sem isto o modelo chuta o significado da coluna.
                "coluna_opcional": {
                    "DATA_AQUISICAO": "Pode ficar vazia. É a data em que o fundo adquire a nota e vira a data-base "
                    "dela: o prazo, os dias até o vencimento e o preço contam a partir dessa data. Vazia, vale a "
                    "data-base do envio (a padrão, se o envio não informar outra)."
                },
                "tamanho_maximo_mb": (upload.get("tamanho_maximo_bytes") or 0) // (1 << 20),
                "formato": "UTF-8 com cabeçalho, valores com vírgula decimal (1.234,56) e datas dd/mm/aaaa.",
            }
        )


def resumo_do_lote(lote: dict[str, Any]) -> dict[str, Any]:
    status = lote.get("status")
    etapas = lote.get("etapas") or []
    atual = next((e.get("titulo") for e in etapas if e.get("status") == "em_andamento"), None)
    progresso = lote.get("progresso") or {}
    out: dict[str, Any] = {
        "lote_id": lote.get("id"),
        "status": status,
        "status_explicado": _STATUS_LOTE.get(status, status),
        "etapas": [f"{e.get('titulo')}: {e.get('status')}" for e in etapas],
        "data_base": data(lote.get("data_base")),
        "regras_versao": lote.get("regras_versao"),
        "preco": (lote.get("precificacao") or {}).get("descricao"),
        "enviado_em": data_hora(lote.get("criado_em")),
        "concluido_em": data_hora(lote.get("concluido_em")),
        "duracao": duracao(lote.get("duracao_ms")),
    }
    if status in ("na_fila", "processando"):
        out["etapa_atual"] = atual
        out["andamento_da_etapa"] = (
            f"{inteiro(progresso.get('linhas_processadas'))} de {inteiro(progresso.get('linhas_total'))} linhas "
            f"({str(progresso.get('percentual', 0)).replace('.', ',')}%)"
        )
        if lote.get("posicao_fila"):
            out["posicao_na_fila"] = lote["posicao_fila"]
    totais = lote.get("totais")
    if totais:
        out["totais"] = {
            "linhas": totais.get("linhas"),
            "aprovadas": totais.get("aprovadas"),
            "recusadas": totais.get("recusadas"),
            "valor_de_face_das_aprovadas": reais(totais.get("face_centavos")),
            "desconto": reais(totais.get("desconto_centavos")),
            "liquido_a_receber": reais(totais.get("liquido_centavos")),
        }
    out["motivos_de_recusa"] = [
        {
            "codigo": m.get("codigo"),
            "titulo": m.get("titulo"),
            "quantidade": m.get("quantidade"),
            "o_que_fazer": m.get("acao"),
        }
        for m in lote.get("motivos") or []
    ]
    avisos = [e.get("mensagem") for e in lote.get("erros_configuracao") or []]
    if avisos:
        out["avisos_de_configuracao"] = avisos
    cessao = lote.get("cessao")
    out["cessao"] = (
        {
            "confirmada_em": data_hora(cessao.get("confirmada_em")),
            "notas_cedidas": cessao.get("notas_cedidas"),
            "valor_de_face": reais(cessao.get("face_centavos")),
            "desconto": reais(cessao.get("desconto_centavos")),
            "liquido": reais(cessao.get("liquido_centavos")),
        }
        if cessao
        else "ainda não confirmada"
    )
    if lote.get("erro"):
        out["erro"] = (lote["erro"] or {}).get("mensagem")
    return out


def _parte(p: dict[str, Any] | None) -> str | None:
    if not p:
        return None
    nome, doc = p.get("nome") or "", p.get("documento") or ""
    return f"{nome} ({doc})" if doc else nome


def resumo_das_notas(pagina: dict[str, Any]) -> dict[str, Any]:
    notas = []
    for n in pagina.get("itens") or []:
        item: dict[str, Any] = {
            "linha_no_arquivo": n.get("linha"),
            "numero": n.get("numero"),
            "tipo": n.get("tipo"),
            "status": n.get("status"),
            "cliente_sacado": _parte(n.get("sacado")),
            "empresa_cedente": _parte(n.get("cedente")),
            "vencimento": data(n.get("vencimento")),
            "dias_ate_o_vencimento": n.get("dias"),
            "valor": reais(n.get("face_centavos")),
        }
        if n.get("status") == "aprovada":
            item["desconto"] = reais(n.get("desconto_centavos"))
            item["liquido"] = reais(n.get("liquido_centavos"))
            if n.get("calculo"):
                item["calculo"] = n["calculo"].get("texto")
        else:
            # Só os textos oficiais; os "dados" técnicos ficam de fora (a primeira recusa é a principal).
            item["recusas"] = [
                {
                    "codigo": r.get("codigo"),
                    "titulo": r.get("titulo"),
                    "mensagem": r.get("mensagem"),
                    "o_que_fazer": r.get("acao"),
                }
                for r in n.get("recusas") or []
            ]
        notas.append(item)
    out: dict[str, Any] = {
        "total_encontrado": pagina.get("total", 0),
        "pagina": pagina.get("pagina", 1),
        "total_paginas": pagina.get("total_paginas", 0),
        "notas": notas,
    }
    if out["total_encontrado"] > len(notas):
        out["observacao"] = (
            f"Mostrando {len(notas)} de {inteiro(out['total_encontrado'])}. "
            "Peça a próxima página ou use os filtros da tabela na tela."
        )
    return out


def resumo_das_regras(r: dict[str, Any]) -> dict[str, Any]:
    titulos = {c.get("codigo"): c.get("titulo") for c in r.get("catalogo") or []}
    regras = r.get("regras") or {}
    return {
        "versao": r.get("versao"),
        "publicada_em": data_hora(r.get("publicada_em")),
        "autor": r.get("autor"),
        "fundo": (regras.get("fundo") or {}).get("nome"),
        "preco": (r.get("precificacao_efetiva") or {}).get("descricao"),
        "regras": [
            {
                "codigo": v.get("codigo"),
                "regra": titulos.get(v.get("codigo"), v.get("codigo")),
                "ativa": v.get("ativo"),
                "parametros": v.get("parametros") or {},
            }
            for v in regras.get("validadores") or []
        ],
    }
