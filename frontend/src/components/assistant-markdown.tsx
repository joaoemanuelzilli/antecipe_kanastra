import { Children, isValidElement, useEffect, useRef, useState, type ReactNode } from 'react';
import { parseAssistantChart } from '@/lib/assistant-chart';
import { AssistantChart } from './assistant-chart';
import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Check, Copy } from 'lucide-react';

function plainText(value: ReactNode): string {
  return Children.toArray(value).map(child => {
    if (typeof child === 'string' || typeof child === 'number') return String(child);
    return isValidElement<{ children?: ReactNode }>(child) ? plainText(child.props.children) : '';
  }).join('');
}

function CodeBlock({ children, lotId }: { children?: ReactNode; lotId?: string }) {
  const [copied, setCopied] = useState(false);
  const [failed, setFailed] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  useEffect(() => () => clearTimeout(timer.current), []);
  const code = Children.toArray(children).find(child => isValidElement(child));
  const language = isValidElement<{ className?: string }>(code) ? code.props.className?.replace('language-', '') : undefined;
  async function copy() {
    try {
      await navigator.clipboard.writeText(plainText(children).replace(/\n$/, ''));
      setCopied(true); setFailed(false);
      clearTimeout(timer.current);
      timer.current = setTimeout(() => setCopied(false), 2000);
    } catch { setFailed(true); }
  }
  if (language === 'antecipe-chart') {
    const spec = parseAssistantChart(plainText(children));
    if (spec) return <AssistantChart spec={spec} contextLotId={lotId} />;
  }
  return <div className="assistant-code-block"><div className="assistant-code-heading"><span>{language || 'Código'}</span><button type="button" onClick={() => { void copy(); }} aria-label={copied ? 'Código copiado' : 'Copiar código'}>{copied ? <Check size={14} /> : <Copy size={14} />}{copied ? 'Copiado' : 'Copiar'}</button></div><pre tabIndex={0} aria-label="Trecho de código">{children}</pre>{failed && <p role="status" className="assistant-copy-error">Selecione o trecho para copiar manualmente</p>}</div>;
}

export default function AssistantMarkdown({ children, lotId }: { children: string; lotId?: string }) {
  return <Markdown skipHtml remarkPlugins={[remarkGfm]} components={{
    h1: ({ children }) => <h3 className="assistant-answer-title">{children}</h3>,
    h2: ({ children }) => <h3>{children}</h3>,
    h3: ({ children }) => <h4>{children}</h4>,
    h4: ({ children }) => <h4>{children}</h4>,
    h5: ({ children }) => <h4>{children}</h4>,
    h6: ({ children }) => <h4>{children}</h4>,
    pre: ({ children }) => <CodeBlock lotId={lotId}>{children}</CodeBlock>,
    table: ({ children }) => <div className="assistant-table-scroll" role="region" aria-label="Tabela da resposta, role para ver todas as colunas" tabIndex={0}><table>{children}</table></div>,
    a: ({ href, children }) => <a href={href} target={/^https?:\/\//i.test(href ?? '') ? '_blank' : undefined} rel="noopener noreferrer">{children}</a>,
  }}>{children}</Markdown>;
}
