import { useEffect, useState, type ReactNode } from 'react';
import { ArrowRight, CheckCircle2, Code2, Download, Eye, History, ListChecks, LoaderCircle, LogOut, RefreshCw, Send, ShieldCheck, Undo2 } from 'lucide-react';
import { logoutAccount } from '@/api/auth';
import { ApiError, MOCK_MODE, clearGestorSenha, getGestorSenha, setGestorSenha, getRegras, getVersao, getVersoes, previaRegras, publicarRegras, verificarGestor } from '@/api/client';
import type { DetalheVersao, ParametroCatalogo, Previa, Regras, RegrasPublicadas, VersaoRegras } from '@/api/types';
import { useRouter } from '@/lib/router';
import './manager-page.css';
import { dateBR, moneyCentavos } from '@/lib/format';
import { validateDraft, withRuleParameter } from '@/lib/rule-draft';
import { downloadText, errorMessage } from './api';
import { Feedback, Loading } from './feedback';
import { notify } from './notifications';
import { PageHeader } from './page-header';
import { Button } from './ui/button';
import { Modal } from './ui/dialog';
import { Switch } from './ui/switch';
import { parameterValue, ruleDescription, ruleTitle } from './rule-labels';

type Tab = 'editor' | 'history' | 'json';
const stringify = (value: unknown) => JSON.stringify(value, null, 2);
function isAbort(cause: unknown): boolean { return cause instanceof Error && cause.name === 'AbortError'; }
function requireRules(value: RegrasPublicadas): RegrasPublicadas {
  if (!value?.regras || !Array.isArray(value.regras.validadores) || !Array.isArray(value.catalogo) || !value.precificacao_efetiva || !Array.isArray(value.catalogo_precificacao?.parametros)) throw new Error('A API não disponibilizou a configuração das regras. Tente novamente.');
  return value;
}

function RequestFeedback({ cause }: { cause: unknown }) {
  return <Feedback><p>{errorMessage(cause)}</p>{cause instanceof ApiError && cause.erro.detalhes?.length ? <ul style={{ paddingLeft: 18, marginTop: 8 }}>{cause.erro.detalhes.map((detail, index) => <li key={`${detail.campo}-${index}`}>{detail.campo}: {detail.problema}</li>)}</ul> : null}</Feedback>;
}

export function ManagerPage() {
  const [entering, setEntering] = useState(MOCK_MODE);
  useEffect(() => {
    if (!MOCK_MODE) return;
    setGestorSenha('demo-gestor');
    const timer = window.setTimeout(() => setEntering(false), 0);
    return () => window.clearTimeout(timer);
  }, []);
  if (entering) return null;
  return <ManagerContent />;
}

