import json
import logging

import httpx
import pytest
from agno.run import RunContext

from app import logs
from tools.antecipe import SESSAO_DO_USUARIO, AntecipeTools

LOTE = "8f1c2a9e-0d4b-4c51-9a77-2f3e6b1d0c42"
OUTRO = "0a0b0c0d-1111-2222-3333-444455556666"

LOTE_CONCLUIDO = {
    "id": LOTE,
    "status": "concluido",
    "etapa": None,
    "etapas": [{"codigo": "lendo", "titulo": "Lendo arquivo", "status": "concluida"}],
    "progresso": {"linhas_processadas": 500000, "linhas_total": 500000, "percentual": 100.0},
    "posicao_fila": None,
    "data_base": "2026-10-22",
    "regras_versao": 1,
    "precificacao": {"descricao": "Juros compostos · 2% ao mês"},
    "criado_em": "2026-10-24T17:03:10.123456789Z",
    "concluido_em": "2026-10-24T17:03:18.243Z",
    "duracao_ms": 8120,
    "totais": {
        "linhas": 500000,
        "aprovadas": 412345,
        "recusadas": 87655,
        "face_centavos": 98765432100,
        "desconto_centavos": 2345678900,
        "liquido_centavos": 96419753200,
    },
    "motivos": [
        {
            "codigo": "PRAZO_MAXIMO",
            "titulo": "Prazo acima do limite",
            "quantidade": 40211,
            "acao": "Antecipe essas notas mais perto do vencimento.",
        }
    ],
    "erros_configuracao": [],
    "cessao": None,
    "erro": None,
}

PAGINA_NOTAS = {
    "pagina": 1,
    "por_pagina": 10,
    "total": 87655,
    "total_paginas": 8766,
    "itens": [
        {
            "linha": 1204,
            "numero": "725714",
            "tipo": "Duplicata",
            "cedente": {"nome": "NLR INDÚSTRIA DE ESTOFADOS LTDA ME", "documento": "10.062.050/0001-57"},
            "sacado": {"nome": "TECIDOS SANTARÉM LTDA ME", "documento": "22.144.961/0001-02"},
            "vencimento": "2027-06-19",
            "dias": 240,
            "status": "recusada",
            "face_centavos": 227808,
            "desconto_centavos": None,
            "liquido_centavos": None,
            "recusas": [
                {
                    "codigo": "PRAZO_MAXIMO",
                    "titulo": "Prazo acima do limite",
                    "mensagem": "Esta nota vence em 240 dias e o fundo compra notas com até 180 dias.",
                    "acao": "Você pode antecipá-la a partir de 21/12/2026.",
                    "dados": {"dias": 240, "maximo": 180, "antecipavel_em": "2026-12-21"},
                }
            ],
            "calculo": None,
        },
        {
            "linha": 1205,
            "numero": "1-759172",
            "tipo": "Duplicata",
            "cedente": {"nome": "CANAÃ INDÚSTRIA DE ETIQUETAS LTDA", "documento": "57.543.525/0001-90"},
            "sacado": {"nome": "LOJA AURORA LTDA", "documento": "44.911.681/0001-15"},
            "vencimento": "2026-11-21",
            "dias": 30,
            "status": "aprovada",
            "face_centavos": 1000000,
            "desconto_centavos": 19608,
            "liquido_centavos": 980392,
            "recusas": [],
            "calculo": {"texto": "R$ 10.000,00 ÷ (1 + 2%)^(30 ÷ 30) = R$ 9.803,92 → desconto de R$ 196,08"},
        },
    ],
}


class FakeAPI:
    """Simula a API Go e guarda as requisições recebidas."""

    def __init__(self, rotas: dict[str, tuple[int, dict]]):
        self.rotas = rotas
        self.pedidos: list[httpx.Request] = []

    def __call__(self, req: httpx.Request) -> httpx.Response:
        self.pedidos.append(req)
        status, corpo = self.rotas.get(
            req.url.path, (404, {"erro": {"codigo": "LOTE_NAO_ENCONTRADO", "mensagem": "Lote não encontrado."}})
        )
        return httpx.Response(status, json=corpo)


def ferramentas(api: FakeAPI) -> AntecipeTools:
    return AntecipeTools("http://api:8080", transport=httpx.MockTransport(api))


def contexto(lote_id: str | None = LOTE) -> RunContext:
    return RunContext(run_id="run", session_id="sessao", dependencies={"lote_id": lote_id} if lote_id else None)


