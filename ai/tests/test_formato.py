from tools.formato import data, data_hora, duracao, inteiro, reais


def test_reais():
    assert reais(980392) == "R$ 9.803,92"
    assert reais(5) == "R$ 0,05"
    assert reais(-500) == "-R$ 5,00"
    assert reais(None) is None


def test_datas():
    assert data("2026-12-21") == "21/12/2026"
    assert data_hora("2026-10-24T17:03:10.123456789Z") == "24/10/2026 14:03"
    assert data_hora("2026-10-24T14:03:10-03:00") == "24/10/2026 14:03"
    assert data_hora(None) is None


def test_duracao_e_inteiro():
    assert duracao(8120) == "8,1 s"
    assert duracao(72000) == "1 min 12 s"
    assert inteiro(500000) == "500.000"
