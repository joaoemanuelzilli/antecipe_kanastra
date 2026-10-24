export const ASSISTANT_UNAVAILABLE = 'O assistente está indisponível agora. Tente novamente em instantes.';
const RESPONSE_ERROR = 'Não consegui responder agora. Tente novamente.';
export async function assistantStatus(signal?: AbortSignal): Promise<boolean> {
  const response = await fetch('/ai/assistente/status', { signal: AbortSignal.any([AbortSignal.timeout(10000), ...(signal ? [signal] : [])]) });
  if (!response.ok) throw new Error(ASSISTANT_UNAVAILABLE);
  try { return (await response.json()).disponivel === true; }
  catch { throw new Error(ASSISTANT_UNAVAILABLE); }
}

/** Parses AgentOS SSE across arbitrary network chunk boundaries, including CRLF. */
export async function readAssistantStream(response: Response, onContent: (text: string) => void): Promise<string> {
  if (!response.body) throw new Error(RESPONSE_ERROR);
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '', answer = '', completed = false;
  function consume(block: string) {
    const lines = block.split(/\r?\n/);
    const raw = lines.filter(line => line.startsWith('data:')).map(line => line.slice(5).trimStart()).join('\n');
    if (!raw || raw === '[DONE]') return;
    let data: Record<string, unknown>;
    try { data = JSON.parse(raw); } catch { throw new Error(RESPONSE_ERROR); }
    const event = lines.find(line => line.startsWith('event:'))?.slice(6).trim() || data.event;
    if (event === 'RunError' || data.status === 'ERROR') throw new Error(RESPONSE_ERROR);
    if (event === 'RunContent' && typeof data.content === 'string') { answer += data.content; onContent(answer); }
    if (event === 'RunCompleted') {
      if (!answer && typeof data.content === 'string') { answer = data.content; onContent(answer); }
      completed = true;
    }
  }
  try {
    while (!completed) {
      const { value, done } = await reader.read();
      buffer += decoder.decode(value, { stream: !done });
      let separator: RegExpExecArray | null;
      while ((separator = /\r?\n\r?\n/.exec(buffer))) {
        const block = buffer.slice(0, separator.index);
        buffer = buffer.slice(separator.index + separator[0].length);
        consume(block);
        if (completed) break;
      }
      if (done) { if (buffer.trim() && !completed) consume(buffer); break; }
    }
    if (!completed || !answer.trim()) throw new Error(RESPONSE_ERROR);
    return answer;
  } finally { await reader.cancel().catch(() => {}); reader.releaseLock(); }
}

export async function askAssistant(question: string, sessionId: string, lotId?: string, signal?: AbortSignal, onContent?: (text: string) => void): Promise<string> {
  const form = new FormData();
  form.append('message', question); form.append('session_id', sessionId); form.append('stream', onContent ? 'true' : 'false');
  if (lotId) form.append('dependencies', JSON.stringify({ lote_id: lotId }));
  const response = await fetch('/ai/agents/antecipe-assistente/runs', { method: 'POST', body: form, signal: AbortSignal.any([AbortSignal.timeout(90000), ...(signal ? [signal] : [])]) });
  if (!response.ok) throw new Error(RESPONSE_ERROR);
  if (response.headers.get('Content-Type')?.includes('text/event-stream')) return readAssistantStream(response, onContent ?? (() => {}));
  let data: { status?: string; content?: unknown };
  try { data = await response.json(); } catch { throw new Error(RESPONSE_ERROR); }
  if (data.status === 'ERROR' || typeof data.content !== 'string' || !data.content.trim()) throw new Error(RESPONSE_ERROR);
  onContent?.(data.content);
  return data.content;
}
