import { describe, expect, it } from 'vitest';
import type { Lote } from '../api/types';
import { filterLots, getLotNetValue, getLotStatus, type LotFilterEntry } from './lot-filters';

function lot(id: string, overrides: Partial<Lote> = {}): Lote {
  return {
    id, status: 'concluido', etapa: null, etapas: [], progresso: { linhas_processadas: 30, linhas_total: 30, percentual: 100 },
    posicao_fila: null, data_base: '2026-10-22', regras_versao: 1,
    precificacao: { metodo: 'DESCONTO_COMPOSTO', taxa_mensal_percentual: 2, base_dias: 30, descricao: 'Juros compostos · 2% ao mês' },
    criado_em: '2026-10-24T12:00:00Z', iniciado_em: '2026-10-24T12:00:00Z', concluido_em: '2026-10-24T12:00:09Z', duracao_ms: 9000,
    totais: { linhas: 30, aprovadas: 19, recusadas: 11, face_centavos: 1000000, desconto_centavos: 19608, liquido_centavos: 980392 },
    motivos: [], erros_configuracao: [], cessao: null, erro: null,
    ...overrides,
  };
}
function entry(id: string, date: string, overrides: Partial<LotFilterEntry> = {}): LotFilterEntry {
  return { local: { id, arquivo: `${id}.csv`, criado_em: date }, lot: lot(id), ...overrides };
}
const ids = (entries: LotFilterEntry[]) => entries.map(item => item.local.id);
function withValue(id: string, value: number | null, date = '2026-10-24T12:00:00Z'): LotFilterEntry {
  return entry(id, date, { lot: lot(id, { totais: value === null ? null : { linhas: 1, aprovadas: 1, recusadas: 0, face_centavos: value, desconto_centavos: 0, liquido_centavos: value } }) });
}

describe('status dos resumos de lotes', () => {
  it('classifica em análise somente quando o backend informa fila ou processamento', () => {
    for (const status of ['na_fila', 'processando'] as const) expect(getLotStatus(entry(status, '', { lot: lot(status, { status, totais: null }) }))).toBe('em_analise');
    const pending = entry('pending', '', { lot: undefined, loading: true });
    expect(getLotStatus(pending)).toBe('desconhecido');
    expect(filterLots([pending], { status: 'em_analise' })).toEqual([]);
    expect(filterLots([pending])).toEqual([pending]);
  });
  it('considera concluído sem cessão analisado mesmo quando todas as notas foram recusadas', () => {
    const allRejected = lot('rejected', { totais: { linhas: 30, aprovadas: 0, recusadas: 30, face_centavos: 0, desconto_centavos: 0, liquido_centavos: 0 } });
    const row = entry('rejected', '', { lot: allRejected });
    expect(getLotStatus(row)).toBe('analisados');
    expect(filterLots([row], { status: 'cedidos' })).toEqual([]);
    expect(getLotNetValue(row)).toBe(0);
  });
  it('usa a cessão confirmada e conserva seu status e valor durante erro de refetch', () => {
    const confirmed = lot('sold', { cessao: { id: 'cession', lote_id: 'sold', confirmada_em: '2026-10-24T12:01:00Z', notas_cedidas: 18, face_centavos: 100000, desconto_centavos: 1000, liquido_centavos: 99000 } });
    const row = entry('sold', '', { lot: confirmed, error: new Error('Refetch indisponível') });
    expect(getLotStatus(row)).toBe('cedidos'); expect(getLotNetValue(row)).toBe(99000);
    expect(filterLots([row], { status: 'com_erro' })).toEqual([]);
    expect(getLotStatus(entry('cached', '', { error: new Error('Rede indisponível') }))).toBe('analisados');
  });
  it('separa falha da API sem dados, falha de processamento e situação desconhecida', () => {
    const apiFailure = entry('api', '', { lot: undefined, loading: false, error: new Error('Não encontrado') });
    const processFailure = entry('process', '', { lot: lot('process', { status: 'erro', totais: null, erro: { codigo: 'FALHA_PROCESSAMENTO', mensagem: 'Falhou' } }) });
    const unknown = entry('unknown', '', { lot: undefined, loading: false });
    expect(getLotStatus(apiFailure)).toBe('com_erro'); expect(getLotNetValue(apiFailure)).toBeNull();
    expect(getLotStatus(processFailure)).toBe('com_erro'); expect(getLotNetValue(processFailure)).toBeNull();
    expect(getLotStatus(unknown)).toBe('desconhecido'); expect(filterLots([unknown], { status: 'analisados' })).toEqual([]);
    expect(filterLots([apiFailure, processFailure, unknown], { status: 'com_erro' })).toEqual([apiFailure, processFailure]);
  });
});

