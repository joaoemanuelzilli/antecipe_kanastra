import { forwardRef, useEffect, useRef, useState, type AnchorHTMLAttributes, type MouseEvent } from 'react';
import { MOCK_MODE } from '@/api/client';

function downloadFilename(response: Response, href: string): string {
  const disposition = response.headers.get('Content-Disposition') ?? '';
  const encoded = /filename\*\s*=\s*UTF-8''([^;]+)/i.exec(disposition)?.[1];
  if (encoded) {
    try { return decodeURIComponent(encoded.trim().replace(/^"|"$/g, '')).split(/[\\/]/).pop() ?? ''; }
    catch { /* The ordinary filename or endpoint remains available. */ }
  }
  const filename = /filename\s*=\s*(?:"([^"]+)"|([^;]+))/i.exec(disposition);
  const provided = (filename?.[1] ?? filename?.[2] ?? '').trim().split(/[\\/]/).pop();
  if (provided) return provided;
  return href.split('?')[0].split('/').pop() ?? '';
}

/** Downloads the endpoint's original bytes; it never rebuilds a CSV in the browser. */
export async function downloadApiBlob(href: string, signal?: AbortSignal): Promise<void> {
  let response: Response;
  try { response = await fetch(href, { cache: 'no-store', signal }); }
  catch (cause) {
    if (cause instanceof Error && cause.name === 'AbortError') throw cause;
    throw new Error('Não foi possível baixar o arquivo. Verifique a conexão e tente novamente.', { cause });
  }
  if (!response.ok) {
    const body = await response.json().catch(() => null) as { erro?: { mensagem?: unknown } } | null;
    throw new Error(typeof body?.erro?.mensagem === 'string' ? body.erro.mensagem : 'Não foi possível baixar o arquivo. Tente novamente.');
  }
  const blob = await response.blob();
  const objectURL = URL.createObjectURL(blob);
  try {
    const anchor = document.createElement('a');
    anchor.href = objectURL;
    anchor.download = downloadFilename(response, href);
    anchor.hidden = true;
    try { document.body.appendChild(anchor); anchor.click(); }
    finally { anchor.remove(); }
  } finally {
    // Browser download handoff may continue after click has returned.
    setTimeout(() => URL.revokeObjectURL(objectURL), 30_000);
  }
}

interface ApiDownloadLinkProps extends Omit<AnchorHTMLAttributes<HTMLAnchorElement>, 'href' | 'onClick'> {
  href: string;
  onDownloadError: (message: string) => void;
}

export const ApiDownloadLink = forwardRef<HTMLAnchorElement, ApiDownloadLinkProps>(function ApiDownloadLink({ href, onDownloadError, children, ...props }, ref) {
  const [busy, setBusy] = useState(false);
  const inFlight = useRef(false);
  const controller = useRef<AbortController | null>(null);
  useEffect(() => () => controller.current?.abort(), []);

  async function handleClick(event: MouseEvent<HTMLAnchorElement>) {
    if (!MOCK_MODE) return;
    event.preventDefault();
    if (inFlight.current) return;
    inFlight.current = true;
    setBusy(true); onDownloadError('');
    controller.current = new AbortController();
    try { await downloadApiBlob(href, controller.current.signal); }
    catch (cause) {
      if (!(cause instanceof Error && cause.name === 'AbortError')) onDownloadError(cause instanceof Error ? cause.message : 'Não foi possível baixar o arquivo. Tente novamente.');
    } finally {
      inFlight.current = false; controller.current = null; setBusy(false);
    }
  }

  return <a {...props} ref={ref} href={href} onClick={(event) => { void handleClick(event); }} aria-busy={busy || undefined} aria-disabled={busy || undefined}>{children}</a>;
});
