# Serviço de IA do Antecipe: AgentOS (Agno) servindo o assistente na porta 7777.
FROM python:3.13-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PYTHONPATH=/app \
    AGNO_TELEMETRY=false

WORKDIR /app
COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY app/ app/
COPY agents/ agents/
COPY tools/ tools/
COPY db/ db/

RUN useradd --uid 10001 --no-create-home antecipe
USER antecipe
EXPOSE 7777
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "7777", "--no-access-log"]
