import { useEffect, useState } from 'react';
import { ChevronDown, Code2, Download, LoaderCircle, RefreshCw, ShieldCheck } from 'lucide-react';
import { getRegras, getVersao, getVersoes } from '@/api/client';
import type { DetalheVersao, RegrasPublicadas, VersaoRegras } from '@/api/types';
import { dateBR } from '@/lib/format';
import { downloadText, errorMessage } from './api';
import { PageHeader } from './page-header';
import { Feedback, Loading } from './feedback';
import { Button } from './ui/button';
import { Modal } from './ui/dialog';
import { parameterValue, ruleDescription, ruleTitle } from './rule-labels';

export function RulesPage() {
  const [data, setData] = useState<RegrasPublicadas | null>(null);
  const [versions, setVersions] = useState<VersaoRegras[] | null>(null);
  const [error, setError] = useState('');
  const [historyError, setHistoryError] = useState('');
  const [versionDetail, setVersionDetail] = useState<DetalheVersao | null>(null);
  const [versionBusy, setVersionBusy] = useState<number | null>(null);
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    getRegras(controller.signal).then((value) => {
      if (!value?.regras || !Array.isArray(value.catalogo) || !value.precificacao_efetiva) throw new Error('A API não disponibilizou a configuração das regras. Tente novamente.');
      if (!controller.signal.aborted) { setData(value); setError(''); }
    }).catch((cause) => { if (!controller.signal.aborted) setError(errorMessage(cause)); });
    getVersoes(controller.signal).then((value) => { if (!controller.signal.aborted) { setVersions(value.versoes); setHistoryError(''); } }).catch((cause) => { if (!controller.signal.aborted) setHistoryError(errorMessage(cause)); });
    return () => controller.abort();
  }, [attempt]);
  async function openVersion(version: number) {
    setVersionBusy(version); setHistoryError('');
    try { setVersionDetail(await getVersao(version)); }
    catch (cause) { setHistoryError(errorMessage(cause)); }
    finally { setVersionBusy(null); }
  }
  return <main className="container page-content"><PageHeader title="Regras do fundo" description="Conheça os critérios antes de enviar suas notas." />
    {error ? <><Feedback>{error}</Feedback><Button variant="outline" style={{ marginTop: 12 }} onClick={() => { setError(''); setAttempt((value) => value + 1); }}><RefreshCw size={15} />Tentar novamente</Button></> : null}{!data && !error ? <Loading label="Carregando as regras publicadas…" /> : null}
    {data ? <><div className="rules-header"><span className="version-tag">Versão {data.versao}</span><span className="helper">Publicada em {dateBR(data.publicada_em)} · {data.regras.fundo.nome}</span></div><div className="rules-layout"><section className="panel rules-panel" aria-label="Regras de elegibilidade">{data.regras.validadores.map((rule, index) => { const catalog = data.catalogo.find((item) => item.codigo === rule.codigo); return <div className="rule-row" key={rule.codigo}><span className="rule-number">{String(index + 1).padStart(2, '0')}</span><div className="rule-copy"><h3>{ruleTitle(rule.codigo, data.catalogo)}</h3><p>{ruleDescription(rule, data.catalogo)}</p>{catalog?.parametros.map((parameter) => <p key={parameter.nome} style={{ marginTop: 5 }}><strong style={{ fontWeight: 500 }}>{parameter.rotulo}: </strong>{parameterValue(rule.parametros[parameter.nome], parameter)}</p>)}</div><span className={`rule-state${rule.ativo ? '' : ' inactive'}`}>{rule.ativo ? 'Ativa' : 'Inativa'}</span></div>; })}</section><aside className="panel rules-aside"><h2>Preço com clareza</h2><div className="pricing-number">{data.precificacao_efetiva.taxa_mensal_percentual.toLocaleString('pt-BR')}%<small>ao mês</small></div><p className="helper">{data.precificacao_efetiva.descricao}</p><hr /><div className="detail-line"><span>Base de cálculo</span><strong>{data.precificacao_efetiva.base_dias.toLocaleString('pt-BR')} dias</strong></div><p className="helper">As análises usam a precificação em vigor. Cada lote registra o método, a taxa e a base utilizados.</p><hr /><div style={{ display: 'flex', alignItems: 'center', gap: 9 }}><ShieldCheck size={19} style={{ color: 'var(--blue)' }} /><h3 style={{ fontSize: 13, letterSpacing: 0 }}>Sem notas repetidas</h3></div><p className="helper">Notas já cedidas e repetições no mesmo arquivo são identificadas automaticamente.</p><Button variant="outline" onClick={() => downloadText(`antecipe-regras-v${data.versao}.json`, JSON.stringify(data.regras, null, 2), 'application/json;charset=utf-8')}><Download size={15} />Baixar regras em JSON</Button><p className="helper" style={{ fontSize: 10 }}>CNPJ do fundo: {data.regras.fundo.cnpj}</p></aside></div><details className="panel json-panel"><summary><Code2 size={16} />Consultar o JSON publicado<ChevronDown size={15} style={{ marginLeft: 'auto' }} /></summary><p className="helper" style={{ padding: '0 20px 12px' }}>A precificação efetiva exibida acima considera a configuração em vigor no servidor.</p><pre className="json-view">{JSON.stringify(data.regras, null, 2)}</pre></details><section className="public-history"><h2>Histórico de versões</h2>{historyError ? <Feedback>{historyError}</Feedback> : null}{!versions && !historyError ? <Loading label="Carregando o histórico…" /> : null}<div className="public-history-list">{versions?.map((version) => <div className="public-history-item" key={version.versao}><strong>Versão {version.versao}</strong><span>{dateBR(version.publicada_em)} · {version.autor}</span>{version.versao === data.versao ? <span className="version-tag">Em vigor</span> : null}<Button variant="ghost" size="small" disabled={versionBusy !== null} onClick={() => { void openVersion(version.versao); }}>{versionBusy === version.versao ? <LoaderCircle className="spin" size={14} /> : <Code2 size={14} />}Ver JSON</Button>{version.resumo.map((item, index) => <p key={`${index}-${item}`} className="helper" style={{ width: '100%' }}>{item}</p>)}</div>)}</div><p className="helper" style={{ marginTop: 13 }}>Cada lote mantém a configuração utilizada na análise. Novas publicações valem para os próximos lotes.</p></section></> : null}
    <Modal open={versionDetail !== null} onOpenChange={(open) => { if (!open) setVersionDetail(null); }} title={`Regras versão ${versionDetail?.versao ?? ''}`} description={versionDetail ? `Publicada em ${dateBR(versionDetail.publicada_em)} por ${versionDetail.autor}.` : ''} className="raw-dialog" footer={<Button variant="outline" onClick={() => { if (versionDetail) downloadText(`antecipe-regras-v${versionDetail.versao}.json`, JSON.stringify(versionDetail.regras, null, 2), 'application/json;charset=utf-8'); }}><Download size={15} />Baixar esta versão</Button>}><pre className="json-view">{versionDetail ? JSON.stringify(versionDetail.regras, null, 2) : ''}</pre></Modal>
  </main>;
}