function ManagerContent() {
  const router = useRouter();
  const [rules, setRules] = useState<RegrasPublicadas | null>(null);
  const [initialError, setInitialError] = useState<unknown>(null);
  const [authenticated, setAuthenticated] = useState(MOCK_MODE);
  const [checkingAccess, setCheckingAccess] = useState(!MOCK_MODE);
  const [password, setPassword] = useState('');
  const [loginError, setLoginError] = useState<unknown>(null);
  const [loginBusy, setLoginBusy] = useState(false);
  const [loadAttempt, setLoadAttempt] = useState(0);

  useEffect(() => {
    const controller = new AbortController();
    getRegras(controller.signal).then(requireRules).then((data) => { if (!controller.signal.aborted) { setRules(data); setInitialError(null); } }).catch((cause) => { if (!isAbort(cause) && !controller.signal.aborted) setInitialError(cause); });
    return () => controller.abort();
  }, [loadAttempt]);

  useEffect(() => {
    if (MOCK_MODE) return;
    let alive = true;
    const saved = getGestorSenha();
    verificarGestor(saved).then(() => { if (alive) setAuthenticated(true); }).catch((cause) => { if (alive) { clearGestorSenha(); setLoginError(cause); } }).finally(() => { if (alive) setCheckingAccess(false); });
    return () => { alive = false; };
  }, []);

  async function login() {
    if (loginBusy) return;
    setLoginError(null); setLoginBusy(true);
    try { await verificarGestor(password); setAuthenticated(true); setPassword(''); }
    catch (cause) { setLoginError(cause); }
    finally { setLoginBusy(false); }
  }
  function logout(cause?: unknown) { if (MOCK_MODE) { clearGestorSenha(); router.push('/dashboard'); return; } if (!cause) { void logoutAccount().then(() => { clearGestorSenha(); window.location.assign('/login'); }).catch(setLoginError); return; } clearGestorSenha(); setAuthenticated(false); setPassword(''); setLoginError(cause); }

  return <main className="container page-content manager-page"><PageHeader showRoleSwitch title={authenticated ? 'Gestão das regras' : 'Área do gestor'} description={authenticated ? 'Edite com segurança. Confira o impacto antes de publicar.' : 'Gerencie as regras do fundo, confira o impacto de uma alteração e publique novas versões.'} />
    {initialError ? <div style={{ marginBottom: 20 }}><RequestFeedback cause={initialError} /><Button variant="outline" size="small" style={{ marginTop: 12 }} onClick={() => { setInitialError(null); setLoadAttempt((value) => value + 1); }}><RefreshCw size={14} />Tentar carregar novamente</Button></div> : null}
    {checkingAccess ? <Loading label="Conferindo o acesso à área do gestor…" /> : !authenticated ? <section className="panel login-wrap"><div className="login-icon"><ShieldCheck size={24} /></div><h2>Acessar a gestão</h2><p>Informe sua senha de acesso para continuar.</p><form className="login-form" onSubmit={(event) => { event.preventDefault(); void login(); }}><div className="field"><label htmlFor="gestor-password">Senha de acesso</label><input className="input" id="gestor-password" type="password" autoComplete="current-password" required value={password} onChange={(event) => setPassword(event.target.value)} disabled={loginBusy} /></div><Button disabled={loginBusy || !password}>{loginBusy ? <LoaderCircle className="spin" size={16} /> : null}Acessar área do gestor<ArrowRight size={16} /></Button></form>{MOCK_MODE ? <div className="demo-login"><p className="helper">Explore a gestão com dados fictícios. As alterações ficam neste ambiente de demonstração.</p><Button variant="outline" disabled={loginBusy} onClick={() => setPassword('demo-gestor')}>Preencher senha da demonstração</Button></div> : null}{loginError ? <RequestFeedback cause={loginError} /> : null}</section> : rules ? <ManagerWorkspace initial={rules} onRulesChange={setRules} onLogout={logout} /> : !initialError ? <Loading label="Carregando as regras publicadas…" /> : null}
  </main>;
}

