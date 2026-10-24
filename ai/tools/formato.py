"""
Formatação pt-BR
================

O assistente recebe valores já formatados ("R$ 9.803,92", "21/12/2026"): o modelo não precisa
converter centavos nem datas, e repete exatamente o que a tela mostra.
"""

import re
from datetime import UTC, datetime, timedelta, timezone

# Brasília não tem horário de verão desde 2019.
BRASILIA = timezone(timedelta(hours=-3))
_FRACAO = re.compile(r"\.(\d+)")


def reais(centavos: int | None) -> str | None:
    if centavos is None:
        return None
    sinal = "-" if centavos < 0 else ""
    inteiro, resto = divmod(abs(int(centavos)), 100)
    return f"{sinal}R$ {inteiro:,}".replace(",", ".") + f",{resto:02d}"


def data(iso: str | None) -> str | None:
    """'2026-12-21' → '21/12/2026'."""
    if not iso:
        return None
    try:
        return datetime.strptime(iso[:10], "%Y-%m-%d").strftime("%d/%m/%Y")
    except ValueError:
        return iso


def data_hora(iso: str | None) -> str | None:
    """'2026-10-24T17:03:10.123456789Z' → '24/10/2026 14:03' (horário de Brasília)."""
    if not iso:
        return None
    # O Go manda até 9 casas de fração de segundo; o Python lê até 6.
    texto = _FRACAO.sub(lambda m: "." + m.group(1)[:6], iso.replace("Z", "+00:00"))
    try:
        momento = datetime.fromisoformat(texto)
    except ValueError:
        return iso
    if momento.tzinfo is None:
        momento = momento.replace(tzinfo=UTC)
    return momento.astimezone(BRASILIA).strftime("%d/%m/%Y %H:%M")


def duracao(ms: int | None) -> str | None:
    """8120 → '8,1 s' · 72000 → '1 min 12 s'."""
    if ms is None:
        return None
    segundos = ms / 1000
    if segundos < 60:
        return f"{segundos:.1f} s".replace(".", ",")
    minutos, resto = divmod(round(segundos), 60)
    return f"{minutos} min {resto} s"


def inteiro(n: int | None) -> str | None:
    """500000 → '500.000'."""
    if n is None:
        return None
    return f"{n:,}".replace(",", ".")
