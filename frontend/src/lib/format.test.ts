import { describe,expect,it } from 'vitest';
import { dateBR,moneyCentavos } from './format';
describe('dinheiro do contrato em centavos',()=>{
 it('exibe o exemplo oficial de juros compostos',()=>{expect(moneyCentavos(980392)).toBe('R$\u00a09.803,92');expect(moneyCentavos(19608)).toBe('R$\u00a0196,08');});
 it('exibe ausências de linha inválida sem confundir com zero',()=>{expect(moneyCentavos(null)).toBe('—');expect(moneyCentavos(0)).toBe('R$\u00a00,00');});
 it('mantém centavos em totais grandes e negativos',()=>{expect(moneyCentavos(96419753200)).toBe('R$\u00a0964.197.532,00');expect(moneyCentavos(-19608)).toBe('-R$\u00a0196,08');});
 it('preserva datas civis e apresenta timestamps em São Paulo',()=>{expect(dateBR('2026-10-22')).toBe('22/10/2026');expect(dateBR('2026-10-25T00:30:00Z')).toBe('24/10/2026');});
});
