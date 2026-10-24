import json
import logging
import os

import httpx
import pytest

# Sem chave e sem banco: o serviço precisa subir mesmo assim (o make up não pode quebrar).
os.environ["OPENROUTER_API_KEY"] = ""
os.environ.setdefault("DB_HOST", "127.0.0.1")
os.environ.setdefault("DB_PORT", "1")

from app.main import app  # noqa: E402


@pytest.fixture
def cliente() -> httpx.AsyncClient:
    return httpx.AsyncClient(transport=httpx.ASGITransport(app=app), base_url="http://ai")


def linhas_de_acesso(caplog: pytest.LogCaptureFixture) -> list[dict]:
    return [getattr(r, "campos", {}) for r in caplog.records if r.name == "antecipe.ai.http"]


async def test_status_sem_chave_avisa_que_o_assistente_esta_indisponivel(cliente, caplog):
    caplog.set_level(logging.INFO, logger="antecipe.ai.http")
    r = await cliente.get("/assistente/status", headers={"X-Request-Id": "abc123"})
    assert r.status_code == 200
    assert r.json() == {"disponivel": False, "agente_id": "antecipe-assistente", "modelo": "deepseek/deepseek-v4-flash"}
    assert r.headers["x-request-id"] == "abc123"
    acesso = linhas_de_acesso(caplog)[-1]
    assert acesso == {
        "request_id": "abc123",
        "metodo": "GET",
        "rota": "/assistente/status",
        "status": 200,
        "duracao_ms": acesso["duracao_ms"],
    }


async def test_log_de_acesso_usa_o_padrao_da_rota(cliente, caplog):
    caplog.set_level(logging.INFO, logger="antecipe.ai.http")
    r = await cliente.get("/agents/antecipe-assistente")
    assert r.status_code == 200
    assert r.json()["id"] == "antecipe-assistente"
    assert linhas_de_acesso(caplog)[-1]["rota"] == "/agents/{agent_id}"


def test_formatador_json_nao_vaza_texto_de_excecao():
    from app.logs import FormatadorJSON

    try:
        raise ValueError("CNPJ 22.144.961/0001-02 inválido")
    except ValueError:
        import sys

        rec = logging.LogRecord("x", logging.ERROR, __file__, 1, "falhou", None, sys.exc_info())
    linha = json.loads(FormatadorJSON().format(rec))
    assert linha["erro"] == "ValueError"
    assert "22.144.961" not in json.dumps(linha)
