import * as Dialog from '@radix-ui/react-dialog';
import { ArrowRight, ShieldCheck } from 'lucide-react';
import type { Lote } from '@/api/types';
import { dateBR, moneyCentavos } from '@/lib/format';
import { DocumentLoadingPanel } from './document-loading';
import { Feedback } from './feedback';
import './new-lot.css';
import './cession-modal.css';

export function CessionModal({ open, onClose, onConfirm, pending, error, lot }: { open: boolean; onClose: () => void; onConfirm: () => void; pending: boolean; error?: string; lot: Lote }) {
  return <Dialog.Root open={open} onOpenChange={value => { if (!value && !pending) onClose(); }}><Dialog.Portal><Dialog.Overlay className="new-lot-overlay new-lot-viewport"><Dialog.Content className="new-lot-dialog cession-dialog" onEscapeKeyDown={event => { if (pending) event.preventDefault(); }} onPointerDownOutside={event => event.preventDefault()} onInteractOutside={event => event.preventDefault()}>
    {!pending ? <div className="new-lot-close-row"><Dialog.Close className="new-lot-close" aria-label="Fechar confirmação"><img src="/figma/new-lot-close.svg" alt="" /></Dialog.Close></div> : null}
    {pending ? <DocumentLoadingPanel phase="confirming" /> : <div className="new-lot-form-panel cession-review"><header className="new-lot-heading"><div className="new-lot-heading-copy"><Dialog.Title>Confirmar cessão</Dialog.Title><Dialog.Description>Revise os valores antes de enviar. Apenas notas aprovadas e disponíveis serão incluídas.</Dialog.Description></div></header><div className="cession-value"><span>Você recebe</span><strong>{moneyCentavos(lot.totais?.liquido_centavos ?? null)}</strong><p>Valor líquido simulado</p></div><dl className="cession-breakdown"><div><dt>Notas aprovadas</dt><dd>{lot.totais?.aprovadas.toLocaleString('pt-BR')} {lot.totais?.aprovadas === 1 ? 'nota' : 'notas'}</dd></div><div><dt>Valor de face</dt><dd>{moneyCentavos(lot.totais?.face_centavos ?? null)}</dd></div><div><dt>Desconto total</dt><dd>{moneyCentavos(lot.totais?.desconto_centavos ?? null)}</dd></div></dl><div className="cession-advice"><ShieldCheck size={20} aria-hidden="true" /><p>O valor final pode mudar se uma nota já tiver sido vendida ou os limites do fundo mudarem durante a revisão.</p></div><p className="cession-meta">Regras versão {lot.regras_versao} · Data-base {dateBR(lot.data_base)}</p>{error ? <Feedback>{error}</Feedback> : null}<div className="cession-actions"><button className="new-lot-submit" onClick={onConfirm}>Confirmar e enviar<ArrowRight size={18} /></button><button className="new-lot-example" onClick={onClose}>Voltar à revisão</button></div></div>}
  </Dialog.Content></Dialog.Overlay></Dialog.Portal></Dialog.Root>;
}
