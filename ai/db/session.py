"""
Sessões do chat
===============

O histórico das conversas fica no mesmo Postgres do Antecipe, mas no schema "ai", separado das
tabelas do backend (schema "public"). O AgentOS cria o schema e as tabelas sozinho ao subir.
"""

from agno.db.postgres import PostgresDb


def sessoes(db_url: str) -> PostgresDb:
    return PostgresDb(id="antecipe-ai", db_url=db_url, db_schema="ai")
