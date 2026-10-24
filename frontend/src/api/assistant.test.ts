import { afterEach, describe, expect, it, vi } from 'vitest';
import { askAssistant, assistantStatus } from './assistant';
afterEach(() => vi.unstubAllGlobals());
describe('contrato do assistente', () => {
  it('respeita disponibilidade e indisponibilidade', async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(Response.json({ disponivel: true })).mockResolvedValueOnce(Response.json({ disponivel: false }));
    vi.stubGlobal('fetch', fetchMock);
    expect(await assistantStatus()).toBe(true); expect(await assistantStatus()).toBe(false);
    expect(fetchMock.mock.calls[0][0]).toBe('/ai/assistente/status');
  });
  it('envia sessão e contexto sem credenciais no navegador', async () => {
    const fetchMock = vi.fn().mockResolvedValue(Response.json({ status: 'COMPLETED', content: 'Resultado do lote' })); vi.stubGlobal('fetch', fetchMock);
    expect(await askAssistant('Como foi?', 'sessao-1', 'lote-1')).toBe('Resultado do lote');
    const [url, options] = fetchMock.mock.calls[0];
    expect(url).toBe('/ai/agents/antecipe-assistente/runs');
    expect(options.body.get('message')).toBe('Como foi?'); expect(options.body.get('session_id')).toBe('sessao-1'); expect(options.body.get('stream')).toBe('false'); expect(JSON.parse(options.body.get('dependencies'))).toEqual({ lote_id: 'lote-1' }); expect(options.headers).toBeUndefined();
  });
  it('não expõe erro técnico devolvido com HTTP 200', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(Response.json({ status: 'ERROR', content: 'secret technical model details' })));
    await expect(askAssistant('Oi', 'sessao')).rejects.toThrow('Não consegui responder agora. Tente novamente.');
  });
  it('trata falha de conexão HTTP e resposta vazia', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValueOnce(new Response(null, { status: 503 })).mockResolvedValueOnce(Response.json({ status: 'COMPLETED', content: '' })));
    await expect(assistantStatus()).rejects.toThrow('indisponível'); await expect(askAssistant('Oi', 'sessao')).rejects.toThrow('Não consegui responder');
  });
});

describe('streaming AgentOS', () => {
  function streamed(chunks: string[]) {
    const encoder = new TextEncoder();
    return new Response(new ReadableStream({ start(controller) { for (const chunk of chunks) controller.enqueue(encoder.encode(chunk)); controller.close(); } }), { headers: { 'Content-Type': 'text/event-stream' } });
  }
  it('monta conteúdo separado em chunks e conserva sessão e lote', async () => {
    const fetchMock = vi.fn().mockResolvedValue(streamed(['event: RunContent\r', '\ndata: {"content":"Olá "}\r\n\r', '\nevent: RunContent\ndata: {"content":"mundo"}\n\nevent: RunCompleted\ndata: {}\n\n']));
    vi.stubGlobal('fetch', fetchMock); const progress = vi.fn();
    expect(await askAssistant('Oi', 'sessao-2', 'lote-2', undefined, progress)).toBe('Olá mundo');
    expect(progress.mock.calls.map(call => call[0])).toEqual(['Olá ', 'Olá mundo']);
    expect(fetchMock.mock.calls[0][1].body.get('stream')).toBe('true');
  });
  it('recusa erro do modelo depois de conteúdo parcial sem expor detalhes', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(streamed(['event: RunContent\ndata: {"content":"Parcial"}\n\nevent: RunError\ndata: {"content":"private model error"}\n\n'])));
    await expect(askAssistant('Oi', 's', undefined, undefined, vi.fn())).rejects.toThrow('Não consegui responder agora. Tente novamente.');
  });
  it('não aceita conexão interrompida como resposta concluída', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(streamed(['event: RunContent\ndata: {"content":"Parcial"}\n\n'])));
    await expect(askAssistant('Oi', 's', undefined, undefined, vi.fn())).rejects.toThrow('Não consegui responder');
  });
});