describe('busca e data-base sem inferências de envio', () => {
  const rows = [
    entry('id-aurora', '2026-10-24T12:00:00Z', { local: { id: 'id-aurora', nome: 'Antecipação São José', arquivo: 'Notas Indústria.csv', criado_em: '2026-10-24T12:00:00Z' } }),
    entry('id-serra', '2026-10-25T12:00:00Z', { lot: lot('id-serra', { data_base: '2026-10-23' }) }),
    entry('sem-base', '2026-10-22T12:00:00Z', { lot: undefined }),
  ];
  it('busca por nome, arquivo, id e data-base brasileira, ignorando acentos e caixa', () => {
    expect(ids(filterLots(rows, { busca: ' antecipacao   sao jose ' }))).toEqual(['id-aurora']);
    expect(ids(filterLots(rows, { busca: 'industria.CSV' }))).toEqual(['id-aurora']);
    expect(ids(filterLots(rows, { busca: 'ID-SERRA' }))).toEqual(['id-serra']);
    expect(ids(filterLots(rows, { busca: '22/10/2026' }))).toEqual(['id-aurora']);
    expect(ids(filterLots(rows, { busca: '2026-10-23' }))).toEqual(['id-serra']);
  });
  it('usa somente data_base do resumo, nunca criado_em como substituto', () => {
    expect(ids(filterLots(rows, { dataBase: '2026-10-22' }))).toEqual(['id-aurora']);
    expect(filterLots(rows, { dataBase: '2026-10-24' })).toEqual([]);
    expect(filterLots(rows, { busca: '24/10/2026' })).toEqual([]);
    expect(ids(filterLots(rows, { dataBase: '' }))).toEqual(['id-serra', 'id-aurora', 'sem-base']);
  });
  it('combina busca, situação e data-base sem alterar os critérios de outros lotes', () => {
    expect(ids(filterLots(rows, { busca: 'sao', status: 'analisados', dataBase: '2026-10-22' }))).toEqual(['id-aurora']);
    expect(filterLots(rows, { busca: 'sao', status: 'em_analise', dataBase: '2026-10-22' })).toEqual([]);
  });
});

describe('ordenação estável de resumos disponíveis', () => {
  it('ordena recentes e antigos pela data de envio local, incluindo offsets de fuso', () => {
    const rows = [entry('older', '2026-10-24T09:00:00-03:00'), entry('newer', '2026-10-24T12:01:00Z'), entry('invalid-date', 'data inválida')];
    expect(ids(filterLots(rows))).toEqual(['newer', 'older', 'invalid-date']);
    expect(ids(filterLots(rows, { ordenacao: 'antigos' }))).toEqual(['older', 'newer', 'invalid-date']);
  });
  it('ordena maior valor por cessão final, mantendo zero válido e ausências por último', () => {
    const confirmed = withValue('confirmed', 1000000);confirmed.lot!.cessao = { id: 'sale', lote_id: 'confirmed', confirmada_em: '2026-10-24T12:01:00Z', notas_cedidas: 1, face_centavos: 100, desconto_centavos: 0, liquido_centavos: 100 };
    const zeroSale = withValue('zero-sale', 900000);zeroSale.lot!.cessao = { ...confirmed.lot!.cessao, lote_id: 'zero-sale', liquido_centavos: 0, face_centavos: 0 };
    const rows = [withValue('unknown', null, '2026-10-30T12:00:00Z'), zeroSale, confirmed, withValue('largest', 1000), withValue('negative', -1), entry('error', '2026-10-29T12:00:00Z', { lot: undefined, error: new Error('404') })];
    expect(ids(filterLots(rows, { ordenacao: 'maior_valor' }))).toEqual(['largest', 'confirmed', 'zero-sale', 'negative', 'unknown', 'error']);
  });
  it('desempata valores pela data mais nova e preserva a ordem original se a data empata', () => {
    const rows = [withValue('older', 100, '2026-10-23T12:00:00Z'), withValue('tie-first', 100), withValue('tie-second', 100)];
    expect(ids(filterLots(rows, { ordenacao: 'maior_valor' }))).toEqual(['tie-first', 'tie-second', 'older']);
  });
  it('não altera o array ou seus objetos e conserva propriedades extras usadas pela UI', () => {
    const first = { ...withValue('first', 100), queryIndex: 3 }, second = { ...withValue('second', 200), queryIndex: 8 };
    const frozen = Object.freeze([Object.freeze(first), Object.freeze(second)]);
    const result = filterLots(frozen, { ordenacao: 'maior_valor' });
    expect(result.map(item => item.queryIndex)).toEqual([8, 3]);
    expect(frozen[0]).toBe(first); expect(result[0]).toBe(second);
  });
});
