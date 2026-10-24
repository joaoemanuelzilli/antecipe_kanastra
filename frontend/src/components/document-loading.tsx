import { useEffect, useRef, useState } from 'react';
import { LoaderCircle } from 'lucide-react';
import * as Dialog from '@radix-ui/react-dialog';
import type { Lote } from '@/api/types';
import { useDemoProfile } from '@/lib/demo-profile';
import './document-loading.css';

export interface DocumentLoadingPanelProps {
  lot?: Lote;
  phase?: 'preparing' | 'uploading' | 'analyzing' | 'confirming';
  uploadProgress?: number | null;
  error?: string;
  onRetry?: () => void;
  onCancel?: () => void;
  showMedia?: boolean;
}

const VIDEO = '/figma/document-review.mp4';
const POSTER = '/figma/document-review-poster.jpg';
const clampPercent = (value: number) => Math.min(100, Math.max(0, value));

export function DocumentLoadingPanel({ lot, phase = lot ? 'analyzing' : 'preparing', uploadProgress, error, onRetry, onCancel, showMedia = true }: DocumentLoadingPanelProps) {
  const title = useRef<HTMLHeadingElement>(null);
  const video = useRef<HTMLVideoElement>(null);
  const [reducedMotion, setReducedMotion] = useState(() => typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches);
  const [visible, setVisible] = useState(() => typeof document === 'undefined' || document.visibilityState === 'visible');
  const [videoFailed, setVideoFailed] = useState(false);
  const { reducedMotion: userReducedMotion } = useDemoProfile();
  const mayAnimate = !reducedMotion && !userReducedMotion && visible && !error;

  useEffect(() => { title.current?.focus({ preventScroll: true }); }, []);
  useEffect(() => {
    const preference = window.matchMedia('(prefers-reduced-motion: reduce)');
    const updatePreference = (event: MediaQueryListEvent) => setReducedMotion(event.matches);
    const updateVisibility = () => setVisible(document.visibilityState === 'visible');
    preference.addEventListener('change', updatePreference);
    document.addEventListener('visibilitychange', updateVisibility);
    return () => { preference.removeEventListener('change', updatePreference); document.removeEventListener('visibilitychange', updateVisibility); };
  }, []);
  useEffect(() => {
    const element = video.current;
    if (!element) return;
    if (mayAnimate) void element.play().catch(() => { /* O poster permanece disponível quando o navegador bloqueia autoplay. */ });
    else element.pause();
    return () => element.pause();
  }, [mayAnimate, reducedMotion, videoFailed]);

  const knownUpload = phase === 'uploading' && typeof uploadProgress === 'number' && Number.isFinite(uploadProgress);
  const knownAnalysis = phase === 'analyzing' && lot?.status === 'processando' && lot.progresso.linhas_total > 0 && Number.isFinite(lot.progresso.percentual);
  const percent = phase === 'uploading' ? knownUpload ? clampPercent(uploadProgress) : 0 : knownAnalysis ? clampPercent(lot.progresso.percentual) : null;
  const currentStep = lot?.etapas.find(step => step.codigo === lot.etapa || step.status === 'em_andamento');
  const label = error ? 'Aguardando atualização...' : phase === 'confirming' ? 'Confirmando sua cessão…' : phase === 'preparing' ? 'Preparando...' : phase === 'uploading' ? 'Enviando...' : lot?.status === 'na_fila' ? 'Na fila...' : 'Analisando...';
  const announcement = error ? '' : phase === 'confirming' ? 'Registrando a cessão. Aguarde a confirmação do servidor.' : phase === 'analyzing' && lot?.status === 'na_fila' ? lot.posicao_fila === null ? 'Aguardando o início do processamento.' : `Seu lote é o ${lot.posicao_fila}º da fila.` : phase === 'analyzing' ? currentStep?.titulo || 'Analisando o arquivo.' : phase === 'uploading' ? 'Enviando o arquivo.' : 'Preparando o arquivo.';
  const hasLineCount = phase === 'analyzing' && !!lot && lot.progresso.linhas_total > 0;
  const counter = hasLineCount ? `${lot.progresso.linhas_processadas}/${lot.progresso.linhas_total}` : knownUpload ? `${Math.round(clampPercent(uploadProgress))}%` : null;
  const counterLabel = hasLineCount ? `${lot.progresso.linhas_processadas.toLocaleString('pt-BR')} de ${lot.progresso.linhas_total.toLocaleString('pt-BR')} linhas processadas` : knownUpload ? `${Math.round(clampPercent(uploadProgress))}% do arquivo enviado` : undefined;

  return <section className="document-loading-panel">
    <Dialog.Title ref={title} tabIndex={-1} className="sr-only">{phase === 'confirming' ? 'Confirmando sua cessão' : phase === 'uploading' ? 'Enviando seu arquivo' : phase === 'preparing' ? 'Preparando seu lote' : 'Analisando suas notas'}</Dialog.Title>
    <Dialog.Description className="sr-only">{phase === 'confirming' ? 'Aguarde enquanto registramos a operação. O resultado será exibido após a resposta do servidor.' : 'Acompanhe o andamento do lote. Um arquivo já enviado permanece disponível no histórico para consulta.'}</Dialog.Description>
    {showMedia ? <div className="document-loading-media" aria-hidden="true"><img className="document-loading-poster" src={POSTER} width="832" height="468" alt="" />{!reducedMotion && !userReducedMotion && !videoFailed ? <video ref={video} className="document-loading-video" src={VIDEO} width="832" height="468" autoPlay={mayAnimate} muted loop playsInline preload="metadata" poster={POSTER} aria-hidden="true" tabIndex={-1} disablePictureInPicture disableRemotePlayback onError={() => setVideoFailed(true)} /> : null}</div> : <div className="document-loading-simple-icon" aria-hidden="true"><LoaderCircle size={28} className="spin" /></div>}
    <div className="document-loading-status"><div className={`document-loading-track${percent === null ? ' is-indeterminate' : ''}${error ? ' has-error' : ''}`} role="progressbar" aria-label={phase === 'confirming' ? 'Progresso da confirmação' : phase === 'uploading' ? 'Progresso do envio' : 'Progresso da análise'} aria-valuemin={0} aria-valuemax={100} aria-valuenow={percent ?? undefined} aria-valuetext={percent === null ? 'Andamento ainda não informado' : `${percent.toLocaleString('pt-BR', { maximumFractionDigits: 1 })}%`}><span key={`${phase}:${lot?.id ?? 'new'}:${percent === null ? 'waiting' : 'measured'}`} style={{ width: `${percent ?? 0}%` }} /></div><div className="document-loading-status-row"><p>{percent === null && !error ? <LoaderCircle size={14} className="spin" aria-hidden="true" /> : null}{label}</p>{counter ? <span className="document-loading-counter" aria-label={counterLabel}>{counter}</span> : null}</div><p className="sr-only" role="status" aria-live="polite" aria-atomic="true">{announcement}</p>{error ? <div className="document-loading-error" role="alert"><p>{error}</p>{onRetry ? <button type="button" onClick={onRetry}>Tentar novamente</button> : null}</div> : null}</div>
    {onCancel ? <button className="document-loading-cancel" type="button" onClick={onCancel}>Cancelar</button> : <p className="document-loading-confirm-note">Aguarde. Você verá o resultado assim que a confirmação terminar.</p>}
  </section>;
}
