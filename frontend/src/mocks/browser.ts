import { BatchInterceptor } from '@mswjs/interceptors';
import { FetchInterceptor } from '@mswjs/interceptors/fetch';
import { XMLHttpRequestInterceptor } from '@mswjs/interceptors/XMLHttpRequest';
import { HttpResponse, isPassthroughResponse } from 'msw';
import { getResponse } from 'msw/utils/get-response';
import { handlers } from './handlers';

type DemoInterceptor=BatchInterceptor<[FetchInterceptor,XMLHttpRequestInterceptor]>;
interface MockRuntime {interceptor:DemoInterceptor;handlers:typeof handlers;started?:Promise<void>;cleanup?:Promise<void>}
const page=globalThis as typeof globalThis & {__antecipeMockTransportV2?:MockRuntime};

async function removeLegacyMockWorker():Promise<void>{
  if(typeof navigator==='undefined' || !('serviceWorker' in navigator) || typeof location==='undefined')return;
  const registrations=await navigator.serviceWorker.getRegistrations();
  await Promise.all(registrations.map(async registration=>{
    const scripts=[registration.active,registration.waiting,registration.installing].filter((worker):worker is ServiceWorker=>worker!==null);
    // Uma inscrição com qualquer outro script é preservada, mesmo no mesmo origin.
    if(!scripts.length || !scripts.every(worker=>{const url=new URL(worker.scriptURL);return url.origin===location.origin && url.pathname==='/mockServiceWorker.js';}))return;
    await registration.unregister();
  }));
}
function createRuntime():MockRuntime{
  const interceptor=new BatchInterceptor({name:'antecipe-explicit-demo',interceptors:[new FetchInterceptor(),new XMLHttpRequestInterceptor()]});
  const runtime:MockRuntime={interceptor,handlers};
  interceptor.on('request',async({request,controller})=>{
    try{
      const response=await getResponse(runtime.handlers,request,{quiet:true});
      // getResponse devolve o marcador302 de passthrough(), não uma resposta de rede.
      // Sem respondWith, o interceptor deixa o cliente original executar a requisição.
      if(response && !isPassthroughResponse(response))controller.respondWith(response);
    }catch{
      // Uma falha do mock explícito nunca deve cair na API real por acidente.
      controller.respondWith(HttpResponse.json({erro:{codigo:'FALHA_DEMONSTRACAO',mensagem:'Não foi possível processar esta operação na demonstração.',detalhes:[]}},{status:500}));
    }
  });
  return runtime;
}
// A intercepção vive na página: não depende do ciclo de vida de Service Workers.
// HMR conserva o mesmo BatchInterceptor e troca apenas a lista de handlers.
const runtime=page.__antecipeMockTransportV2??=createRuntime();
runtime.handlers=handlers;
export const interceptor=runtime.interceptor;
/** Nome mantido para o bootstrap; esta função não registra nem inicia Service Worker. */
export function startMockWorker():Promise<void>{
  return runtime.started??=Promise.resolve().then(()=>{
    interceptor.apply();
    // Migração opcional em segundo plano: a UI nunca aguarda a infraestrutura SW antiga.
    runtime.cleanup=removeLegacyMockWorker().catch(()=>{console.warn('[Antecipe] Não foi possível remover a inscrição antiga da demonstração.');});
  });
}
if(import.meta.hot){
  import.meta.hot.accept('./handlers',module=>{if(module)runtime.handlers=module.handlers;});
  import.meta.hot.accept();
}