async def test_consultar_lote_usa_o_lote_aberto_na_tela():
    api = FakeAPI({f"/api/lotes/{LOTE}": (200, LOTE_CONCLUIDO)})
    out = json.loads(await ferramentas(api).consultar_lote(contexto()))
    assert api.pedidos[0].url.path == f"/api/lotes/{LOTE}"
    assert out["status"] == "concluido"
    assert out["totais"]["recusadas"] == 87655
    assert out["totais"]["liquido_a_receber"] == "R$ 964.197.532,00"
    assert out["duracao"] == "8,1 s"
    assert out["enviado_em"] == "24/10/2026 14:03"  # horário de Brasília
    assert out["motivos_de_recusa"][0]["o_que_fazer"] == "Antecipe essas notas mais perto do vencimento."
    assert out["cessao"] == "ainda não confirmada"


async def test_lote_citado_na_pergunta_tem_prioridade_sobre_o_da_tela():
    api = FakeAPI({f"/api/lotes/{OUTRO}": (200, {**LOTE_CONCLUIDO, "id": OUTRO})})
    out = json.loads(await ferramentas(api).consultar_lote(contexto(), lote_id=OUTRO.upper()))
    assert out["lote_id"] == OUTRO


async def test_sem_lote_pede_o_id():
    api = FakeAPI({})
    out = json.loads(await ferramentas(api).consultar_lote(contexto(None)))
    assert "Peça ao usuário o id do lote" in out["erro"]
    assert api.pedidos == []


async def test_id_que_nao_e_uuid_nao_chega_na_api():
    api = FakeAPI({})
    out = json.loads(await ferramentas(api).consultar_lote(contexto(), lote_id="../config"))
    assert "formato de um lote" in out["erro"]
    assert api.pedidos == []


async def test_lote_inexistente():
    out = json.loads(await ferramentas(FakeAPI({})).consultar_lote(contexto()))
    assert out["erro"] == "Não encontrei esse lote. Confira o id."


async def test_api_fora_do_ar():
    def cair(req: httpx.Request) -> httpx.Response:
        raise httpx.ConnectError("recusada", request=req)

    tk = AntecipeTools("http://api:8080", transport=httpx.MockTransport(cair))
    out = json.loads(await tk.consultar_lote(contexto()))
    assert "não respondeu" in out["erro"]


async def test_lote_em_processamento_mostra_etapa_e_andamento():
    em_andamento = {
        **LOTE_CONCLUIDO,
        "status": "processando",
        "etapas": [
            {"codigo": "lendo", "titulo": "Lendo arquivo", "status": "concluida"},
            {"codigo": "validando", "titulo": "Validando regras", "status": "em_andamento"},
        ],
        "progresso": {"linhas_processadas": 312000, "linhas_total": 500000, "percentual": 62.4},
        "totais": None,
        "motivos": [],
    }
    out = json.loads(await ferramentas(FakeAPI({f"/api/lotes/{LOTE}": (200, em_andamento)})).consultar_lote(contexto()))
    assert out["etapa_atual"] == "Validando regras"
    assert out["andamento_da_etapa"] == "312.000 de 500.000 linhas (62,4%)"
    assert "totais" not in out


async def test_buscar_notas_repassa_filtros_e_explica_recusas():
    api = FakeAPI({f"/api/lotes/{LOTE}/notas": (200, PAGINA_NOTAS)})
    out = json.loads(
        await ferramentas(api).buscar_notas(contexto(), busca=" 725714 ", status="Recusada", motivo="prazo_maximo")
    )
    params = dict(api.pedidos[0].url.params)
    assert params == {
        "pagina": "1",
        "por_pagina": "10",
        "busca": "725714",
        "status": "recusada",
        "motivo": "PRAZO_MAXIMO",
    }
    recusada, aprovada = out["notas"]
    assert recusada["recusas"][0] == {
        "codigo": "PRAZO_MAXIMO",
        "titulo": "Prazo acima do limite",
        "mensagem": "Esta nota vence em 240 dias e o fundo compra notas com até 180 dias.",
        "o_que_fazer": "Você pode antecipá-la a partir de 21/12/2026.",
    }
    assert recusada["vencimento"] == "19/06/2027"
    assert recusada["cliente_sacado"] == "TECIDOS SANTARÉM LTDA ME (22.144.961/0001-02)"
    assert aprovada["liquido"] == "R$ 9.803,92"
    assert "calculo" in aprovada
    assert out["observacao"].startswith("Mostrando 2 de 87.655")


