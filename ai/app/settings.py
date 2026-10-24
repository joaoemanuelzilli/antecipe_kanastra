"""
Configuração
============

Lida das variáveis de ambiente (segredos e infraestrutura), como no backend Go.
A chave do OpenRouter nunca é logada nem devolvida pela API.
"""

from dataclasses import dataclass
from os import getenv
from urllib.parse import quote

MODELO_PADRAO = "deepseek/deepseek-v4-flash"


@dataclass(frozen=True)
class Settings:
    api_url: str  # API do Antecipe (backend Go)
    modelo: str  # id do modelo no OpenRouter
    chave_configurada: bool
    db_url: str  # Postgres das sessões do chat (schema "ai")
    cors_origens: list[str]
    log_level: str


def carregar() -> Settings:
    usuario = getenv("DB_USER", "antecipe")
    senha = quote(getenv("DB_PASS", ""), safe="")
    host = getenv("DB_HOST", "localhost")
    porta = getenv("DB_PORT", "5433")
    banco = getenv("DB_DATABASE", "antecipe")
    origens = getenv("CORS_ORIGINS", "http://localhost:5173,http://localhost:3000")
    return Settings(
        api_url=getenv("ANTECIPE_API_URL", "http://localhost:8080"),
        modelo=getenv("AI_MODELO", "").strip() or MODELO_PADRAO,
        chave_configurada=bool(getenv("OPENROUTER_API_KEY", "").strip()),
        db_url=f"postgresql+psycopg://{usuario}:{senha}@{host}:{porta}/{banco}",
        cors_origens=[o.strip() for o in origens.split(",") if o.strip()],
        log_level=getenv("LOG_LEVEL", "info"),
    )