function ParameterField({ parameter, value, id, onChange, readOnly = false, source }: { parameter: ParametroCatalogo; value: unknown; id: string; onChange: (value: unknown) => void; readOnly?: boolean; source?: string }) {
  const label = `${parameter.rotulo}${parameter.unidade ? ` (${parameter.unidade})` : ''}`;
  if (parameter.tipo === 'lista_texto' && parameter.opcoes?.length && !readOnly) {
    const selected = Array.isArray(value) ? value.map(String) : [];
    const options = [...new Set([...parameter.opcoes, ...selected])];
    return <fieldset className="field" style={{ border: 0, padding: 0, margin: 0 }}><legend style={{ fontSize: 13, fontWeight: 500, marginBottom: 12 }}>{label}</legend><div style={{ display: 'flex', gap: '9px 18px', flexWrap: 'wrap' }}>{options.map((option) => <label key={option} style={{ display: 'flex', alignItems: 'center', gap: 7, fontWeight: 400, fontSize: 14 }}><input type="checkbox" checked={selected.includes(option)} onChange={(event) => onChange(event.target.checked ? [...selected, option] : selected.filter((item) => item !== option))} />{option}</label>)}</div></fieldset>;
  }
  if (parameter.tipo === 'opcao') return <div className="field"><label htmlFor={id}>{label}</label><select className="input" id={id} disabled={readOnly} value={typeof value === 'string' ? value : ''} onChange={(event) => onChange(parameter.opcional && event.target.value === '' ? undefined : event.target.value)}><option value="" disabled={!parameter.opcional}>{parameter.opcional ? 'Usar padrão' : 'Selecione uma opção'}</option>{parameter.opcoes?.map((option) => <option key={option} value={option}>{option}</option>)}</select>{readOnly ? <span className="helper">Definido em {source ?? 'config/antecipe.yaml'}.</span> : null}</div>;
  const numeric = parameter.tipo === 'inteiro' || parameter.tipo === 'decimal';
  const displayed = Array.isArray(value) ? value.join('; ') : typeof value === 'number' && !Number.isFinite(value) ? '' : value === undefined || value === null ? '' : String(value);
  return <div className="field"><label htmlFor={id}>{label}</label><input id={id} className="input" type={numeric ? 'number' : 'text'} min={parameter.minimo} max={parameter.maximo} step={parameter.tipo === 'inteiro' ? 1 : 'any'} value={displayed} readOnly={readOnly} onChange={(event) => { const raw = event.target.value; onChange(numeric ? raw === '' ? '' : Number(raw) : parameter.tipo === 'lista_texto' ? raw.split(';').map((item) => item.trim()) : raw); }} />{readOnly ? <span className="helper">Definido em {source ?? 'config/antecipe.yaml'}.</span> : parameter.tipo === 'lista_texto' ? <span className="helper">Separe os valores por ponto e vírgula.</span> : null}</div>;
}