async def test_buscar_notas_recusa_filtros_invalidos_sem_chamar_a_api():
    api = FakeAPI({})
    tk = ferramentas(api)
    assert "status" in json.loads(await tk.buscar_notas(contexto(), status="todas"))["erro"]
    assert "motivo" in json.loads(await tk.buscar_notas(contexto(), motivo="1; drop"))["erro"]
    assert api.pedidos == []


async def test_consultar_regras_usa_titulos_do_catalogo():
    regras = {
        "versao": 2,
        "publicada_em": "2026-10-24T18:10:00Z",
        "autor": "Rafael",
        "regras": {
            "fundo": {"nome": "ANTECIPE FIDC"},
            "validadores": [{"codigo": "PRAZO_MAXIMO", "ativo": True, "parametros": {"prazo_maximo_dias": 120}}],
        },
        "catalogo": [{"codigo": "PRAZO_MAXIMO", "titulo": "Prazo máximo"}],
        "precificacao_efetiva": {"descricao": "Juros compostos · 2% ao mês"},
    }
    out = json.loads(await ferramentas(FakeAPI({"/api/regras": (200, regras)})).consultar_regras())
    assert out["regras"] == [
        {"codigo": "PRAZO_MAXIMO", "regra": "Prazo máximo", "ativa": True, "parametros": {"prazo_maximo_dias": 120}}
    ]
    assert out["publicada_em"] == "24/10/2026 15:10"


async def test_consulta_vai_com_a_sessao_de_quem_pergunta():
    api = FakeAPI({f"/api/lotes/{LOTE}": (200, LOTE_CONCLUIDO)})
    marca = SESSAO_DO_USUARIO.set("token-da-empresa")
    try:
        await ferramentas(api).consultar_lote(contexto())
    finally:
        SESSAO_DO_USUARIO.reset(marca)
    assert api.pedidos[0].headers["authorization"] == "Bearer token-da-empresa"
    await ferramentas(api).consultar_lote(contexto())  # sem sessão: anônimo, sem header
    assert "authorization" not in api.pedidos[1].headers


async def test_formato_explica_a_data_de_aquisicao():
    config = {
        "fundo": {"nome": "ANTECIPE FIDC"},
        "data_base_padrao": "2026-10-22",
        "precificacao": {"descricao": "Juros compostos · 2% ao mês"},
        "upload": {
            "separador": ";",
            "colunas": ["NU_DOCUMENTO", "VALOR_NOMINAL", "DATA_VENCIMENTO_ORIGINAL", "DATA_AQUISICAO"],
            "tamanho_maximo_bytes": 300 << 20,
        },
    }
    out = json.loads(await ferramentas(FakeAPI({"/api/config": (200, config)})).consultar_formato_do_arquivo())
    assert out["data_base_padrao"] == "22/10/2026"
    assert out["tamanho_maximo_mb"] == 300
    assert "data em que o fundo adquire a nota" in out["coluna_opcional"]["DATA_AQUISICAO"]


async def test_logs_nao_levam_dados_da_nota(caplog: pytest.LogCaptureFixture):
    # mesma configuração de produção, capturando tudo que chega em qualquer logger
    logs.configurar("info")
    raiz = logging.getLogger()
    raiz.addHandler(caplog.handler)
    caplog.handler.setLevel(logging.DEBUG)
    api = FakeAPI({f"/api/lotes/{LOTE}/notas": (200, PAGINA_NOTAS)})
    await ferramentas(api).buscar_notas(contexto(), busca="TECIDOS SANTARÉM 22.144.961/0001-02")
    texto = " ".join(f"{r.getMessage()} {getattr(r, 'campos', '')}" for r in caplog.records)
    assert "GET /api/lotes/{id}/notas" in texto
    for proibido in ("TECIDOS", "22.144.961", "725714", LOTE):
        assert proibido not in texto


def test_run_context_nao_vai_para_o_schema_do_modelo():
    tk = AntecipeTools("http://api:8080")
    funcoes = tk.get_async_functions()
    assert set(funcoes) == {"consultar_lote", "buscar_notas", "consultar_regras", "consultar_formato_do_arquivo"}
    f = funcoes["buscar_notas"]
    f.process_entrypoint()
    assert "run_context" not in f.parameters["properties"]
    assert {"busca", "status", "motivo", "pagina", "lote_id"} <= set(f.parameters["properties"])
