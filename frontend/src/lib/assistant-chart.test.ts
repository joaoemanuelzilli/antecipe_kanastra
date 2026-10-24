import { describe, expect, it } from 'vitest';
import { parseAssistantChart } from './assistant-chart';
describe('gráficos usam referências a dados da API', () => {
  it('aceita gráfico contextual e UUID explícito', () => {
    expect(parseAssistantChart('{"tipo":"rosca","titulo":"Notas","metrica":"status"}')?.metrica).toBe('status');
    expect(parseAssistantChart('{"tipo":"barras","titulo":"Notas","metrica":"motivos","lote_id":"8f1c2a9e-0d4b-4c51-9a77-2f3e6b1d0c42"}')?.lote_id).toBeTruthy();
  });
  it('rejeita números arbitrários do modelo e configuração executável', () => {
    expect(parseAssistantChart('{"tipo":"barras","titulo":"Notas","metrica":"status","valores":[19,11]}')).toBeNull();
    expect(parseAssistantChart('{"tipo":"javascript","titulo":"Notas","metrica":"status"}')).toBeNull();
  });
  it('rejeita rosca de valores que não são partes independentes e JSON incompleto', () => {
    expect(parseAssistantChart('{"tipo":"rosca","titulo":"Valores","metrica":"valores"}')).toBeNull();
    expect(parseAssistantChart('{"tipo":')).toBeNull();
    expect(parseAssistantChart('x'.repeat(2001))).toBeNull();
  });
});
