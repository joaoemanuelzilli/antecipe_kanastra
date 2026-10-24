"""
Serviço de IA do Antecipe
=========================

O AgentOS serve o assistente por HTTP (POST /agents/antecipe-assistente/runs). O serviço é um
cliente da API do Antecipe, como o front: não lê as tabelas do backend, só guarda as sessões do
chat no schema "ai".
"""

import logging

from agno.os import AgentOS
from fastapi import FastAPI

from agents.assistente import ASSISTENTE_ID, criar_assistente
from app import logs
from app.settings import carregar
from db.session import sessoes

cfg = carregar()
logs.configurar(cfg.log_level)
log = logging.getLogger("antecipe.ai")

base = FastAPI(title="Antecipe · Assistente", version="1.0.0")


@base.get("/assistente/status")
def status() -> dict:
    """O front usa para mostrar ou esconder o painel do assistente."""
    return {"disponivel": cfg.chave_configurada, "agente_id": ASSISTENTE_ID, "modelo": cfg.modelo}


agent_os = AgentOS(
    id="antecipe-ai",
    name="Antecipe · Assistente",
    description="Assistente que responde sobre lotes, notas e regras do Antecipe.",
    agents=[criar_assistente(cfg, db=sessoes(cfg.db_url))],
    base_app=base,
    cors_allowed_origins=cfg.cors_origens,
    telemetry=False,
)
app = agent_os.get_app()
app.add_middleware(logs.LogDeAcesso)

if not cfg.chave_configurada:
    log.warning("OPENROUTER_API_KEY ausente: o assistente fica indisponível; o resto do Antecipe funciona normalmente")
log.info("assistente pronto", extra={"campos": {"modelo": cfg.modelo, "agente_id": ASSISTENTE_ID}})
