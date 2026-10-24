export const ASSISTANT_UNAVAILABLE = 'O assistente está indisponível agora. Tente novamente em instantes.';
export async function assistantStatus(signal?: AbortSignal): Promise<boolean> {
  const response = await fetch('/ai/assistente/status', { signal });
  if (!response.ok) throw new Error(ASSISTANT_UNAVAILABLE);
  const data = await response.json();
  return data.disponivel === true;
}
export async function askAssistant(question: string, sessionId: string, lotId?: string, signal?: AbortSignal): Promise<string> {
  const form = new FormData();
  form.append('message', question); form.append('session_id', sessionId); form.append('stream', 'false');
  if (lotId) form.append('dependencies', JSON.stringify({ lote_id: lotId }));
  const response = await fetch('/ai/agents/antecipe-assistente/runs', { method: 'POST', body: form, signal });
  if (!response.ok) throw new Error('Não consegui responder agora. Tente novamente.');
  const data = await response.json();
  if (data.status === 'ERROR' || typeof data.content !== 'string' || !data.content.trim()) throw new Error('Não consegui responder agora. Tente novamente.');
  return data.content;
}
