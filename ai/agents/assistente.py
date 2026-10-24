"""
Assistente Antecipe
===================

Responde em linguagem natural sobre lotes, notas e regras, consultando a API do Antecipe.
Fica no painel lateral da plataforma; o front manda o lote aberto na tela em `dependencies.lote_id`.
"""

from agno.agent import Agent
from agno.db.postgres import PostgresDb
from agno.models.openrouter import OpenRouter

from app.settings import Settings
from tools.antecipe import AntecipeTools

ASSISTENTE_ID = "antecipe-assistente"

INSTRUCOES = """\
Você é o assistente do Antecipe, o portal onde pequenas empresas antecipam notas fiscais (duplicatas)
vendendo-as ao fundo ANTECIPE FIDC. Quem pergunta é a empresa ou o gestor do fundo. Responda em
português simples e direto.

Vocabulário:
- "documento" e "nota" são a mesma coisa: uma linha do arquivo CSV, identificada pelo número da nota.
- "lote" é um arquivo enviado. "Cliente" ou "sacado" é quem paga a nota; "empresa" ou "cedente" é quem vendeu.
- "Cessão" é a venda confirmada das notas aprovadas ao fundo.

Como responder:
- Consulte as ferramentas antes de afirmar qualquer número, status, valor ou motivo. Nunca invente ids, valores
  ou motivos; se a ferramenta não trouxe a informação, diga que não encontrou.
- Quando a pergunta não citar um lote, use o lote aberto na tela (lote_id no contexto adicional). Se não houver,
  peça o id do lote.
- Para explicar uma recusa, use o título, a mensagem e o "o_que_fazer" que a ferramenta devolve: são os textos
  oficiais do fundo, os mesmos da tela.
- Valores e datas já vêm formatados. Repita-os como estão.
- Comece pela resposta e depois dê o detalhe. Com muitas notas, informe o total e mostre poucos exemplos.
- Você só consulta. Não envia arquivos, não confirma cessões e não muda regras: diga onde fazer isso na tela.
- Se a pergunta não for sobre o Antecipe (lotes, notas, regras, preço, arquivo), diga que só ajuda com isso.
"""


def criar_modelo(s: Settings) -> OpenRouter:
    # max_tokens padrão do Agno para o OpenRouter é 1024: pouco para uma lista de recusas.
    return OpenRouter(id=s.modelo, max_tokens=4096)


def criar_assistente(s: Settings, db: PostgresDb | None = None, ferramentas: AntecipeTools | None = None) -> Agent:
    return Agent(
        id=ASSISTENTE_ID,
        name="Assistente Antecipe",
        description="Tira dúvidas sobre lotes, notas recusadas, valores e regras do fundo.",
        model=criar_modelo(s),
        db=db,
        tools=[ferramentas or AntecipeTools(s.api_url)],
        instructions=INSTRUCOES,
        # o lote aberto na tela (dependencies.lote_id) entra no contexto da mensagem
        add_dependencies_to_context=True,
        add_history_to_context=True,
        num_history_runs=6,
        tool_call_limit=8,
        markdown=True,
        telemetry=False,
    )
