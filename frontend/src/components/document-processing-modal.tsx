import * as Dialog from '@radix-ui/react-dialog';
import { useNavigate } from 'react-router-dom';
import type { Lote } from '@/api/types';
import { DashboardPage } from './dashboard-page';
import { DocumentLoadingPanel } from './document-loading';
import './new-lot.css';

export function DocumentProcessingModal({ lot, error, onRetry }: { lot: Lote; error?: string; onRetry?: () => void }) {
  const navigate = useNavigate();
  function close() {
    navigate('/dashboard');
    window.requestAnimationFrame(() => document.querySelector<HTMLAnchorElement>('.dashboard-upload-link')?.focus({ preventScroll: true }));
  }
  return <><DashboardPage /><Dialog.Root open onOpenChange={open => { if (!open) close(); }}><Dialog.Portal><Dialog.Overlay className="new-lot-overlay new-lot-viewport"><Dialog.Content className="new-lot-dialog" onCloseAutoFocus={event => event.preventDefault()}>
    <div className="new-lot-close-row"><Dialog.Close className="new-lot-close" aria-label="Fechar análise do documento"><img src="/figma/new-lot-close.svg" alt="" /></Dialog.Close></div>
    <DocumentLoadingPanel showMedia={false} lot={lot} phase="analyzing" error={error} onRetry={onRetry} onCancel={close} />
  </Dialog.Content></Dialog.Overlay></Dialog.Portal></Dialog.Root></>;
}
