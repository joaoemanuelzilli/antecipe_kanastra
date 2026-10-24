import { useEffect, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import * as Dialog from '@radix-ui/react-dialog';
import { X } from 'lucide-react';
import type { FormEvent } from 'react';
import type { ErroDetalhe } from '@/api/types';
import { ApiError, getConfig, getLote, MOCK_MODE, registrarLote, uploadLote } from '@/api/client';
import { useNavigate } from 'react-router-dom';
import { cx } from '@/lib/format';
import { Feedback } from './feedback';
import { DocumentLoadingPanel } from './document-loading';
import './new-lot.css';

const errorMessage = (error: unknown) => error instanceof Error ? error.message : 'Não foi possível enviar o arquivo. Tente novamente.';

export function UploadPage({ onDismiss }: { onDismiss?: () => void } = {}) {
  const navigate = useNavigate();
  const config = useQuery({ queryKey: ['config'], queryFn: ({ signal }) => getConfig(signal) });
  const form = useRef<HTMLFormElement>(null);
  const fileInput = useRef<HTMLInputElement>(null);
  const nameInput = useRef<HTMLInputElement>(null);
  const dateInput = useRef<HTMLInputElement>(null);
  const dropButton = useRef<HTMLButtonElement>(null);
  const title = useRef<HTMLHeadingElement>(null);
  const previousFocus = useRef<HTMLElement | null>(null);
  const controller = useRef<AbortController | null>(null);
  const mounted = useRef(true);
  const inFlight = useRef(false);
  const closed = useRef(false);
  const exampleKey = useRef<{ date: string; key: string } | null>(null);
  const [open, setOpen] = useState(true);
  const [guideOpen, setGuideOpen] = useState(false);
  const [name, setName] = useState('');
  const [date, setDate] = useState<string | null>(null);
  const [file, setFile] = useState<File | null>(null);
  const [dragging, setDragging] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [analysis, setAnalysis] = useState<{ id: string; replay: boolean } | null>(null);
  const [preparingExample, setPreparingExample] = useState(false);
  const [progress, setProgress] = useState<number | null>(null);
  const [error, setError] = useState('');
  const [parseErrors, setParseErrors] = useState<ErroDetalhe[]>([]);
  const dataBase = date ?? config.data?.data_base_padrao ?? '';
  const busy = uploading || analysis !== null;
  const disabled = busy || !config.data;
  const analysisQuery = useQuery({
    queryKey: ['lote', analysis?.id],
    queryFn: ({ signal }) => getLote(analysis!.id, signal),
    enabled: analysis !== null,
    refetchInterval: query => query.state.error || query.state.data?.status === 'concluido' || query.state.data?.status === 'erro' ? false : 500,
  });

  const analysisStatus = analysisQuery.data?.status;
  useEffect(() => {
    if (!analysis || closed.current) return;
    if (analysisStatus === 'concluido' || analysisStatus === 'erro') {
      onDismiss?.();
      navigate(`/lote/${analysis.id}${analysis.replay ? '?replay=1' : ''}`);
    }
  }, [analysis, analysisStatus, navigate, onDismiss]);

  useEffect(() => { mounted.current = true; return () => { mounted.current = false; controller.current?.abort(); }; }, []);

  function restoreDashboardFocus() {
    window.requestAnimationFrame(() => {
      if (window.location.pathname !== '/dashboard') return;
      const trigger = document.querySelector<HTMLAnchorElement>('.dashboard-upload-link');
      if (trigger) trigger.focus({ preventScroll: true });
      else if (previousFocus.current?.isConnected) previousFocus.current.focus({ preventScroll: true });
    });
  }
  function close() {
    if (closed.current) return;
    closed.current = true;
    controller.current?.abort();
    setOpen(false);
    if (onDismiss) {
      onDismiss();
      window.requestAnimationFrame(() => previousFocus.current?.focus({ preventScroll: true }));
    } else {
      navigate('/dashboard');
      restoreDashboardFocus();
    }
  }
  function selectFile(selected?: File) {
    setError(''); setParseErrors([]);
    if (!selected || disabled) return;
    setFile(null);
    if (!selected.name.toLowerCase().endsWith('.csv')) { setError('Escolha um arquivo com a extensão .csv.'); return; }
    if (config.data && selected.size > config.data.upload.tamanho_maximo_bytes) { setError(`O arquivo deve ter até ${Math.ceil(config.data.upload.tamanho_maximo_bytes / 1024 / 1024).toLocaleString('pt-BR')} MB. Divida suas notas em lotes menores.`); return; }
    if (!selected.size) { setError('Este arquivo está vazio. Inclua suas notas ou use o lote de exemplo.'); return; }
    setFile(selected);
  }
  async function transfer(selected: File, lotName: string, signal: AbortSignal, idempotencyKey?: string) {
    setPreparingExample(false);
    const result = await uploadLote(selected, dataBase, value => { if (mounted.current) setProgress(value); }, signal, idempotencyKey);
    if (!mounted.current || closed.current) return;
    try { registrarLote(result.lote_id, selected.name, lotName); } catch { /* A falta de espaço local não impede a consulta do lote na API. */ }
    setAnalysis({ id: result.lote_id, replay: result.replay });
  }
  function reportUploadError(cause: unknown) {
    if (!mounted.current || closed.current || cause instanceof Error && cause.name === 'AbortError') return;
    setError(errorMessage(cause));
    if (cause instanceof ApiError) setParseErrors(cause.erro.detalhes ?? []);
  }
  function finish() {
    inFlight.current = false;
    if (!mounted.current || closed.current) return;
    setUploading(false); setPreparingExample(false);
  }
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (inFlight.current || !config.data) return;
    setError(''); setParseErrors([]);
    nameInput.current?.setCustomValidity(name.trim() ? '' : 'Informe um nome para este lote.');
    if (!form.current?.reportValidity()) return;
    if (!file) { setError('Escolha um arquivo CSV ou arraste-o para a área de envio.'); dropButton.current?.focus(); return; }
    inFlight.current = true; setUploading(true); setProgress(null);
    const currentController = new AbortController(); controller.current = currentController;
    try { await transfer(file, name.trim(), currentController.signal); }
    catch (cause) { reportUploadError(cause); }
    finally { finish(); }
  }
  async function useExample() {
    if (inFlight.current || !config.data || !dateInput.current?.reportValidity()) return;
    const lotName = name.trim() || 'Lote de demonstração';
    setName(lotName); nameInput.current?.setCustomValidity('');
    setError(''); setParseErrors([]); setProgress(null);
    inFlight.current = true; setUploading(true); setPreparingExample(true);
    const currentController = new AbortController(); controller.current = currentController;
    try {
      const response = await fetch(MOCK_MODE ? `/demo/exemplo.csv?data_base=${encodeURIComponent(dataBase)}` : '/exemplo.csv', { signal: currentController.signal });
      if (!response.ok) throw new Error('Não foi possível carregar o exemplo. Tente novamente.');
      const example = new File([await response.blob()], 'exemplo-demonstracao.csv', { type: 'text/csv' });
      if (!mounted.current || closed.current) return;
      setFile(example);
      if (MOCK_MODE && exampleKey.current?.date !== dataBase) exampleKey.current = { date: dataBase, key: crypto.randomUUID() };
      await transfer(example, lotName, currentController.signal, MOCK_MODE ? exampleKey.current?.key : undefined);
    } catch (cause) { reportUploadError(cause); }
    finally { finish(); }
  }

  return <Dialog.Root open={open} onOpenChange={value => { if (!value) close(); }}><Dialog.Portal><Dialog.Overlay className="new-lot-overlay new-lot-viewport"><Dialog.Content className={cx('new-lot-dialog', guideOpen && !busy && 'new-lot-expanded')} onOpenAutoFocus={event => { previousFocus.current = document.activeElement instanceof HTMLElement ? document.activeElement : null; event.preventDefault(); title.current?.focus({ preventScroll: true }); }} onCloseAutoFocus={event => { event.preventDefault(); if (closed.current) restoreDashboardFocus(); }}>
    <div className="new-lot-close-row"><Dialog.Close className="new-lot-close" aria-label="Fechar novo lote"><img src="/figma/new-lot-close.svg" alt="" /></Dialog.Close></div>
    {busy ? <DocumentLoadingPanel lot={analysisQuery.data} phase={analysis ? 'analyzing' : preparingExample ? 'preparing' : 'uploading'} uploadProgress={progress} error={analysisQuery.error ? errorMessage(analysisQuery.error) : undefined} onRetry={analysisQuery.error ? () => { void analysisQuery.refetch(); } : undefined} onCancel={close} /> : <div className="new-lot-panels"><form ref={form} noValidate className="new-lot-form-panel" onSubmit={submit}>
      <div className="new-lot-heading"><div className="new-lot-heading-copy"><Dialog.Title ref={title} tabIndex={-1}>Novo lote</Dialog.Title><Dialog.Description>Envie o CSV e veja na hora quanto você recebe e por que alguma nota não entrou.</Dialog.Description></div><button className="new-lot-info" type="button" aria-label={guideOpen ? 'Recolher como funciona' : 'Como funciona'} aria-expanded={guideOpen} aria-controls="new-lot-onboarding" onClick={() => setGuideOpen(previous => !previous)}><img src={guideOpen ? '/figma/new-lot-collapse.svg' : '/figma/new-lot-info.svg'} alt="" /></button></div>
      <div className="new-lot-fields">
        <label className="sr-only" htmlFor="new-lot-name">Nome do lote</label><input ref={nameInput} id="new-lot-name" className="new-lot-input" placeholder="Nome do lote*" type="text" value={name} required maxLength={80} autoComplete="off" disabled={busy} onChange={event => { event.currentTarget.setCustomValidity(''); setName(event.target.value); }} />
        <div className="new-lot-date"><label className="sr-only" htmlFor="new-lot-date">Data-base</label><input ref={dateInput} id="new-lot-date" className="new-lot-input" type="date" value={dataBase} required disabled={disabled} onInput={event => setDate(event.currentTarget.value)} onChange={event => setDate(event.target.value)} /><img src="/figma/new-lot-calendar.svg" alt="" /></div>
        <input ref={fileInput} className="sr-only" type="file" accept=".csv,text/csv" tabIndex={-1} aria-label="Escolher arquivo CSV" disabled={disabled} onChange={event => { selectFile(event.target.files?.[0]); event.target.value = ''; }} />
        <div className={cx('new-lot-dropzone', dragging && 'new-lot-dragging', !!file && 'new-lot-selected')} onDragOver={event => { event.preventDefault(); if (!disabled) setDragging(true); }} onDragLeave={event => { event.preventDefault(); setDragging(false); }} onDrop={event => { event.preventDefault(); setDragging(false); if (disabled) return; if (event.dataTransfer.files.length > 1) { setError('Envie um arquivo CSV por vez.'); return; } selectFile(event.dataTransfer.files[0]); }}>
          <button ref={dropButton} className="new-lot-drop-target" type="button" disabled={!config.data} onClick={() => fileInput.current?.click()} aria-label={file ? `Trocar arquivo ${file.name}` : 'Escolher arquivo CSV'}><span className="new-lot-cloud"><img src="/figma/new-lot-cloud.svg" alt="" /></span>{file ? <span className="new-lot-file"><strong title={file.name}>{file.name}</strong><small>{Math.max(1, Math.ceil(file.size / 1024)).toLocaleString('pt-BR')} KB · Clique para trocar</small></span> : <span className="new-lot-drop-copy">Arraste o CSV aqui ou clique<br />para escolher um arquivo.</span>}</button>{file ? <button className="new-lot-remove-file" type="button" aria-label="Remover arquivo" onClick={() => { setFile(null); setError(''); setParseErrors([]); }}><X size={16} /></button> : null}
        </div>
        {config.error ? <div className="new-lot-feedback"><Feedback><strong>Não foi possível carregar as configurações</strong>{errorMessage(config.error)}<div><button type="button" className="new-lot-retry" onClick={() => config.refetch()}>Tentar novamente</button></div></Feedback></div> : null}
        {error ? <div className="new-lot-feedback"><Feedback>{error}</Feedback></div> : null}
        {parseErrors.length ? <div className="new-lot-error-table" tabIndex={0} aria-label="Ajustes necessários no CSV"><table><thead><tr><th scope="col">Linha</th><th scope="col">Campo</th><th scope="col">Como corrigir</th></tr></thead><tbody>{parseErrors.map((item, index) => <tr key={`${item.linha}-${item.campo}-${index}`}><td>{item.linha ?? '—'}</td><td>{item.campo}</td><td>{item.problema}</td></tr>)}</tbody></table></div> : null}
      </div>
      <div className="new-lot-actions"><button className="new-lot-submit" type="submit" disabled={disabled}>Enviar lote<img src="/figma/new-lot-sparkle.svg" alt="" /></button><button className="new-lot-example" type="button" disabled={disabled} onClick={useExample}>Usar lote de exemplo</button></div>
    </form>
    {guideOpen ? <aside className="new-lot-onboarding" id="new-lot-onboarding" aria-labelledby="new-lot-onboarding-title"><h2 id="new-lot-onboarding-title">Como funciona</h2><ol><li><div className="new-lot-step-rail"><span>01</span><i /></div><div className="new-lot-step-copy"><h3>Envie suas notas</h3><p>Importe o arquivo CSV com suas notas fiscais de forma rápida e segura.</p></div></li><li><div className="new-lot-step-rail"><span>02</span><i /></div><div className="new-lot-step-copy"><h3>Entenda o resultado</h3><p>Veja na hora quanto você recebe por cada nota e por que alguma não entrou</p></div></li><li><div className="new-lot-step-rail"><span>03</span></div><div className="new-lot-step-copy"><h3>Confirme a cessão</h3><p>Revise os detalhes e, quando tudo estiver certo, confirme a cessão das notas</p></div></li></ol></aside> : null}
    </div>}
  </Dialog.Content></Dialog.Overlay></Dialog.Portal></Dialog.Root>;
}
