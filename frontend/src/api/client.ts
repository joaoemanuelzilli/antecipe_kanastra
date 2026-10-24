import type { ApiErro,Config,DetalheVersao,FiltrosNotas,Lote,LoteLocal,PaginaNotas,Previa,Regras,RegrasPublicadas,ResultadoCessao,UploadResposta,VersaoRegras } from './types';
export const MOCK_MODE=import.meta.env.VITE_MOCK_API==='true';
const base=(import.meta.env.VITE_API_BASE_URL??'').replace(/\/$/,'');
export function apiURL(path:string){return `${base}${path}`;}
export class ApiError extends Error {constructor(public erro:ApiErro,public status:number){super(erro.mensagem);this.name='ApiError';}}
export async function request<T>(path:string,options:RequestInit={}):Promise<T>{let response:Response;try{response=await fetch(apiURL(path),{...options,headers:{...options.headers},cache:'no-store'});}catch(error){if(error instanceof Error&&error.name==='AbortError')throw error;throw new ApiError({codigo:'SEM_CONEXAO',mensagem:'Não foi possível conectar à API. Verifique a conexão e tente novamente.'},0);}if(response.status===204)return undefined as T;const body=await response.json().catch(()=>null);if(!response.ok)throw new ApiError(body?.erro??{codigo:'RESPOSTA_INVALIDA',mensagem:'O servidor não conseguiu concluir. Tente novamente.'},response.status);if(body===null)throw new ApiError({codigo:'RESPOSTA_INVALIDA',mensagem:'A API devolveu uma resposta inválida.'},502);return body as T;}
export function jsonRequest<T>(path:string,body:unknown,options:RequestInit={}){return request<T>(path,{method:'POST',...options,headers:{'Content-Type':'application/json',...options.headers},body:JSON.stringify(body)});}
export function getConfig(signal?:AbortSignal){return request<Config>('/api/config',{signal});}
export function getLote(id:string,signal?:AbortSignal){return request<Lote>(`/api/lotes/${encodeURIComponent(id)}`,{signal});}
export function filtrosQuery(filters:FiltrosNotas,download=false){const params=new URLSearchParams();for(const [key,value] of Object.entries(filters))if(value!==undefined&&value!==''&&(!download||['status','motivo','busca'].includes(key)))params.set(key,String(value));return params.toString();}
export function getNotas(id:string,filters:FiltrosNotas,signal?:AbortSignal){return request<PaginaNotas>(`/api/lotes/${encodeURIComponent(id)}/notas?${filtrosQuery(filters)}`,{signal});}
export function notasCSVURL(id:string,filters:FiltrosNotas={}){return apiURL(`/api/lotes/${encodeURIComponent(id)}/notas.csv?${filtrosQuery(filters,true)}`);}
export function confirmarCessao(id:string,key:string){return request<ResultadoCessao>(`/api/lotes/${encodeURIComponent(id)}/cessao`,{method:'POST',headers:{'Idempotency-Key':key}});}
const AUTH_KEY='antecipe.gestor.senha';
export function getGestorSenha(){return sessionStorage.getItem(AUTH_KEY)??'';}
export function setGestorSenha(senha:string){sessionStorage.setItem(AUTH_KEY,senha);}
export function clearGestorSenha(){sessionStorage.removeItem(AUTH_KEY);}
function gestorHeaders(){return {'X-Gestor-Senha':getGestorSenha()};}
export async function verificarGestor(senha:string){await request<void>('/api/gestor/verificar',{headers:{'X-Gestor-Senha':senha}});setGestorSenha(senha);}
export function getRegras(signal?:AbortSignal){return request<RegrasPublicadas>('/api/regras',{signal});}
export function getVersoes(signal?:AbortSignal){return request<{versoes:VersaoRegras[]}>('/api/regras/versoes',{signal});}
export function getVersao(versao:number){return request<DetalheVersao>(`/api/regras/versoes/${versao}`);}
export function previaRegras(regras:Regras,lote_id?:string){return jsonRequest<Previa>('/api/regras/previa',{regras,...(lote_id?{lote_id}:{})},{headers:gestorHeaders()});}
export function publicarRegras(regras:Regras,autor:string){return jsonRequest<{versao:number;publicada_em:string}>('/api/regras',{regras,autor},{method:'PUT',headers:gestorHeaders()});}
export function uploadLote(arquivo:File,data_base:string,onProgress:(percent:number)=>void,signal?:AbortSignal,idempotencyKey?:string):Promise<UploadResposta>{return new Promise((resolve,reject)=>{const xhr=new XMLHttpRequest(),form=new FormData();form.set('arquivo',arquivo);form.set('data_base',data_base);xhr.open('POST',apiURL('/api/lotes'));if(idempotencyKey)xhr.setRequestHeader('Idempotency-Key',idempotencyKey);xhr.responseType='json';xhr.upload.onprogress=event=>{if(event.lengthComputable)onProgress(Math.round(event.loaded/event.total*100));};const abort=()=>xhr.abort();signal?.addEventListener('abort',abort,{once:true});const cleanup=()=>signal?.removeEventListener('abort',abort);xhr.onload=()=>{cleanup();if(xhr.status>=200&&xhr.status<300)resolve(xhr.response as UploadResposta);else reject(new ApiError(xhr.response?.erro??{codigo:'FALHA_UPLOAD',mensagem:'Não conseguimos enviar o arquivo. Tente novamente.'},xhr.status));};xhr.onerror=()=>{cleanup();reject(new ApiError({codigo:'SEM_CONEXAO',mensagem:'Não foi possível conectar à API. Tente novamente.'},0));};xhr.onabort=()=>{cleanup();reject(new DOMException('Envio cancelado.','AbortError'));};if(signal?.aborted){cleanup();reject(new DOMException('Envio cancelado.','AbortError'));return;}xhr.send(form);});}
const LOTES_KEY=`antecipe.lotes.v1.${MOCK_MODE?'demo':base||'local-api'}`;
export function lotesLocais():LoteLocal[]{try{return JSON.parse(localStorage.getItem(LOTES_KEY)??'[]') as LoteLocal[];}catch{return [];}}
export function registrarLote(id:string,arquivo:string,nome?:string){
  const list=lotesLocais();
  const normalizedName=nome===undefined?'':[...nome.trim()].slice(0,80).join('').trim();
  const existing=list.find(l=>l.id===id);
  if(existing){
    if(!normalizedName||existing.nome===normalizedName)return;
    existing.nome=normalizedName;
  }else{
    list.unshift({id,arquivo,criado_em:new Date().toISOString(),...(normalizedName?{nome:normalizedName}:{})});
  }
  localStorage.setItem(LOTES_KEY,JSON.stringify(list.slice(0,100)));
}
export function nomeArquivoLote(id:string){return lotesLocais().find(l=>l.id===id)?.arquivo??'Lote de notas';}
export function nomeLote(id:string){const local=lotesLocais().find(l=>l.id===id);return local?.nome||local?.arquivo||'Lote de notas';}
