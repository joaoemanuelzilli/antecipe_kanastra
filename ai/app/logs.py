"""
Logs estruturados
=================

Mesma política do backend Go: uma linha JSON por evento, só com metadados (rota, status,
duração, ids). Nada de perguntas, respostas, dados do CSV, corpo de requisição ou chaves.
Erros saem só com o tipo da exceção, nunca com o texto (que pode carregar dados).
"""

import json
import logging
import secrets
import sys
import time
from datetime import UTC, datetime

# Loggers do Agno e do uvicorn: têm handlers próprios, que trocamos pelo JSON.
_AGNO = ("agno", "agno-team", "agno-workflow")
_UVICORN = ("uvicorn", "uvicorn.error", "uvicorn.access")
# Clientes HTTP logam a URL inteira em INFO, com a query (a busca pode ter nome e CNPJ do sacado).
_CLIENTES_HTTP = ("httpx", "httpcore", "openai")


class FormatadorJSON(logging.Formatter):
    def format(self, r: logging.LogRecord) -> str:
        linha = {
            "time": datetime.fromtimestamp(r.created, UTC).isoformat(timespec="milliseconds"),
            "level": r.levelname,
            "logger": r.name,
            "msg": r.getMessage(),
        }
        campos = getattr(r, "campos", None)
        if isinstance(campos, dict):
            linha.update(campos)
        if r.exc_info and r.exc_info[0] is not None:
            linha["erro"] = r.exc_info[0].__name__
        return json.dumps(linha, ensure_ascii=False)


def configurar(nivel: str) -> None:
    handler = logging.StreamHandler(sys.stdout)
    handler.setFormatter(FormatadorJSON())
    raiz = logging.getLogger()
    raiz.handlers = [handler]
    raiz.setLevel(nivel.upper() if nivel.upper() in ("DEBUG", "INFO", "WARNING", "ERROR") else "INFO")
    for nome in _AGNO:
        # O Agno em debug loga mensagens e resultados de ferramentas: fica sempre em WARNING.
        lg = logging.getLogger(nome)
        lg.handlers = [handler]
        lg.setLevel(logging.WARNING)
        lg.propagate = False
    for nome in _UVICORN:
        lg = logging.getLogger(nome)
        lg.handlers = []
        lg.propagate = True
    for nome in _CLIENTES_HTTP:
        logging.getLogger(nome).setLevel(logging.WARNING)
    # o log de acesso é o LogDeAcesso abaixo (rota sem valores, sem query string)
    logging.getLogger("uvicorn.access").disabled = True


class LogDeAcesso:
    """Middleware ASGI: uma linha por requisição, com a rota como padrão (/agents/{agent_id}/runs).
    É ASGI puro para não atrapalhar o streaming (SSE) das respostas do agente."""

    def __init__(self, app):  # type: ignore[no-untyped-def]
        self.app = app
        self.log = logging.getLogger("antecipe.ai.http")

    async def __call__(self, scope, receive, send):  # type: ignore[no-untyped-def]
        if scope["type"] != "http":
            await self.app(scope, receive, send)
            return
        inicio = time.perf_counter()
        status = 500
        cabecalhos = dict(scope.get("headers") or [])
        request_id = cabecalhos.get(b"x-request-id", b"").decode("latin-1")[:64] or secrets.token_hex(8)

        async def enviar(msg):  # type: ignore[no-untyped-def]
            nonlocal status
            if msg["type"] == "http.response.start":
                status = msg["status"]
                msg.setdefault("headers", [])
                msg["headers"] = [*msg["headers"], (b"x-request-id", request_id.encode("latin-1"))]
            await send(msg)

        try:
            await self.app(scope, receive, enviar)
        finally:
            self.log.info(
                "requisicao",
                extra={
                    "campos": {
                        "request_id": request_id,
                        "metodo": scope.get("method"),
                        "rota": rota(scope),
                        "status": status,
                        "duracao_ms": round((time.perf_counter() - inicio) * 1000),
                    }
                },
            )


def rota(scope) -> str:  # type: ignore[no-untyped-def]
    """O padrão da rota que atendeu, sem os valores do caminho."""
    r = scope.get("route")
    if r is not None and getattr(r, "path", None):
        return r.path
    caminho: str = scope.get("path", "")
    for nome, valor in (scope.get("path_params") or {}).items():
        caminho = caminho.replace(str(valor), "{" + nome + "}")
    return caminho if scope.get("path_params") else "sem_rota"
