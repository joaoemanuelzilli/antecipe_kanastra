import Decimal from 'decimal.js';
const FormatDecimal=Decimal.clone({precision:50,rounding:Decimal.ROUND_HALF_UP});
export function money(value: string | number) { const decimal=new FormatDecimal(String(value));const [integer,cents]=decimal.abs().toFixed(2).split('.');return `${decimal.isNegative()?'-':''}R$\u00a0${integer.replace(/\B(?=(\d{3})+(?!\d))/g,'.')},${cents}`; }
export function dateBR(value: string) { if(value.includes('T'))return new Intl.DateTimeFormat('pt-BR',{timeZone:'America/Sao_Paulo'}).format(new Date(value));const date = value.slice(0,10).split('-'); return date.length === 3 ? `${date[2]}/${date[1]}/${date[0]}` : value; }
export function todayISO() { return new Intl.DateTimeFormat('en-CA', {timeZone:'America/Sao_Paulo'}).format(new Date()); }
export function cx(...items: (string | boolean | undefined)[]) { return items.filter(Boolean).join(' '); }
export function moneyCentavos(value:number|null|undefined){if(value===null||value===undefined)return '—';return money(new FormatDecimal(String(value)).dividedBy(100).toFixed(2));}
export function countBR(value:number){return new Intl.NumberFormat('pt-BR').format(value);}
