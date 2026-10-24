import type { ParametroCatalogo, Regra, RegraCatalogo } from '@/api/types';

/** User-facing titles and descriptions come from the backend's catalog. */
export function ruleTitle(code: string, catalog: RegraCatalogo[] = []): string {
  return catalog.find((item) => item.codigo === code)?.titulo ?? code;
}

export function ruleDescription(rule: Regra, catalog: RegraCatalogo[] = []): string {
  return catalog.find((item) => item.codigo === rule.codigo)?.descricao ?? '';
}

export function parameterValue(value: unknown, parameter?: ParametroCatalogo): string {
  if (value === undefined || value === null || value === '') return 'Não configurado';
  const text = Array.isArray(value)
    ? value.map(String).join(', ')
    : typeof value === 'number' ? value.toLocaleString('pt-BR', { maximumFractionDigits: 12 }) : String(value);
  return parameter?.unidade ? `${text} ${parameter.unidade}` : text;
}
