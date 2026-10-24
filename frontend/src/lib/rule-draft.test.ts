import { describe, expect, it } from 'vitest';
import type { Regras, RegrasPublicadas } from '../api/types';
import { CATALOGO, CATALOGO_PRECO, precoEfetivo, REGRAS_INICIAIS } from '../mocks/fixtures';
import { validateDraft, withRuleParameter } from './rule-draft';

function setup() {
  const draft = structuredClone(REGRAS_INICIAIS);
  const catalog: RegrasPublicadas = {
    versao: 1, publicada_em: '2026-10-24T12:00:00Z', autor: 'semente',
    regras: structuredClone(draft), catalogo: structuredClone(CATALOGO),
    catalogo_precificacao: structuredClone(CATALOGO_PRECO), precificacao_efetiva: precoEfetivo(draft),
  };
  return { draft, catalog };
}
function rule(draft: Regras, code: string) {
  const result = draft.validadores.find(item => item.codigo === code);
  if (!result) throw new Error(`Regra ausente no cenário: ${code}`);
  return result;
}

describe('rascunho conforme parâmetros opcionais do catálogo Go', () => {
  it('permite ativar concentração do cedente sem agrupamento e omitir o agrupamento do sacado', () => {
    const { draft, catalog } = setup();
    const cedente = rule(draft, 'CONCENTRACAO_MAXIMA_CEDENTE');
    cedente.ativo = true;
    expect(Object.hasOwn(cedente.parametros, 'agrupamento')).toBe(false);
    for (const code of ['CONCENTRACAO_MAXIMA_CEDENTE', 'CONCENTRACAO_MAXIMA_SACADO']) {
      expect(catalog.catalogo.find(item => item.codigo === code)?.parametros.find(item => item.nome === 'agrupamento')?.opcional).toBe(true);
      delete rule(draft, code).parametros.agrupamento;
    }
    expect(() => validateDraft(draft, catalog)).not.toThrow();
  });

  it('remove a propriedade ao usar padrão, preservando os outros campos e o objeto original', () => {
    const { draft, catalog } = setup();
    const sacado = rule(draft, 'CONCENTRACAO_MAXIMA_SACADO');
    const original = Object.freeze({ ...sacado.parametros });
    const explicit = withRuleParameter(original, 'agrupamento', 'CNPJ');
    expect(explicit.agrupamento).toBe('CNPJ');
    sacado.parametros = withRuleParameter(explicit, 'agrupamento', undefined);
    expect(sacado.parametros).toEqual({ percentual_maximo: original.percentual_maximo });
    expect(Object.hasOwn(sacado.parametros, 'agrupamento')).toBe(false);
    expect(original.agrupamento).toBe('RAIZ_CNPJ');
    const submitted = JSON.parse(JSON.stringify({ regras: draft })) as { regras: Regras };
    expect(Object.hasOwn(rule(submitted.regras, sacado.codigo).parametros, 'agrupamento')).toBe(false);
    expect(() => validateDraft(submitted.regras, catalog)).not.toThrow();
  });

  it.each(['', 'PADRAO', null])('rejeita agrupamento opcional presente e inválido: %s', value => {
    const { draft, catalog } = setup();
    rule(draft, 'CONCENTRACAO_MAXIMA_SACADO').parametros.agrupamento = value;
    expect(() => validateDraft(draft, catalog)).toThrow('Selecione uma opção');
  });

  it.each([
    ['PRAZO_MAXIMO', 'prazo_maximo_dias', undefined],
    ['PRAZO_MAXIMO', 'prazo_maximo_dias', 1.5],
    ['VALOR_MINIMO', 'valor_minimo', undefined],
    ['TIPO_RECEBIVEL', 'tipos_aceitos', undefined],
    ['CONCENTRACAO_MAXIMA_SACADO', 'percentual_maximo', 101],
  ])('mantém a validação obrigatória de %s.%s', (code, name, value) => {
    const { draft, catalog } = setup();
    rule(draft, code as string).parametros[name as string] = value;
    expect(() => validateDraft(draft, catalog)).toThrow();
  });

  it('exige uma escolha quando o catálogo não marca a opção como opcional', () => {
    const { draft, catalog } = setup();
    const sacado = rule(draft, 'CONCENTRACAO_MAXIMA_SACADO');
    delete sacado.parametros.agrupamento;
    const parameter = catalog.catalogo.find(item => item.codigo === sacado.codigo)!.parametros.find(item => item.nome === 'agrupamento')!;
    delete parameter.opcional;
    expect(() => validateDraft(draft, catalog)).toThrow('Selecione uma opção');
    sacado.parametros.agrupamento = 'CNPJ';
    expect(() => validateDraft(draft, catalog)).not.toThrow();
  });
});
