import type { ParametroCatalogo, Regras, RegrasPublicadas } from '../api/types';

/** Ausência usa o padrão do backend; a propriedade não deve ser enviada vazia. */
export function withRuleParameter(parameters: Record<string, unknown>, name: string, value: unknown): Record<string, unknown> {
  const next = { ...parameters };
  if (value === undefined) delete next[name];
  else next[name] = value;
  return next;
}

export function validateDraft(draft: Regras, catalog: RegrasPublicadas): void {
  function validate(parameter: ParametroCatalogo, value: unknown): void {
    if (parameter.opcional && value === undefined) return;
    if (parameter.tipo === 'inteiro' || parameter.tipo === 'decimal') {
      if (typeof value !== 'number' || !Number.isFinite(value) || parameter.tipo === 'inteiro' && !Number.isInteger(value)) throw new Error(`Confira o campo “${parameter.rotulo}”. Informe ${parameter.tipo === 'inteiro' ? 'um número inteiro' : 'um número válido'}.`);
      if (parameter.minimo !== undefined && value < parameter.minimo) throw new Error(`O campo “${parameter.rotulo}” deve ser maior ou igual a ${parameter.minimo.toLocaleString('pt-BR')}.`);
      if (parameter.maximo !== undefined && value > parameter.maximo) throw new Error(`O campo “${parameter.rotulo}” deve ser menor ou igual a ${parameter.maximo.toLocaleString('pt-BR')}.`);
    }
    if (parameter.tipo === 'opcao' && (!Array.isArray(parameter.opcoes) || !parameter.opcoes.includes(String(value)))) throw new Error(`Selecione uma opção para “${parameter.rotulo}”.`);
    if (parameter.tipo === 'lista_texto' && (!Array.isArray(value) || value.some((item) => typeof item !== 'string' || !item.trim()))) throw new Error(`Confira os valores de “${parameter.rotulo}”.`);
  }
  for (const rule of draft.validadores.filter((item) => item.ativo)) for (const parameter of catalog.catalogo.find((item) => item.codigo === rule.codigo)?.parametros ?? []) validate(parameter, rule.parametros[parameter.nome]);
  for (const parameter of catalog.catalogo_precificacao.parametros) if (!catalog.precificacao_efetiva.sobrescritos_pelo_yaml?.includes(parameter.nome)) validate(parameter, draft.precificacao[parameter.nome as keyof Regras['precificacao']]);
}
