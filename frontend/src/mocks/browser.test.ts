import { createServer } from 'node:http';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { http, HttpResponse } from 'msw';
interface RuntimeForTest {interceptor:{dispose:()=>void};handlers:ReturnType<typeof http.get>[];cleanup?:Promise<void>}
const page=globalThis as typeof globalThis & {__antecipeMockTransportV2?:RuntimeForTest};
beforeEach(()=>{delete page.__antecipeMockTransportV2;vi.resetModules();});
afterEach(()=>{page.__antecipeMockTransportV2?.interceptor.dispose();delete page.__antecipeMockTransportV2;vi.unstubAllGlobals();vi.restoreAllMocks();vi.useRealTimers();});
const origin='http://demo.test';
describe('transporte explícito da demonstração na página',()=>{
  it('ativa Fetch+XHR uma única vez para chamadas concorrentes, sem registrar SW',async()=>{
    const register=vi.fn(),getRegistrations=vi.fn().mockResolvedValue([]);
    vi.stubGlobal('navigator',{serviceWorker:{register,getRegistrations}});vi.stubGlobal('location',{origin});
    const {interceptor,startMockWorker}=await import('./browser');const apply=vi.spyOn(interceptor,'apply');
    const first=startMockWorker(),second=startMockWorker();expect(first).toBe(second);await Promise.all([first,second]);
    expect(apply).toHaveBeenCalledTimes(1);expect(register).not.toHaveBeenCalled();
    const response=await fetch(`${origin}/api/config`);expect(response.status).toBe(200);expect((await response.json()).precificacao.metodo).toBe('DESCONTO_COMPOSTO');
  });
  it('responde mesmo após tempo ocioso e sem disponibilidade da API de Service Worker',async()=>{
    vi.stubGlobal('navigator',{});
    const {startMockWorker}=await import('./browser');await startMockWorker();
    const first=await fetch(`${origin}/api/config`);expect(first.status).toBe(200);
    vi.useFakeTimers({toFake:['Date']});vi.setSystemTime(Date.now()+4*60*1000);
    const afterIdle=await fetch(`${origin}/api/config`);expect(afterIdle.status).toBe(200);expect((await afterIdle.json()).regras_versao).toBe(1);
  });
  it('preserva o mesmo interceptor e a ativação quando o módulo é reavaliado por HMR',async()=>{
    const before=await import('./browser');const apply=vi.spyOn(before.interceptor,'apply');await before.startMockWorker();
    const patchedFetch=globalThis.fetch;vi.resetModules();
    const after=await import('./browser');await after.startMockWorker();
    expect(after.interceptor).toBe(before.interceptor);expect(globalThis.fetch).toBe(patchedFetch);expect(apply).toHaveBeenCalledTimes(1);
    expect((await fetch(`${origin}/api/saude`)).status).toBe(200);
  });
  it('preserva rede real para passthrough MSW e ausência de handler, mantendo API desconhecida501',async()=>{
    const source='export const client="javascript-real";',received:string[]=[];
    const realServer=createServer((request,response)=>{received.push(request.url??'');response.writeHead(200,{'Content-Type':'application/javascript'});response.end(source);});
    await new Promise<void>(resolve=>realServer.listen(0,'127.0.0.1',resolve));
    try{
      const {startMockWorker}=await import('./browser');await startMockWorker();
      const address=realServer.address();if(!address || typeof address==='string')throw new Error('Porta temporária indisponível.');
      const base=`http://127.0.0.1:${address.port}`;
      // O catch-all retorna passthrough(), que não pode ser enviado como302 ao cliente.
      const module=await fetch(`${base}/src/api/client.ts`);expect(module.status).toBe(200);expect(module.headers.get('Content-Type')).toBe('application/javascript');expect(await module.text()).toBe(source);
      const unknown=await fetch(`${base}/api/legada`);expect(unknown.status).toBe(501);expect((await unknown.json()).erro.codigo).toBe('ROTA_NAO_PREVISTA_NO_CONTRATO');
      // Simula uma lista estreita: getResponse retorna undefined para o asset.
      page.__antecipeMockTransportV2!.handlers=[http.get('*/api/exclusiva',()=>HttpResponse.json({ok:true}))];
      const asset=await fetch(`${base}/asset.js`);expect(asset.status).toBe(200);expect(await asset.text()).toBe(source);
      expect(received).toEqual(['/src/api/client.ts','/asset.js']);
    }finally{await new Promise<void>((resolve,reject)=>realServer.close(error=>error?reject(error):resolve()));}
  });
  it('faz upload multipart pelo transporte em página e preserva replay do contrato',async()=>{
    const {startMockWorker}=await import('./browser');const {gerarCSVExemplo}=await import('./fixtures');await startMockWorker();
    const send=()=>{const form=new FormData();form.set('arquivo',new File([gerarCSVExemplo()],'demo.csv',{type:'text/csv'}));form.set('data_base','2026-10-24');return fetch(`${origin}/api/lotes`,{method:'POST',body:form});};
    const accepted=await send();expect(accepted.status).toBe(202);const body=await accepted.json();expect(accepted.headers.get('Location')).toBe(`/api/lotes/${body.lote_id}`);
    const replay=await send();expect(replay.status).toBe(200);expect(replay.headers.get('Idempotent-Replay')).toBe('true');expect((await replay.json()).lote_id).toBe(body.lote_id);
  });
  it('remove apenas a inscrição antiga do próprio origin e não aguarda SW para iniciar a UI',async()=>{
    let finishCleanup!:(value:unknown[])=>void;
    const getRegistrations=vi.fn(()=>new Promise<unknown[]>(resolve=>{finishCleanup=resolve;}));
    vi.stubGlobal('navigator',{serviceWorker:{getRegistrations}});vi.stubGlobal('location',{origin});
    const {startMockWorker}=await import('./browser');await startMockWorker();
    // O cleanup ainda está pendente, mas os mocks já respondem sem consultar a rede real.
    expect((await fetch(`${origin}/api/config`)).status).toBe(200);
    const ours=vi.fn().mockResolvedValue(true),foreign=vi.fn(),other=vi.fn(),mixed=vi.fn();
    finishCleanup([
      {active:{scriptURL:`${origin}/mockServiceWorker.js`},waiting:null,installing:null,unregister:ours},
      {active:{scriptURL:'https://outro.test/mockServiceWorker.js'},waiting:null,installing:null,unregister:foreign},
      {active:{scriptURL:`${origin}/offline.js`},waiting:null,installing:null,unregister:other},
      {active:{scriptURL:`${origin}/mockServiceWorker.js`},waiting:{scriptURL:`${origin}/offline.js`},installing:null,unregister:mixed},
    ]);
    await page.__antecipeMockTransportV2!.cleanup;
    expect(ours).toHaveBeenCalledTimes(1);expect(foreign).not.toHaveBeenCalled();expect(other).not.toHaveBeenCalled();expect(mixed).not.toHaveBeenCalled();
  });
  it('propaga falha de ativação sem renderização com transporte real como fallback',async()=>{
    const {interceptor,startMockWorker}=await import('./browser');const failure=new Error('Intercepção indisponível');const apply=vi.spyOn(interceptor,'apply').mockImplementation(()=>{throw failure;});
    await expect(startMockWorker()).rejects.toBe(failure);await expect(startMockWorker()).rejects.toBe(failure);expect(apply).toHaveBeenCalledTimes(1);
  });
});
