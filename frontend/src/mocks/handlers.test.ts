import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { z } from 'zod';
import { createServer } from 'node:http';
import { server } from './server';
import { advanceMockTime, DEMO_PROCESSING_MS, DEMO_QUEUE_MS, DEMO_STAGE_MS, DEMO_TOTAL_MS, resetMockState } from './handlers';
import { COLUNAS, DEMO_PASSWORD, gerarCSVExemplo, REGRAS_INICIAIS } from './fixtures';
import type { Lote, PaginaNotas, Previa, Regras, RegrasPublicadas, ResultadoCessao, UploadResposta } from '../api/types';
const origin='http://localhost:5173';
const auth={'X-Gestor-Senha':DEMO_PASSWORD,'Content-Type':'application/json'};
const json=(method:string,body:unknown,headers:HeadersInit=auth):RequestInit=>({method,headers,body:JSON.stringify(body)});
async function upload(csv=gerarCSVExemplo(),key?:string){const form=new FormData();form.set('arquivo',new File([csv],'fixture-sintetica.csv',{type:'text/csv'}));form.set('data_base','2026-10-24');return fetch(`${origin}/api/lotes`,{method:'POST',body:form,headers:key?{'Idempotency-Key':key}:undefined});}
async function completed(){const response=await upload();const body=await response.json() as UploadResposta;advanceMockTime(DEMO_TOTAL_MS+1);return body.lote_id;}
const getLot=(id:string)=>fetch(`${origin}/api/lotes/${id}`).then(r=>r.json()) as Promise<Lote>;
beforeAll(()=>server.listen({onUnhandledFrame:'error'}));
afterAll(()=>server.close());
beforeEach(()=>{vi.useFakeTimers({toFake:['Date']});vi.setSystemTime(new Date('2026-10-24T12:00:00Z'));resetMockState();server.resetHandlers();});
afterEach(()=>vi.useRealTimers());
describe('MSW contrato Go v1.1 — transporte e processamento',()=>{
  it('responde saúde/config e conserva precedência YAML sobre JSON original',async()=>{
    expect(await (await fetch(`${origin}/api/saude`)).json()).toEqual({status:'ok'});
    const config=await (await fetch(`${origin}/api/config`)).json();expect(config.precificacao.metodo).toBe('DESCONTO_COMPOSTO');expect(config.precificacao.descricao).toBe('Juros compostos · 2% ao mês');expect(config.upload.colunas).toEqual(COLUNAS);expect(config.upload.tamanho_maximo_bytes).toBe(314572800);
    const rules=await (await fetch(`${origin}/api/regras`)).json() as RegrasPublicadas;expect(rules.regras.precificacao.metodo).toBe('DESCONTO_SIMPLES');expect(rules.catalogo).toHaveLength(9);expect(rules.precificacao_efetiva.sobrescritos_pelo_yaml).toEqual(['metodo']);expect(rules.catalogo.find(c=>c.codigo==='CONCENTRACAO_MAXIMA_SACADO')?.parametros.find(p=>p.nome==='agrupamento')?.opcoes).toEqual(['RAIZ_CNPJ','CNPJ']);
  });
  it('upload multipart202, Location, cinco etapas, conclusão e replay200',async()=>{
    const response=await upload();expect(response.status).toBe(202);const accepted=await response.json() as UploadResposta;expect(response.headers.get('Location')).toBe(`/api/lotes/${accepted.lote_id}`);expect(accepted.status).toBe('na_fila');
    const queue=await getLot(accepted.lote_id);expect(queue.posicao_fila).toBe(1);expect(queue.totais).toBeNull();
    advanceMockTime(DEMO_QUEUE_MS);const steps:string[]=[];for(let i=0;i<5;i++){const lot=await getLot(accepted.lote_id);expect(lot.status).toBe('processando');steps.push(lot.etapa!);expect(lot.progresso.percentual).toBe(i*20);expect(lot.progresso.linhas_processadas).toBe(i*6);expect(lot.etapas.filter(e=>e.status==='em_andamento')).toHaveLength(1);if(i<4)advanceMockTime(DEMO_STAGE_MS);}
    expect(steps).toEqual(['lendo','validando','concentracao','precificando','salvando']);advanceMockTime(DEMO_STAGE_MS);
    const result=await getLot(accepted.lote_id);expect(result.status).toBe('concluido');expect(result.progresso.percentual).toBe(100);expect(result.etapas.every(e=>e.status==='concluida')).toBe(true);expect(result.totais?.linhas).toBe(30);expect(result.totais?.aprovadas).toBe(19);expect(result.totais?.recusadas).toBe(11);expect(result.motivos.reduce((s,m)=>s+m.quantidade,0)).toBe(result.totais?.recusadas);
    const replay=await upload();expect(replay.status).toBe(200);expect(replay.headers.get('Idempotent-Replay')).toBe('true');expect((await replay.json()).lote_id).toBe(accepted.lote_id);
  });
  it('mantém fila, progresso e timestamps coerentes nos limites dos 9,5 segundos',async()=>{
    const accepted=await (await upload()).json() as UploadResposta;
    const initial=await getLot(accepted.lote_id);
    advanceMockTime(DEMO_QUEUE_MS-1);
    const queued=await getLot(accepted.lote_id);expect(queued.status).toBe('na_fila');expect(queued.iniciado_em).toBeNull();expect(queued.progresso).toMatchObject({percentual:0,linhas_processadas:0});
    advanceMockTime(1);
    const started=await getLot(accepted.lote_id);expect(started.status).toBe('processando');expect(started.etapa).toBe('lendo');expect(started.progresso.percentual).toBe(0);expect(started.totais).toBeNull();
    expect(Date.parse(started.iniciado_em!)-Date.parse(initial.criado_em)).toBe(DEMO_QUEUE_MS);
    // Polling repetido não acelera a simulação; só o relógio avança as etapas.
    const polled=await getLot(accepted.lote_id);expect(polled.progresso).toEqual(started.progresso);expect(polled.iniciado_em).toBe(started.iniciado_em);
    advanceMockTime(DEMO_PROCESSING_MS-1);
    const last=await getLot(accepted.lote_id);expect(last.status).toBe('processando');expect(last.etapa).toBe('salvando');expect(last.progresso.percentual).toBeLessThan(100);expect(last.progresso.linhas_processadas).toBe(29);expect(last.concluido_em).toBeNull();expect(last.duracao_ms).toBeNull();
    advanceMockTime(1);
    const done=await getLot(accepted.lote_id);expect(done.status).toBe('concluido');expect(done.progresso).toMatchObject({percentual:100,linhas_processadas:30});
    expect(Date.parse(done.concluido_em!)-Date.parse(done.criado_em)).toBe(DEMO_TOTAL_MS);
    expect(Date.parse(done.concluido_em!)-Date.parse(done.iniciado_em!)).toBe(DEMO_PROCESSING_MS);expect(done.duracao_ms).toBe(DEMO_PROCESSING_MS);
    expect(done.criado_em).toBe(initial.criado_em);
  });
  it('retorna erros aninhados de cabeçalho, arquivo ausente e extensão',async()=>{
    const bad=await upload('COLUNA_ERRADA\nabc');expect(bad.status).toBe(422);const body=await bad.json();expect(body.erro.codigo).toBe('CABECALHO_INVALIDO');expect(body.erro.detalhes).toContainEqual({linha:1,campo:'VALOR_NOMINAL',problema:'coluna não encontrada'});
    expect((await fetch(`${origin}/api/lotes`,{method:'POST',body:new FormData()})).status).toBe(400);
    const form=new FormData();form.set('arquivo',new File(['abc'],'notas.xlsx'));expect((await fetch(`${origin}/api/lotes`,{method:'POST',body:form})).status).toBe(415);
    expect((await upload(COLUNAS.join(';'))).status).toBe(422);
  });
  it('preserva módulos Vite dentro de /src/api e mantém501 somente no namespace /api',async()=>{
    const source='export const client = "modulo-real-do-vite";';
    let received=0;
    const realServer=createServer((_request,response)=>{received++;response.writeHead(200,{'Content-Type':'application/javascript'});response.end(source);});
    await new Promise<void>(resolve=>realServer.listen(0,'127.0.0.1',resolve));
    try{
      const address=realServer.address();
      if(!address || typeof address==='string')throw new Error('O servidor temporário não abriu uma porta TCP.');
      for(const pathname of ['/src/api/client.ts','/src/api']){
        const response=await fetch(`http://127.0.0.1:${address.port}${pathname}`);
        expect(response.status).toBe(200);
        expect(response.headers.get('Content-Type')).toBe('application/javascript');
        expect(await response.text()).toBe(source);
      }
      expect(received).toBe(2);
      const legacy=await fetch(`${origin}/api/regras/sugerir`,json('POST',{texto:'legado'}));
      expect(legacy.status).toBe(501);
      expect((await legacy.json()).erro.codigo).toBe('ROTA_NAO_PREVISTA_NO_CONTRATO');
      expect((await fetch(`${origin}/api`)).status).toBe(501);
    }finally{
      await new Promise<void>((resolve,reject)=>realServer.close(error=>error?reject(error):resolve()));
    }
  });
  it('não confunde nomes herdados com lotes ou chaves de idempotência',async()=>{
    for(const id of ['toString','constructor','__proto__'])expect((await fetch(`${origin}/api/lotes/${id}`)).status).toBe(404);
    const response=await upload(undefined,'__proto__');expect(response.status).toBe(202);expect((await upload(undefined,'__proto__')).status).toBe(200);
    expect((await fetch(`${origin}/api`)).status).toBe(501);
  });
  it('mesma chave com conteúdo diferente dá409; id inexistente dá404; rota legada não atravessa rede',async()=>{
    await upload(undefined,'fixed');expect((await upload(gerarCSVExemplo().replace('DEMO-0002','OUTRA-NOTA'),'fixed')).status).toBe(409);expect((await fetch(`${origin}/api/lotes/inexistente`)).status).toBe(404);const unknown=await fetch(`${origin}/api/regras/sugerir`,json('POST',{texto:'legado'}));expect(unknown.status).toBe(501);expect((await unknown.json()).erro.codigo).toBe('ROTA_NAO_PREVISTA_NO_CONTRATO');
  });
});
describe('tabela, centavos e cessão',()=>{
  it('busca páginas/filtros/ordem no servidor e preço composto correto sem todos itens',async()=>{
    const id=await completed();const page=await (await fetch(`${origin}/api/lotes/${id}/notas?pagina=1&por_pagina=5`)).json() as PaginaNotas;expect(page.itens).toHaveLength(5);expect(page.total).toBe(30);expect(page.total_paginas).toBe(6);expect(page.itens[0]).toMatchObject({face_centavos:1000000,desconto_centavos:19608,liquido_centavos:980392});expect(page.itens[0].calculo?.metodo).toBe('DESCONTO_COMPOSTO');
    const filtered=await (await fetch(`${origin}/api/lotes/${id}/notas?status=recusada&motivo=PRAZO_MAXIMO&busca=DEMO-0023`)).json() as PaginaNotas;expect(filtered.total).toBe(1);expect(filtered.itens[0].recusas[0].codigo).toBe('PRAZO_MAXIMO');
    const ordered=await (await fetch(`${origin}/api/lotes/${id}/notas?status=aprovada&ordenar=face&direcao=desc`)).json() as PaginaNotas;expect(ordered.itens[0].face_centavos).toBeGreaterThanOrEqual(ordered.itens[1].face_centavos!);
    expect((await fetch(`${origin}/api/lotes/${id}/notas?por_pagina=201`)).status).toBe(400);
  });
  it('FORMATO_INVALIDO não derruba lote; campos ilegíveis null e todas recusas preservadas',async()=>{
    const id=await completed();const page=await (await fetch(`${origin}/api/lotes/${id}/notas`)).json() as PaginaNotas;const invalid=page.itens.find(n=>n.numero==='DEMO-FORMATO')!;expect(invalid).toMatchObject({status:'recusada',vencimento:null,dias:null,face_centavos:null,desconto_centavos:null,liquido_centavos:null,calculo:null});expect(invalid.recusas[0]).toMatchObject({codigo:'FORMATO_INVALIDO',titulo:'Linha com erro'});expect(page.itens[29].recusas.length).toBeGreaterThan(1);
    const recusaSchema=z.object({codigo:z.string(),titulo:z.string().min(1),mensagem:z.string().min(1),acao:z.string().min(1),dados:z.record(z.string(),z.unknown())});for(const n of page.itens)for(const r of n.recusas)expect(recusaSchema.safeParse(r).success).toBe(true);
    const csv=await fetch(`${origin}/api/lotes/${id}/notas.csv?status=recusada&motivo=FORMATO_INVALIDO`);expect(csv.headers.get('Content-Type')).toContain('text/csv');expect(csv.headers.get('Content-Disposition')).toContain('attachment');const text=await csv.text();expect(text).toContain('O_QUE_FAZER');expect(text).toContain('DEMO-FORMATO');expect(text).not.toContain('DEMO-0002');
  });
  it('cessão exige chave, revalida exclusões e é201/200 idempotente sem duplicar',async()=>{
    const pending=await (await upload()).json() as UploadResposta;const url=`${origin}/api/lotes/${pending.lote_id}/cessao`;expect((await fetch(url,{method:'POST',headers:{'Idempotency-Key':'confirm'}})).status).toBe(409);advanceMockTime(DEMO_TOTAL_MS);
    expect((await fetch(url,{method:'POST'})).status).toBe(400);const response=await fetch(url,{method:'POST',headers:{'Idempotency-Key':'confirm'}});expect(response.status).toBe(201);const result=await response.json() as ResultadoCessao;expect(result.notas_excluidas).toHaveLength(1);expect(result.cessao.notas_cedidas).toBe(18);expect(result.cessao.face_centavos-result.cessao.desconto_centavos).toBe(result.cessao.liquido_centavos);
    const replay=await fetch(url,{method:'POST',headers:{'Idempotency-Key':'confirm'}});expect(replay.status).toBe(200);expect(replay.headers.get('Idempotent-Replay')).toBe('true');expect(await replay.json()).toEqual(result);expect((await getLot(pending.lote_id)).cessao).toEqual(result.cessao);
  });
});
describe('gestor, prévia e versões do contrato',()=>{
  it('verifica senha por header204 e erro401, sem endpoint de sessão legado',async()=>{expect((await fetch(`${origin}/api/gestor/verificar`)).status).toBe(401);const response=await fetch(`${origin}/api/gestor/verificar`,{headers:auth});expect(response.status).toBe(204);expect(await response.text()).toBe('');expect((await fetch(`${origin}/api/regras`,json('PUT',{regras:REGRAS_INICIAIS,autor:'Teste'},{}))).status).toBe(401);});
  it('prévia usa regras/lote_id, não publica; PUT cria versão e próximos uploads usam nova regra',async()=>{
    expect((await fetch(`${origin}/api/regras/previa`,json('POST',{regras:REGRAS_INICIAIS}))).status).toBe(409);const id=await completed();const proposed=structuredClone(REGRAS_INICIAIS);proposed.validadores.find(r=>r.codigo==='PRAZO_MAXIMO')!.parametros.prazo_maximo_dias=120;
    const preview=await (await fetch(`${origin}/api/regras/previa`,json('POST',{regras:proposed,lote_id:id}))).json() as Previa;expect(preview.lote_id).toBe(id);expect(preview.depois.aprovadas).toBeLessThan(preview.antes.aprovadas);expect(preview.mudancas[0]).toContain('PRAZO_MAXIMO');expect((await (await fetch(`${origin}/api/config`)).json()).regras_versao).toBe(1);
    const published=await fetch(`${origin}/api/regras`,json('PUT',{regras:proposed,autor:'Gestor de demonstração'}));expect(published.status).toBe(201);expect((await published.json()).versao).toBe(2);const next=await upload();expect(next.status).toBe(202);const nextId=(await next.json()).lote_id;expect(nextId).not.toBe(id);advanceMockTime(DEMO_TOTAL_MS);expect((await getLot(nextId)).totais?.aprovadas).toBe(preview.depois.aprovadas);
    const versions=await (await fetch(`${origin}/api/regras/versoes`)).json();expect(versions.versoes.map((v:{versao:number})=>v.versao)).toEqual([2,1]);const detail=await (await fetch(`${origin}/api/regras/versoes/1`)).json();expect(detail.regras.validadores.find((r:{codigo:string})=>r.codigo==='PRAZO_MAXIMO').parametros.prazo_maximo_dias).toBe(180);
  });
  it('valida parâmetros desligados e agrupamento; desconhecido ativo vira aviso de configuração',async()=>{
    const wrong:Regras=structuredClone(REGRAS_INICIAIS);wrong.validadores.find(r=>r.codigo==='PRAZO_MINIMO')!.parametros.prazo_minimo_dias=0;const invalid=await fetch(`${origin}/api/regras`,json('PUT',{regras:wrong,autor:'Teste'}));expect(invalid.status).toBe(400);expect((await invalid.json()).erro.detalhes[0].campo).toContain('prazo_minimo_dias');
    const unknown=structuredClone(REGRAS_INICIAIS);unknown.validadores.push({codigo:'REGRA_X',ativo:true,parametros:{}});expect((await fetch(`${origin}/api/regras`,json('PUT',{regras:unknown,autor:'Teste'}))).status).toBe(201);const id=await completed();const result=await getLot(id);expect(result.erros_configuracao).toEqual([{codigo:'REGRA_X',mensagem:'Regra REGRA_X não existe e foi ignorada.'}]);expect(result.totais?.aprovadas).toBe(19);
  });
  it('aceita agrupamento omitido e rejeita opcional explicitamente vazio',async()=>{
    const rules=structuredClone(REGRAS_INICIAIS);rules.validadores.filter(r=>r.codigo.startsWith('CONCENTRACAO_MAXIMA_')).forEach(r=>{r.ativo=true;delete r.parametros.agrupamento;});
    const accepted=await fetch(`${origin}/api/regras`,json('PUT',{regras:rules,autor:'Teste'}));expect(accepted.status).toBe(201);
    rules.validadores.find(r=>r.codigo==='CONCENTRACAO_MAXIMA_CEDENTE')!.parametros.agrupamento='';
    const rejected=await fetch(`${origin}/api/regras`,json('PUT',{regras:rules,autor:'Teste'}));expect(rejected.status).toBe(400);expect((await rejected.json()).erro.detalhes[0].campo).toContain('agrupamento');
  });
  it('fixtures permitem exercitar as duas regras originais desligadas',async()=>{
    const rules=structuredClone(REGRAS_INICIAIS);rules.validadores.filter(r=>['PRAZO_MINIMO','CONCENTRACAO_MAXIMA_CEDENTE'].includes(r.codigo)).forEach(r=>r.ativo=true);expect((await fetch(`${origin}/api/regras`,json('PUT',{regras:rules,autor:'Teste'}))).status).toBe(201);const id=await completed();const page=await (await fetch(`${origin}/api/lotes/${id}/notas`)).json() as PaginaNotas;const codes=page.itens.flatMap(n=>n.recusas.map(r=>r.codigo));expect(codes).toContain('PRAZO_MINIMO');expect(codes).toContain('CONCENTRACAO_MAXIMA_CEDENTE');
  });
  it('validação síncrona recebe CSV ou JSON snake_case sem gravar lote',async()=>{
    const csv=gerarCSVExemplo();const response=await fetch(`${origin}/api/validar`,{method:'POST',headers:{'Content-Type':'text/csv'},body:csv});expect(response.status).toBe(200);const result=await response.json();expect(result.notas).toHaveLength(30);expect(result.notas[0].liquido_centavos).toBe(980392);
    const nota={nome_fundo:'ANTECIPE FIDC',doc_fundo:'92.234.400/0001-80',nome_cedente:'Empresa fictícia',doc_cedente:'11.222.333/0001-81',nome_sacado:'Cliente fictício',doc_sacado:'11.444.777/0001-61',nu_documento:'API-MOCK',tipo_recebivel:'Duplicata',valor_nominal:'10000,00',data_vencimento_original:'23/11/2026',data_aquisicao:''};const jsonResponse=await fetch(`${origin}/api/validar`,json('POST',{data_base:'2026-10-24',notas:[nota]}));const single=await jsonResponse.json();expect(single.notas[0].liquido_centavos).toBe(980392);expect((await fetch(`${origin}/api/regras/previa`,json('POST',{regras:REGRAS_INICIAIS}))).status).toBe(409);
    expect((await fetch(`${origin}/api/validar`,json('POST',{notas:Array(10001).fill(nota)}))).status).toBe(413);
  });
});
