"""Com OS_SECURITY_KEY definida (o make up gera a chave), o AgentOS exige "Authorization: Bearer <chave>".
No compose, só o nginx tem a chave e a injeta em /ai/: ela nunca chega ao navegador, e a porta 7777 não é publicada."""

import importlib

import httpx
import pytest


@pytest.fixture
def app_protegido(monkeypatch):
    monkeypatch.setenv("OS_SECURITY_KEY", "chave-de-teste")
    monkeypatch.setenv("OPENROUTER_API_KEY", "")
    monkeypatch.setenv("DB_HOST", "127.0.0.1")
    monkeypatch.setenv("DB_PORT", "1")
    import app.main as modulo

    modulo = importlib.reload(modulo)
    yield modulo.app
    monkeypatch.delenv("OS_SECURITY_KEY")
    importlib.reload(modulo)


async def test_sem_a_chave_o_assistente_recusa(app_protegido):
    c = httpx.AsyncClient(transport=httpx.ASGITransport(app=app_protegido), base_url="http://ai")
    for rota in ("/agents/antecipe-assistente", "/sessions", "/assistente/status"):
        r = await c.get(rota)
        assert r.status_code == 401, rota
    r = await c.get("/agents/antecipe-assistente", headers={"Authorization": "Bearer errada"})
    assert r.status_code == 401


async def test_com_a_chave_o_assistente_responde(app_protegido):
    c = httpx.AsyncClient(transport=httpx.ASGITransport(app=app_protegido), base_url="http://ai")
    r = await c.get("/agents/antecipe-assistente", headers={"Authorization": "Bearer chave-de-teste"})
    assert r.status_code == 200
    assert r.json()["id"] == "antecipe-assistente"
