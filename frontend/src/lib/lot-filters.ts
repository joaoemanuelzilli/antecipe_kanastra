import type { Lote, LoteLocal } from '../api/types';

/** Resumos já recebidos pela UI; este módulo não consulta notas ou endpoints. */
export interface LotFilterEntry {
  local: LoteLocal;
  lot?: Lote;
  loading?: boolean;
  error?: Error | null;
}
export type LotStatusFilter = 'todos' | 'em_analise' | 'analisados' | 'cedidos' | 'com_erro';
export type LotStatus = Exclude<LotStatusFilter, 'todos'> | 'desconhecido';
export type LotSort = 'recentes' | 'antigos' | 'maior_valor';
export interface LotFilters {
  busca?: string;
  status?: LotStatusFilter;
  dataBase?: string;
  ordenacao?: LotSort;
}
export const LOT_STATUS_LABELS: Record<LotStatusFilter, string> = {
  todos: 'Todos os lotes',
  em_analise: 'Em análise',
  analisados: 'Analisados',
  cedidos: 'Cedidos',
  com_erro: 'Com erro',
};
export const LOT_SORT_LABELS: Record<LotSort, string> = {
  recentes: 'Mais recentes',
  antigos: 'Mais antigos',
  maior_valor: 'Maior valor líquido',
};

export function getLotStatus(entry: LotFilterEntry): LotStatus {
  const { lot, error } = entry;
  if (lot?.status === 'erro' || (!lot && error)) return 'com_erro';
  if (lot?.cessao) return 'cedidos';
  if (lot?.status === 'na_fila' || lot?.status === 'processando') return 'em_analise';
  // Conclusão não implica que haja notas disponíveis para confirmar uma cessão.
  if (lot?.status === 'concluido') return 'analisados';
  return 'desconhecido';
}

/** Um erro de refetch não apaga o último resumo válido; ausência nunca vira zero. */
export function getLotNetValue(entry: LotFilterEntry): number | null {
  const value = entry.lot?.cessao?.liquido_centavos ?? entry.lot?.totais?.liquido_centavos;
  return typeof value === 'number' && Number.isFinite(value) ? value : null;
}

function normalized(value: string): string {
  return value.normalize('NFD').replace(/[\u0300-\u036f]/g, '').toLocaleLowerCase('pt-BR').trim().replace(/\s+/g, ' ');
}
function searchable(entry: LotFilterEntry): string {
  const base = entry.lot?.data_base ?? '';
  const baseBR = /^\d{4}-\d{2}-\d{2}$/.test(base) ? `${base.slice(8, 10)}/${base.slice(5, 7)}/${base.slice(0, 4)}` : '';
  // A data de envio não substitui a data-base desconhecida.
  return normalized([entry.local.nome ?? '', entry.local.arquivo, entry.local.id, base, baseBR].join(' '));
}
function createdAt(entry: LotFilterEntry): number | null {
  const value = Date.parse(entry.local.criado_em);
  return Number.isFinite(value) ? value : null;
}
function numericOrder(left: number | null, right: number | null, direction: 1 | -1): number {
  if (left === null) return right === null ? 0 : 1;
  if (right === null) return -1;
  return (left < right ? -1 : left > right ? 1 : 0) * direction;
}

/** Filtra e ordena sem alterar a coleção recebida; preserva propriedades extras de T. */
export function filterLots<T extends LotFilterEntry>(entries: readonly T[], filters: LotFilters = {}): T[] {
  const busca = normalized(filters.busca ?? '');
  const status = filters.status ?? 'todos';
  const dataBase = filters.dataBase?.trim();
  const order = filters.ordenacao ?? 'recentes';
  return entries.map((entry, index) => ({ entry, index }))
    .filter(({ entry }) => (!busca || searchable(entry).includes(busca))
      && (status === 'todos' || getLotStatus(entry) === status)
      && (!dataBase || entry.lot?.data_base === dataBase))
    .sort((left, right) => {
      const valueOrder = order === 'maior_valor' ? numericOrder(getLotNetValue(left.entry), getLotNetValue(right.entry), -1) : 0;
      const dateOrder = numericOrder(createdAt(left.entry), createdAt(right.entry), order === 'antigos' ? 1 : -1);
      return valueOrder || dateOrder || left.index - right.index;
    })
    .map(({ entry }) => entry);
}