function ManagerWorkspace({ initial, onRulesChange, onLogout }: { initial: RegrasPublicadas; onRulesChange: (rules: RegrasPublicadas) => void; onLogout: (cause?: unknown) => void }) {
  const [tab, setTab] = useState<Tab>('editor');
  const [draft, setDraft] = useState<Regras>(() => structuredClone(initial.regras));
  const [author, setAuthor] = useState('');
  const [lotId, setLotId] = useState('');
  const [preview, setPreview] = useState<{ data: Previa; signature: string } | null>(null);
  const [previewBusy, setPreviewBusy] = useState(false);
  const [publishOpen, setPublishOpen] = useState(false);
  const [publishBusy, setPublishBusy] = useState(false);
  const [publishCommitted, setPublishCommitted] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [versions, setVersions] = useState<VersaoRegras[] | null>(null);
  const [historyError, setHistoryError] = useState<unknown>(null);
  const [historyVersion, setHistoryVersion] = useState<DetalheVersao | null>(null);
  const [historyBusy, setHistoryBusy] = useState<number | null>(null);
  const [historyAttempt, setHistoryAttempt] = useState(0);
  const dirty = stringify(draft) !== stringify(initial.regras);
  const signature = stringify({ regras: draft, lote_id: lotId.trim() });
  const previewCurrent = preview?.signature === signature;
  const locked = publishBusy || publishCommitted;

  useEffect(() => {
    const controller = new AbortController();
    getVersoes(controller.signal).then((value) => { if (!controller.signal.aborted) { setVersions(value.versoes); setHistoryError(null); } }).catch((cause) => { if (!isAbort(cause) && !controller.signal.aborted) setHistoryError(cause); });
    return () => controller.abort();
  }, [historyAttempt]);

  function handleFailure(cause: unknown) { if (cause instanceof ApiError && cause.status === 401) onLogout(cause); else setError(cause); }
  function updateDraft(value: Regras) { setDraft(value); setError(null); }
  function updateRule(index: number, patch: Partial<Regras['validadores'][number]>) { updateDraft({ ...draft, validadores: draft.validadores.map((rule, position) => position === index ? { ...rule, ...patch } : rule) }); }
  async function createPreview() {
    if (previewBusy || locked) return;
    setError(null);
    try {
      validateDraft(draft, initial);
      setPreviewBusy(true);
      const data = await previaRegras(draft, lotId.trim() || undefined);
      setPreview({ data, signature });
    } catch (cause) { handleFailure(cause); }
    finally { setPreviewBusy(false); }
  }
  async function refreshPublished() {
    const updated = requireRules(await getRegras());
    onRulesChange(updated); setDraft(structuredClone(updated.regras)); setPublishCommitted(false); setHistoryAttempt((value) => value + 1);
  }
  async function publish() {
    if (publishBusy || !previewCurrent || !author.trim() || publishCommitted) return;
    setPublishBusy(true); setError(null);
    try {
      const version = await publicarRegras(draft, author.trim());
      setPublishOpen(false); setPreview(null); setPublishCommitted(true);
      notify.success(`Versão ${version.versao} publicada. As próximas análises já usam as novas regras.`);
      await refreshPublished();
    } catch (cause) { handleFailure(cause); }
    finally { setPublishBusy(false); }
  }
  async function openHistory(version: number) {
    setHistoryBusy(version); setHistoryError(null);
    try { setHistoryVersion(await getVersao(version)); }
    catch (cause) { setHistoryError(cause); }
    finally { setHistoryBusy(null); }
  }
  const tabs: { id: Tab; label: string; icon: ReactNode }[] = [{ id: 'editor', label: 'Editar regras', icon: <ListChecks size={16} /> }, { id: 'history', label: 'Histórico de versões', icon: <History size={16} /> }, { id: 'json', label: 'Consultar JSON', icon: <Code2 size={16} /> }];
  return <><div className="page-actions"><p className="helper" style={{ marginRight: 'auto' }}>{initial.regras.fundo.nome} · <span className="version-tag">Versão {initial.versao} em vigor</span> · Publicada em {dateBR(initial.publicada_em)}</p><div className="manager-session"><span>{MOCK_MODE ? 'Acesso de demonstração' : 'Sessão do gestor'}</span><Button variant="ghost" size="small" onClick={() => onLogout()}><LogOut size={14} />Sair</Button></div></div>
    <div className="manager-tabs" role="tablist" aria-label="Gerenciar regras" onKeyDown={(event) => { if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return; event.preventDefault(); const index = tabs.findIndex((item) => item.id === tab); const next = event.key === 'Home' ? 0 : event.key === 'End' ? tabs.length - 1 : (index + (event.key === 'ArrowRight' ? 1 : -1) + tabs.length) % tabs.length; setTab(tabs[next].id); document.getElementById(`tab-${tabs[next].id}`)?.focus(); }}>{tabs.map((item) => <button key={item.id} role="tab" tabIndex={tab === item.id ? 0 : -1} aria-selected={tab === item.id} aria-controls={`panel-${item.id}`} id={`tab-${item.id}`} className={tab === item.id ? 'selected' : ''} onClick={() => { setTab(item.id); setError(null); }}>{item.icon}{item.label}</button>)}</div>
    {error && !publishOpen ? <div style={{ marginBottom: 20 }}><RequestFeedback cause={error} /></div> : null}
    {publishCommitted ? <div style={{ marginBottom: 20 }}><Feedback variant="info">A publicação foi registrada. Atualize a configuração antes de fazer novas alterações.</Feedback><Button variant="outline" style={{ marginTop: 10 }} onClick={() => { void refreshPublished().catch(handleFailure); }}><RefreshCw size={14} />Atualizar regras publicadas</Button></div> : null}
    {tab === 'editor' ? <div id="panel-editor" role="tabpanel" aria-labelledby="tab-editor" className="manager-layout"><div><section className="panel manager-editor"><div className="section-heading"><h2>Critérios de elegibilidade</h2><p>Ative as regras e ajuste os parâmetros para os próximos lotes.</p></div><fieldset disabled={locked} style={{ border: 0, padding: 0, margin: 0, minWidth: 0 }}><div className="manager-fund"><div className="field"><label htmlFor="fundo-nome">Nome do fundo</label><input id="fundo-nome" className="input" value={draft.fundo.nome} onChange={(event) => updateDraft({ ...draft, fundo: { ...draft.fundo, nome: event.target.value } })} /></div><div className="field"><label htmlFor="fundo-cnpj">CNPJ do fundo</label><input id="fundo-cnpj" className="input" value={draft.fundo.cnpj} onChange={(event) => updateDraft({ ...draft, fundo: { ...draft.fundo, cnpj: event.target.value } })} /></div></div>
      {draft.validadores.map((rule, index) => { const metadata = initial.catalogo.find((item) => item.codigo === rule.codigo); return <div key={rule.codigo} className={`edit-rule-row${rule.ativo ? '' : ' inactive'}`}><div className="edit-rule-heading"><h3>{ruleTitle(rule.codigo, initial.catalogo)}</h3><Switch label={`Ativar regra: ${ruleTitle(rule.codigo, initial.catalogo)}`} checked={rule.ativo} disabled={locked} onCheckedChange={(ativo) => updateRule(index, { ativo })} /></div><p>{ruleDescription(rule, initial.catalogo)}</p>{metadata?.parametros.length ? <div className="edit-rule-params">{metadata.parametros.map((parameter) => <ParameterField key={parameter.nome} parameter={parameter} value={rule.parametros[parameter.nome]} id={`${rule.codigo}-${parameter.nome}`} onChange={(value) => updateRule(index, { parametros: withRuleParameter(rule.parametros, parameter.nome, value) })} />)}</div> : null}{!metadata ? <p className="helper">O catálogo não contém os parâmetros desta regra.</p> : null}</div>; })}
      <div className="pricing-editor"><h3>Precificação</h3><p className="helper" style={{ margin: '12px 0' }}>{initial.precificacao_efetiva.descricao}</p><div className="detail-line"><span>Método em vigor</span><strong>{initial.precificacao_efetiva.metodo}</strong></div><p className="helper" style={{ margin: '8px 0 16px' }}>Método somente leitura, definido em {initial.precificacao_efetiva.fonte_metodo ?? 'config/antecipe.yaml'}.</p><div className="pricing-fields">{initial.catalogo_precificacao.parametros.map((parameter) => { const overridden = initial.precificacao_efetiva.sobrescritos_pelo_yaml?.includes(parameter.nome) ?? false; const value = overridden ? initial.precificacao_efetiva[parameter.nome as keyof typeof initial.precificacao_efetiva] : draft.precificacao[parameter.nome as keyof Regras['precificacao']]; return <ParameterField key={parameter.nome} parameter={parameter} value={value} id={`preco-${parameter.nome}`} readOnly={overridden} source={initial.precificacao_efetiva.fonte_metodo} onChange={(next) => updateDraft({ ...draft, precificacao: { ...draft.precificacao, [parameter.nome]: typeof next === 'number' ? next : Number.NaN } })} />; })}</div></div></fieldset>
      </section>{preview && previewCurrent ? <PreviewPanel preview={preview.data} /> : null}</div><aside className="panel manager-actions"><h2>Revise antes de publicar</h2><p className="helper">Compare a versão publicada e seu rascunho em um lote concluído. As regras só mudam quando você confirma a publicação.</p><div className={`draft-status${dirty ? ' dirty' : ''}`}>{dirty ? 'Há alterações no rascunho' : 'Rascunho igual à versão publicada'}{preview && !previewCurrent ? ' · Refaça a prévia' : ''}</div><div className="field"><label htmlFor="previa-lote">Lote para a prévia (opcional)</label><input className="input" id="previa-lote" value={lotId} onChange={(event) => setLotId(event.target.value)} placeholder="Último lote concluído" disabled={locked} /></div><Button variant="outline" disabled={previewBusy || locked} onClick={() => { void createPreview(); }}>{previewBusy ? <LoaderCircle size={16} className="spin" /> : <Eye size={16} />}{previewBusy ? 'Calculando prévia' : 'Gerar prévia de impacto'}</Button><Button disabled={!dirty || !previewCurrent || locked || previewBusy} onClick={() => { setError(null); setPublishOpen(true); }}><Send size={15} />Publicar nova versão</Button><Button variant="ghost" disabled={!dirty || locked} onClick={() => { updateDraft(structuredClone(initial.regras)); setPreview(null); notify.success('Rascunho restaurado para a versão publicada.'); }}><Undo2 size={14} />Descartar alterações</Button><Feedback variant="info">A publicação exige sua confirmação explícita. A prévia não altera as regras publicadas.</Feedback></aside></div> : null}
    {tab === 'history' ? <section id="panel-history" role="tabpanel" aria-labelledby="tab-history" className="panel history-list">{historyError ? <div style={{ padding: 20 }}><RequestFeedback cause={historyError} /><Button variant="outline" size="small" onClick={() => setHistoryAttempt((value) => value + 1)} style={{ marginTop: 12 }}>Tentar novamente</Button></div> : null}{!versions && !historyError ? <Loading label="Carregando o histórico de versões…" /> : null}{versions?.map((version) => <div className="history-row" key={version.versao}><span className="history-version">v{version.versao}</span><div className="history-meta"><h3>Regras versão {version.versao}{version.versao === initial.versao ? <span className="version-tag">Em vigor</span> : null}</h3><p>Publicada em {dateBR(version.publicada_em)} · {version.autor}</p>{version.resumo.map((item, index) => <p key={`${index}-${item}`}>{item}</p>)}</div><Button variant="outline" size="small" disabled={historyBusy !== null} onClick={() => { void openHistory(version.versao); }}>{historyBusy === version.versao ? <LoaderCircle className="spin" size={14} /> : <Code2 size={14} />}Consultar versão</Button></div>)}</section> : null}
    {tab === 'json' ? <section id="panel-json" role="tabpanel" aria-labelledby="tab-json" className="panel json-editor-panel"><div className="json-editor-heading"><div><h2>JSON do rascunho</h2><p className="helper" style={{ marginTop: 5 }}>Consulte a configuração completa. Ajuste os campos na aba “Editar regras”.</p></div><Button variant="outline" size="small" onClick={() => downloadText(`antecipe-rascunho-v${initial.versao}.json`, stringify(draft), 'application/json;charset=utf-8')}><Download size={14} />Baixar JSON</Button></div><pre className="json-view">{stringify(draft)}</pre></section> : null}
    <Modal open={publishOpen} onOpenChange={(open) => { if (!publishBusy) setPublishOpen(open); }} className="manager-dialog" title="Publicar uma nova versão?" description="As próximas análises passarão a usar as regras do seu rascunho. Os lotes já analisados mantêm sua configuração original." footer={<><Button variant="ghost" disabled={publishBusy} onClick={() => setPublishOpen(false)}>Continuar revisando</Button><Button disabled={publishBusy || !previewCurrent || !author.trim()} onClick={() => { void publish(); }}>{publishBusy ? <LoaderCircle className="spin" size={15} /> : <Send size={14} />}{publishBusy ? 'Publicando' : 'Confirmar publicação'}</Button></>}><div className="dialog-summary"><div className="detail-line"><span>Versão atual</span><strong>Versão {initial.versao}</strong></div><div className="detail-line"><span>Regras ativas</span><strong>{draft.validadores.filter((rule) => rule.ativo).length} de {draft.validadores.length}</strong></div><div className="detail-line"><span>Método em vigor</span><strong>{initial.precificacao_efetiva.metodo}</strong></div>{initial.catalogo_precificacao.parametros.map((parameter) => <div className="detail-line" key={parameter.nome}><span>{parameter.rotulo}</span><strong>{parameterValue(initial.precificacao_efetiva.sobrescritos_pelo_yaml?.includes(parameter.nome) ? initial.precificacao_efetiva[parameter.nome as keyof typeof initial.precificacao_efetiva] : draft.precificacao[parameter.nome as keyof Regras['precificacao']], parameter)}</strong></div>)}</div><div className="field" style={{ margin: '18px 0' }}><label htmlFor="gestor-autor">Nome do responsável pela publicação</label><input id="gestor-autor" className="input" required value={author} disabled={publishBusy} onChange={(event) => setAuthor(event.target.value)} autoComplete="name" /></div><p className="dialog-note">{MOCK_MODE ? 'Esta publicação altera apenas o ambiente de demonstração, com dados fictícios.' : 'Esta ação publica a configuração revisada para as próximas análises.'}</p>{error ? <RequestFeedback cause={error} /> : null}</Modal>
    <Modal open={historyVersion !== null} onOpenChange={(open) => { if (!open) setHistoryVersion(null); }} title={`Regras versão ${historyVersion?.versao ?? ''}`} description={historyVersion ? `Publicada em ${dateBR(historyVersion.publicada_em)} por ${historyVersion.autor}.` : ''} className="raw-dialog manager-dialog" footer={<Button variant="outline" onClick={() => { if (historyVersion) downloadText(`antecipe-regras-v${historyVersion.versao}.json`, stringify(historyVersion.regras), 'application/json;charset=utf-8'); }}><Download size={15} />Baixar esta versão</Button>}><pre className="json-view">{historyVersion ? stringify(historyVersion.regras) : ''}</pre></Modal>
  </>;
}

