import { lazy, Suspense, useEffect, useRef, useState } from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import { ArrowUp, MessageCircle, RotateCcw, X } from 'lucide-react';
import { nomeLote } from '@/api/client';
import { useQuery } from '@tanstack/react-query';
import { assistantStatus, askAssistant } from '@/api/assistant';
import { usePathname } from '@/lib/router';
import './assistant-chat.css';

const Markdown = lazy(() => import('react-markdown'));

type Message = { id?: string; role: 'user' | 'assistant'; text: string };
export function AssistantChat({ open, onOpenChange, initialQuestion }: { open: boolean; onOpenChange: (open: boolean) => void; initialQuestion: string }) {
  const path = usePathname();
  const lotId = path.match(/^\/lote\/([^/]+)$/)?.[1];
  const availability = useQuery({ queryKey: ['assistant-status'], queryFn: ({ signal }) => assistantStatus(signal), enabled: open, retry: false, staleTime: 30000 });
  const available = availability.data === true && !availability.error;
  const [draft, setDraft] = useState(initialQuestion);
  const [messages, setMessages] = useState<Message[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const session = useRef(crypto.randomUUID());
  const request = useRef<AbortController | null>(null);
  const bottom = useRef<HTMLDivElement>(null);
  useEffect(() => () => request.current?.abort(), []);
  useEffect(() => { if (open) bottom.current?.scrollIntoView({ block: 'nearest' }); }, [messages, busy, open]);
  const suggestions = lotId ? ['Resuma o resultado do lote', 'Por que minhas notas foram recusadas?', 'Como corrigir o arquivo?'] : ['Como envio meu CSV?', 'Como funciona a cessão?', 'Como entender uma nota recusada?'];
  async function send(text = draft) {
    if (!text.trim() || busy || !available) return;
    const question = text.trim();
    const controller = new AbortController(); request.current = controller;
    const replyId = crypto.randomUUID();
    setMessages(previous => [...previous, { role: 'user', text: question }, { id: replyId, role: 'assistant', text: '' }]); setDraft(''); setBusy(true); setError('');
    try {
      const answer = await askAssistant(question, session.current, lotId, controller.signal, text => { if (!controller.signal.aborted) setMessages(previous => previous.map(message => message.id === replyId ? { ...message, text } : message)); });
      if (!controller.signal.aborted) setMessages(previous => previous.map(message => message.id === replyId ? { ...message, text: answer } : message));
    } catch (cause) { if (!controller.signal.aborted) { setMessages(previous => previous.filter(message => message.id !== replyId)); setError(cause instanceof Error && !(cause instanceof TypeError) ? cause.message : 'Não consegui responder agora. Confira sua conexão e tente novamente.'); setDraft(question); } }
    finally { if (!controller.signal.aborted) setBusy(false); }
  }
  return <Dialog.Root open={open} onOpenChange={onOpenChange} modal={false}><Dialog.Portal><Dialog.Content className="assistant-chat" onInteractOutside={event => event.preventDefault()}><header className="assistant-chat-header"><img src="/figma/assistant-head-poster.jpg" alt="" width="48" height="48" /><div><Dialog.Title>Assistente Antecipe</Dialog.Title><Dialog.Description>{availability.isFetching ? 'Conectando ao assistente…' : available ? 'Conectado ao serviço de IA' : 'Assistente indisponível'}</Dialog.Description></div><Dialog.Close className="icon-button" aria-label="Fechar chat"><X size={20} /></Dialog.Close></header><div className="assistant-chat-context"><MessageCircle size={16} /><span>{lotId ? nomeLote(lotId) : 'Ajuda com a antecipação'}</span><button aria-label="Iniciar nova conversa" disabled={busy || !messages.length} onClick={() => { setMessages([]); setError(''); setDraft(''); session.current = crypto.randomUUID(); }}><RotateCcw size={16} /></button></div><div className="assistant-chat-messages" role="log" aria-label="Conversa com o assistente" aria-live="polite">{!messages.length ? <div className="assistant-chat-welcome"><h3>Vamos entender suas notas?</h3><p>Posso ajudar você a encontrar o próximo passo. Escolha uma pergunta ou escreva a sua.</p><div className="assistant-suggestions">{suggestions.map(question => <button key={question} disabled={!available} onClick={() => void send(question)}>{question}<ArrowUp size={16} /></button>)}</div></div> : messages.filter(message => message.text).map((message, index) => <div key={index} className={`assistant-message ${message.role}`}><strong>{message.role === 'user' ? 'Você' : 'Antecipe'}</strong>{message.role === 'assistant' ? <div className="assistant-markdown"><Suspense fallback={<p>{message.text}</p>}><Markdown skipHtml>{message.text}</Markdown></Suspense></div> : <p>{message.text}</p>}</div>)}{busy ? <p className="assistant-chat-busy" role="status">{lotId ? 'Consultando seu lote…' : 'Preparando uma resposta…'}</p> : null}{error ? <div role="alert" className="assistant-chat-error">{error}<button onClick={() => void send()}>Tentar novamente</button></div> : null}{!available && !availability.isFetching ? <div className="assistant-chat-unavailable" role="status"><strong>Não foi possível conectar</strong><p>O serviço de IA precisa estar disponível para responder. Você pode continuar consultando os detalhes das notas.</p><button onClick={() => void availability.refetch()}>Tentar conectar novamente</button></div> : null}<div ref={bottom} /></div><form className="assistant-chat-form" onSubmit={event => { event.preventDefault(); void send(); }}><label className="sr-only" htmlFor="assistant-question">Sua pergunta</label><textarea id="assistant-question" placeholder="Escreva sua pergunta…" maxLength={2000} value={draft} onChange={event => setDraft(event.target.value)} onKeyDown={event => { if (event.key === 'Enter' && !event.shiftKey && !event.nativeEvent.isComposing) { event.preventDefault(); void send(); } }} rows={2} /><button type="submit" aria-label="Enviar pergunta" disabled={busy || !draft.trim() || !available}><ArrowUp size={22} /></button><p>O assistente explica. Você decide e confirma.</p></form></Dialog.Content></Dialog.Portal></Dialog.Root>;
}
