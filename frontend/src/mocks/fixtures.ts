/** Fixtures educativas do contrato Go v1.1. Nenhum dado ou motor do projeto Next é importado. */
import Decimal from 'decimal.js';
import type { Config, Precificacao, RegraCatalogo, Regras } from '../api/types';
import regrasOriginais from './regras.json';
export const DEMO_PASSWORD = 'demo-gestor';
export const COLUNAS = ['NOME_FUNDO','DOC_FUNDO','NOME_CEDENTE','DOC_CEDENTE','NOME_SACADO','DOC_SACADO','NU_DOCUMENTO','TIPO_RECEBIVEL','VALOR_NOMINAL','DATA_VENCIMENTO_ORIGINAL','DATA_AQUISICAO'];
export const REGRAS_INICIAIS = regrasOriginais as Regras;
export const CATALOGO: RegraCatalogo[] = [
  {codigo:'CNPJ_SACADO_VALIDO',titulo:'CNPJ do cliente',descricao:'Verifica os dígitos do documento do sacado.',parametros:[]},
  {codigo:'SITUACAO_CADASTRAL_SACADO',titulo:'Situação cadastral',descricao:'Situações do cliente aceitas pelo fundo. Nesta demonstração, os cadastros são simulados.',parametros:[{nome:'situacoes_aceitas',rotulo:'Situações aceitas',tipo:'lista_texto',opcoes:['ATIVA','INAPTA','BAIXADA']}]},
  {codigo:'TIPO_RECEBIVEL',titulo:'Tipos de nota aceitos',descricao:'Só os tipos marcados podem ser comprados.',parametros:[{nome:'tipos_aceitos',rotulo:'Tipos aceitos',tipo:'lista_texto',opcoes:['Duplicata','Duplicata de Serviço','Cheque','CCB','Nota Comercial']}]},
  {codigo:'VALOR_MINIMO',titulo:'Valor mínimo',descricao:'Menor valor de face aceito por nota.',parametros:[{nome:'valor_minimo',rotulo:'Valor mínimo',tipo:'decimal',unidade:'R$',minimo:0}]},
  {codigo:'NAO_VENCIDA',titulo:'Vencimento futuro',descricao:'A nota deve vencer depois da data-base.',parametros:[]},
  {codigo:'PRAZO_MAXIMO',titulo:'Prazo máximo',descricao:'Maior número de dias entre a data-base e o vencimento.',parametros:[{nome:'prazo_maximo_dias',rotulo:'Prazo máximo',tipo:'inteiro',unidade:'dias',minimo:1}]},
  {codigo:'CONCENTRACAO_MAXIMA_SACADO',titulo:'Concentração máxima por cliente (sacado)',descricao:'Parte máxima da carteira que um mesmo cliente pode ter.',parametros:[{nome:'percentual_maximo',rotulo:'Limite',tipo:'decimal',unidade:'%',minimo:0,maximo:100},{nome:'agrupamento',rotulo:'Agrupar por',tipo:'opcao',opcoes:['RAIZ_CNPJ','CNPJ'],opcional:true}]},
  {codigo:'PRAZO_MINIMO',titulo:'Prazo mínimo',descricao:'Menor número de dias entre a data-base e o vencimento.',parametros:[{nome:'prazo_minimo_dias',rotulo:'Prazo mínimo',tipo:'inteiro',unidade:'dias',minimo:1}]},
  {codigo:'CONCENTRACAO_MAXIMA_CEDENTE',titulo:'Concentração máxima por empresa (cedente)',descricao:'Parte máxima da carteira que um mesmo cedente pode ter.',parametros:[{nome:'percentual_maximo',rotulo:'Limite',tipo:'decimal',unidade:'%',minimo:0,maximo:100},{nome:'agrupamento',rotulo:'Agrupar por',tipo:'opcao',opcoes:['RAIZ_CNPJ','CNPJ'],opcional:true}]},
];
export const CATALOGO_PRECO = {parametros:[{nome:'taxa_mensal_percentual',rotulo:'Taxa',tipo:'decimal' as const,unidade:'% ao mês',minimo:0},{nome:'base_dias',rotulo:'Base',tipo:'inteiro' as const,unidade:'dias',minimo:1}]};
export function precoEfetivo(regras:Regras):Precificacao {return {metodo:'DESCONTO_COMPOSTO',taxa_mensal_percentual:regras.precificacao.taxa_mensal_percentual,base_dias:regras.precificacao.base_dias,descricao:`Juros compostos · ${String(regras.precificacao.taxa_mensal_percentual).replace('.',',')}% ao mês`,fonte_metodo:'config/antecipe.yaml',sobrescritos_pelo_yaml:['metodo']};}
export function configDemo(regras:Regras,versao:number):Config {return {fundo:regras.fundo,data_base_padrao:'2026-10-24',regras_versao:versao,precificacao:precoEfetivo(regras),upload:{tamanho_maximo_bytes:314572800,separador:';',colunas:COLUNAS}};}
export function dataBR(iso:string):string {return `${iso.slice(8,10)}/${iso.slice(5,7)}/${iso.slice(0,4)}`;}
export function adicionarDias(base:string,dias:number):string {const data=new Date(`${base}T00:00:00Z`);data.setUTCDate(data.getUTCDate()+dias);return data.toISOString().slice(0,10);}
export function moedaCentavos(valor:number):string {return new Intl.NumberFormat('pt-BR',{style:'currency',currency:'BRL'}).format(valor/100);}
export function csvCell(valor:unknown):string {const texto=String(valor??'');return `"${(/^[\s]*[=+\-@]/.test(texto)?`'${texto}`:texto).replaceAll('"','""')}"`;}
export function gerarCSVExemplo(dataBase='2026-10-24'):string {
  const nomes=['Loja Aurora','Tecidos Ipê','Comercial Brisa','Móveis Horizonte','Embalagens Cedro','Papelaria Girassol'];
  const linhas=Array.from({length:30},(_,i)=>({NOME_FUNDO:'ANTECIPE FIDC',DOC_FUNDO:'92.234.400/0001-80',NOME_CEDENTE:'Estofados Claro — empresa fictícia',DOC_CEDENTE:'11.222.333/0001-81',NOME_SACADO:`${nomes[i%nomes.length]} — empresa fictícia`,DOC_SACADO:'11.444.777/0001-61',NU_DOCUMENTO:`DEMO-${String(i+1).padStart(4,'0')}`,TIPO_RECEBIVEL:i%3===0?'Duplicata de Serviço':'Duplicata',VALOR_NOMINAL:new Decimal(i===0?'10000':String(1500+i*420)).toFixed(2).replace('.',','),DATA_VENCIMENTO_ORIGINAL:dataBR(adicionarDias(dataBase,i===0?30:35+(i*7)%145)),DATA_AQUISICAO:''}));
  linhas[16].NU_DOCUMENTO='DEMO-PARALELA';
  linhas[17].DOC_SACADO='12.345.678/0001-00';
  linhas[18].NOME_SACADO='Cadastro INAPTA — empresa fictícia';
  linhas[19].TIPO_RECEBIVEL='Cheque';
  linhas[20].VALOR_NOMINAL='65,50';
  linhas[21].DATA_VENCIMENTO_ORIGINAL=dataBR(adicionarDias(dataBase,-4));
  linhas[22].DATA_VENCIMENTO_ORIGINAL=dataBR(adicionarDias(dataBase,240));
  linhas[23].NOME_SACADO='Limite Sacado — empresa fictícia';
  linhas[24].NU_DOCUMENTO='EXISTENTE-MOCK';
  linhas[25]={...linhas[0]};
  linhas[26].NU_DOCUMENTO='DEMO-FORMATO';linhas[26].DATA_VENCIMENTO_ORIGINAL='31/02/2026';linhas[26].VALOR_NOMINAL='ilegível';
  linhas[27].DATA_VENCIMENTO_ORIGINAL=dataBR(adicionarDias(dataBase,7));
  linhas[28].NOME_CEDENTE='Limite Cedente — empresa fictícia';
  linhas[29].TIPO_RECEBIVEL='Cheque';linhas[29].VALOR_NOMINAL='35,00';linhas[29].DATA_VENCIMENTO_ORIGINAL=dataBR(adicionarDias(dataBase,-2));
  return '\uFEFF'+[COLUNAS.join(';'),...linhas.map(row=>COLUNAS.map(col=>csvCell(row[col as keyof typeof row])).join(';'))].join('\r\n')+'\r\n';
}
