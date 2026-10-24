import { afterEach,beforeEach,describe,expect,it,vi } from 'vitest';
import { clearGestorSenha,confirmarCessao,filtrosQuery,getGestorSenha,getNotas,notasCSVURL,previaRegras,publicarRegras,registrarLote,lotesLocais,nomeArquivoLote,nomeLote,request,setGestorSenha,uploadLote,verificarGestor } from './client';
import type { Regras } from './types';
class MemoryStorage implements Storage {private values=new Map<string,string>();get length(){return this.values.size;}clear(){this.values.clear();}getItem(key:string){return this.values.get(key)??null;}key(index:number){return [...this.values.keys()][index]??null;}removeItem(key:string){this.values.delete(key);}setItem(key:string,value:string){this.values.set(key,value);}}
const fetchMock=vi.fn();
beforeEach(()=>{vi.stubGlobal('fetch',fetchMock);vi.stubGlobal('sessionStorage',new MemoryStorage());vi.stubGlobal('localStorage',new MemoryStorage());fetchMock.mockReset();});
afterEach(()=>vi.unstubAllGlobals());
describe('cliente fiel ao contrato Go v1.1',()=>{
 it('trata 204 de autenticação e salva a senha só depois de validá-la',async()=>{fetchMock.mockResolvedValue(new Response(null,{status:204}));await verificarGestor('senha-de-teste');expect(getGestorSenha()).toBe('senha-de-teste');expect(fetchMock.mock.calls[0][0]).toBe('/api/gestor/verificar');expect(fetchMock.mock.calls[0][1].headers['X-Gestor-Senha']).toBe('senha-de-teste');clearGestorSenha();expect(getGestorSenha()).toBe('');});
 it('preserva mensagem, código e detalhes de erro do backend',async()=>{fetchMock.mockResolvedValue(Response.json({erro:{codigo:'REGRAS_INVALIDAS',mensagem:'Algumas regras têm valores inválidos.',detalhes:[{campo:'validadores[0]',problema:'inválido'}]}},{status:400}));await expect(request('/api/regras')).rejects.toMatchObject({message:'Algumas regras têm valores inválidos.',status:400,erro:{codigo:'REGRAS_INVALIDAS',detalhes:[{campo:'validadores[0]',problema:'inválido'}]}});});
 it('envia a página e os filtros ao servidor e preserva campos nulos',async()=>{const page={pagina:2,por_pagina:50,total:87,total_paginas:2,itens:[{linha:77,vencimento:null,dias:null,face_centavos:null,status:'recusada'}]};fetchMock.mockResolvedValue(Response.json(page));const result=await getNotas('a/b',{pagina:2,por_pagina:50,status:'recusada',motivo:'FORMATO_INVALIDO',busca:'a & b',ordenar:'face',direcao:'desc'});expect(result).toEqual(page);const url=String(fetchMock.mock.calls[0][0]);expect(url).toContain('/api/lotes/a%2Fb/notas?');const params=new URL(url,'http://localhost').searchParams;expect(params.get('pagina')).toBe('2');expect(params.get('motivo')).toBe('FORMATO_INVALIDO');expect(params.get('busca')).toBe('a & b');});
 it('exporta no servidor com os mesmos filtros, sem acumular páginas',()=>{const url=notasCSVURL('id',{pagina:4,por_pagina:50,status:'recusada',busca:'loja',motivo:'PRAZO_MAXIMO',ordenar:'dias'});expect(url).toContain('/api/lotes/id/notas.csv?');expect(url).not.toContain('pagina');expect(url).not.toContain('ordenar');expect(new URL(url,'http://localhost').searchParams.get('status')).toBe('recusada');expect(filtrosQuery({pagina:1,busca:''})).toBe('pagina=1');});
 it('reutiliza a chave de cessão e envia confirmação sem corpo',async()=>{fetchMock.mockResolvedValue(Response.json({cessao:{id:'cessao1'},notas_excluidas:[]},{status:201}));await confirmarCessao('lote1','chave-estavel');const [,init]=fetchMock.mock.calls[0];expect(init.headers['Idempotency-Key']).toBe('chave-estavel');expect(init.body).toBeUndefined();});
 it('publica e simula com nomes exatos e senha no header',async()=>{setGestorSenha('gestor-teste');fetchMock.mockImplementation(()=>Promise.resolve(Response.json({versao:2})));const regras={fundo:{nome:'ANTECIPE FIDC',cnpj:'92.234.400/0001-80'},validadores:[],precificacao:{metodo:'DESCONTO_SIMPLES',taxa_mensal_percentual:2,base_dias:30}} satisfies Regras;await publicarRegras(regras,'Rafael');expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toEqual({regras,autor:'Rafael'});expect(fetchMock.mock.calls[0][1].method).toBe('PUT');expect(fetchMock.mock.calls[0][1].headers['X-Gestor-Senha']).toBe('gestor-teste');await previaRegras(regras,'lote1');expect(JSON.parse(fetchMock.mock.calls[1][1].body)).toEqual({regras,lote_id:'lote1'});});
 it('mantém o histórico legado sem nome e não grava notas ou CSV',()=>{registrarLote('id1','exemplo.csv');registrarLote('id1','reenvio.csv');expect(lotesLocais()).toHaveLength(1);expect(Object.keys(lotesLocais()[0]).sort()).toEqual(['arquivo','criado_em','id']);expect(nomeLote('id1')).toBe('exemplo.csv');expect(nomeArquivoLote('id1')).toBe('exemplo.csv');expect(nomeLote('ausente')).toBe('Lote de notas');});
 it('salva o nome opcional como metadado com espaços aparados e limite de 80 caracteres',()=>{registrarLote('id-com-nome','origem.csv',`  ${'N'.repeat(90)}  `);const local=lotesLocais()[0];expect(local.nome).toBe('N'.repeat(80));expect(Object.keys(local).sort()).toEqual(['arquivo','criado_em','id','nome']);expect(nomeLote(local.id)).toBe('N'.repeat(80));expect(nomeArquivoLote(local.id)).toBe('origem.csv');registrarLote(local.id,'outra-origem.csv','   ');expect(lotesLocais()[0].nome).toBe(local.nome);});
 it('atualiza o nome em replay sem duplicar o id ou mudar o arquivo e a data originais',()=>{registrarLote('replay1','arquivo-original.csv','Nome anterior');const original=lotesLocais()[0];const historyKey=localStorage.key(0);expect(historyKey).toBeTruthy();localStorage.setItem(historyKey!,JSON.stringify([{...original,criado_em:'2020-01-02T03:04:05.000Z'}]));registrarLote('replay1','arquivo-renomeado.csv','  Carteira de novembro  ');expect(lotesLocais()).toEqual([{id:'replay1',arquivo:'arquivo-original.csv',criado_em:'2020-01-02T03:04:05.000Z',nome:'Carteira de novembro'}]);registrarLote('replay1','terceiro-arquivo.csv');expect(lotesLocais()[0].nome).toBe('Carteira de novembro');});
 it('envia somente arquivo e data-base no multipart, sem nome local ou renomear o CSV',async()=>{
  const sent:FormData[]=[];
  const opened:string[][]=[];
  const headers:string[][]=[];
  class UploadXHR {
   upload={};status=202;response={lote_id:'upload-com-nome',status:'na_fila',replay:false};onload:(()=>void)|null=null;
   open(method:string,url:string){opened.push([method,url]);}
   setRequestHeader(name:string,value:string){headers.push([name,value]);}
   send(form:FormData){sent.push(form);this.onload?.();}
  }
  vi.stubGlobal('XMLHttpRequest',UploadXHR);
  const csv='COLUNA;OUTRA\nCONTEUDO_EXCLUSIVO_DO_CSV;123';
  const file=new File([csv],'arquivo-original.csv',{type:'text/csv'});
  const result=await uploadLote(file,'2026-10-24',vi.fn());
  registrarLote(result.lote_id,file.name,'Nome apenas no navegador');
  expect(opened).toEqual([['POST','/api/lotes']]);
  expect(headers).toEqual([]);
  expect([...sent[0].keys()]).toEqual(['arquivo','data_base']);
  expect(sent[0].get('data_base')).toBe('2026-10-24');
  expect((sent[0].get('arquivo') as File).name).toBe('arquivo-original.csv');
  expect(await (sent[0].get('arquivo') as File).text()).toBe(csv);
  const stored=localStorage.getItem(localStorage.key(0)!)!;
  expect(stored).toContain('Nome apenas no navegador');expect(stored).not.toContain('CONTEUDO_EXCLUSIVO_DO_CSV');
  expect(fetchMock).not.toHaveBeenCalled();
 });
 it('preserva a chave opcional de upload numa nova tentativa sem mudar multipart ou arquivo',async()=>{
  const requests:{headers:Record<string,string>;form:FormData}[]=[];
  class UploadXHR {
   upload={};status=202;response={lote_id:'example-analysis',status:'na_fila',replay:false};onload:(()=>void)|null=null;onerror:(()=>void)|null=null;
   headers:Record<string,string>={};
   open(){}
   setRequestHeader(name:string,value:string){this.headers[name]=value;}
   send(form:FormData){requests.push({headers:{...this.headers},form});if(requests.length===1)this.onerror?.();else this.onload?.();}
  }
  vi.stubGlobal('XMLHttpRequest',UploadXHR);
  const csv='COLUNA;OUTRA\nMESMO_EXEMPLO;123';
  const file=new File([csv],'exemplo-demonstracao.csv',{type:'text/csv'});
  const stableKey='example-attempt-1';
  await expect(uploadLote(file,'2026-10-24',vi.fn(),undefined,stableKey)).rejects.toMatchObject({status:0,erro:{codigo:'SEM_CONEXAO'}});
  await expect(uploadLote(file,'2026-10-24',vi.fn(),undefined,stableKey)).resolves.toMatchObject({lote_id:'example-analysis',replay:false});
  expect(requests).toHaveLength(2);
  for(const request of requests){
   expect(request.headers).toEqual({'Idempotency-Key':stableKey});
   expect([...request.form.keys()]).toEqual(['arquivo','data_base']);
   expect(request.form.get('data_base')).toBe('2026-10-24');
   const sentFile=request.form.get('arquivo') as File;
   expect(sentFile.name).toBe('exemplo-demonstracao.csv');expect(await sentFile.text()).toBe(csv);
  }
 });
 it('apresenta falhas de conexão sem fallback para mock',async()=>{fetchMock.mockRejectedValue(new TypeError('Failed to fetch'));await expect(request('/api/config')).rejects.toMatchObject({erro:{codigo:'SEM_CONEXAO'},status:0});expect(fetchMock).toHaveBeenCalledTimes(1);});
});