function PreviewPanel({ preview }: { preview: Previa }) {
  const delta = preview.depois.liquido_centavos - preview.antes.liquido_centavos;
  return <section className="panel preview-panel" aria-live="polite"><h2>Prévia de impacto</h2><p>Comparação sobre {preview.linhas.toLocaleString('pt-BR')} notas. Lote {preview.lote_id}.</p><div className="preview-grid"><div className="preview-block"><h3>Versão {preview.antes.regras_versao}</h3><strong>{moneyCentavos(preview.antes.liquido_centavos)}</strong><p>{preview.antes.aprovadas.toLocaleString('pt-BR')} aprovadas · {preview.antes.recusadas.toLocaleString('pt-BR')} recusadas</p></div><div className="preview-block after"><h3>Com seu rascunho</h3><strong>{moneyCentavos(preview.depois.liquido_centavos)}</strong><p>{preview.depois.aprovadas.toLocaleString('pt-BR')} aprovadas · {preview.depois.recusadas.toLocaleString('pt-BR')} recusadas</p></div></div><div className="preview-delta"><span>Variação do valor líquido</span><strong>{delta > 0 ? '+' : ''}{moneyCentavos(delta)}</strong></div>{preview.mudancas.length ? <ul style={{ margin: '16px 0', paddingLeft: 18 }}>{preview.mudancas.map((change, index) => <li key={`${index}-${change}`} style={{ fontSize: 12, marginBottom: 5 }}>{change}</li>)}</ul> : null}{preview.motivos.length ? <div className="table-scroll" style={{ marginTop: 16 }}><table className="data-table"><thead><tr><th scope="col">Motivo</th><th scope="col">Antes</th><th scope="col">Depois</th></tr></thead><tbody>{preview.motivos.map((reason) => <tr key={reason.codigo}><td>{reason.titulo}</td><td>{reason.antes.toLocaleString('pt-BR')}</td><td>{reason.depois.toLocaleString('pt-BR')}</td></tr>)}</tbody></table></div> : null}<p className="helper" style={{ marginTop: 13, display: 'flex', gap: 6, alignItems: 'center', fontSize: 11 }}><CheckCircle2 size={14} style={{ color: 'var(--green)' }} />A prévia não altera as regras nem registra cessão.</p></section>;
}
