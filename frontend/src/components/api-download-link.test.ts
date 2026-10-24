import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { downloadApiBlob } from './api-download-link';

const fetchMock = vi.fn();
const anchor = { href: '', download: '', hidden: false, click: vi.fn(), remove: vi.fn() };
const appendChild = vi.fn();

beforeEach(() => {
  vi.useFakeTimers();
  vi.stubGlobal('fetch', fetchMock);
  vi.stubGlobal('document', { createElement: vi.fn(() => anchor), body: { appendChild } });
  vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:arquivo-api');
  vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => undefined);
  fetchMock.mockReset(); appendChild.mockClear(); anchor.click.mockClear(); anchor.remove.mockClear();
  anchor.href = ''; anchor.download = ''; anchor.hidden = false;
});

afterEach(() => { vi.clearAllTimers(); vi.useRealTimers(); vi.restoreAllMocks(); vi.unstubAllGlobals(); });

describe('download de CSV por endpoint interceptável', () => {
  it('preserva URL, filtros e bytes do Blob e usa o nome recebido do servidor', async () => {
    const bytes = new Uint8Array([0xef, 0xbb, 0xbf, 0x61, 0x3b, 0xc3, 0xa7, 0x0d, 0x0a]);
    fetchMock.mockResolvedValue(new Response(bytes, { headers: { 'Content-Type': 'text/csv;charset=utf-8', 'Content-Disposition': "attachment; filename*=UTF-8''notas%20recusadas.csv" } }));
    const href = '/api/lotes/id/notas.csv?status=recusada&motivo=PRAZO_MAXIMO&busca=loja';
    await downloadApiBlob(href);
    expect(fetchMock).toHaveBeenCalledWith(href, { cache: 'no-store', signal: undefined });
    const blob = vi.mocked(URL.createObjectURL).mock.calls[0][0] as Blob;
    expect(new Uint8Array(await blob.arrayBuffer())).toEqual(bytes);
    expect(anchor.href).toBe('blob:arquivo-api');
    expect(anchor.download).toBe('notas recusadas.csv');
    expect(appendChild).toHaveBeenCalledWith(anchor); expect(anchor.click).toHaveBeenCalledOnce(); expect(anchor.remove).toHaveBeenCalledOnce();
    expect(URL.revokeObjectURL).not.toHaveBeenCalled();
    vi.advanceTimersByTime(29_999); expect(URL.revokeObjectURL).not.toHaveBeenCalled();
    vi.advanceTimersByTime(1); expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:arquivo-api');
  });

  it('usa o nome do endpoint quando não há Content-Disposition, sem inventar conteúdo', async () => {
    fetchMock.mockResolvedValue(new Response('STATUS;MOTIVO\r\n', { headers: { 'Content-Type': 'text/csv' } }));
    await downloadApiBlob('/api/lotes/id/notas.csv?status=aprovada');
    expect(anchor.download).toBe('notas.csv');
    expect(fetchMock).toHaveBeenCalledOnce();
  });

  it('preserva o erro do backend e não baixa sua resposta de erro como CSV', async () => {
    fetchMock.mockResolvedValue(Response.json({ erro: { codigo: 'LOTE_NAO_CONCLUIDO', mensagem: 'O lote ainda está sendo processado.' } }, { status: 409 }));
    await expect(downloadApiBlob('/api/lotes/id/notas.csv')).rejects.toThrow('O lote ainda está sendo processado.');
    expect(URL.createObjectURL).not.toHaveBeenCalled(); expect(anchor.click).not.toHaveBeenCalled();
  });

  it('informa a falha de conexão sem recorrer a dados fictícios alternativos', async () => {
    fetchMock.mockRejectedValue(new TypeError('Failed to fetch'));
    await expect(downloadApiBlob('/api/lotes/id/notas.csv')).rejects.toThrow('Não foi possível baixar o arquivo. Verifique a conexão e tente novamente.');
    expect(fetchMock).toHaveBeenCalledOnce(); expect(URL.createObjectURL).not.toHaveBeenCalled();
  });
});
