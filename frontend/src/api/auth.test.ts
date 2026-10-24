import { afterEach, describe, expect, it, vi } from 'vitest';
import { currentAccount, loginAccount, registerAccount, logoutAccount } from './auth';
afterEach(() => vi.unstubAllGlobals());
const account = { id: 'user', nome: 'Ana', email: 'ana@example.com', perfil: 'empresa', criado_em: '2026-10-24' };
describe('sessão por cookie do Go', () => {
  it('interpreta 401 como sessão ausente sem simular usuário', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ erro: { codigo: 'NAO_AUTENTICADO', mensagem: 'Entre' } }), { status: 401 })));
    expect(await currentAccount()).toBeNull();
  });
  it('preserva falha do serviço em vez de entrar como demo', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{}', { status: 503 })));
    await expect(currentAccount()).rejects.toThrow();
  });
  it('envia credenciais e usa somente usuario, sem persistir token', async () => {
    const fetch = vi.fn().mockResolvedValue(new Response(JSON.stringify({ usuario: account, token: 'ignorado' })));
    vi.stubGlobal('fetch', fetch);
    expect(await loginAccount('ana@example.com', 'inventada123')).toEqual(account);
    expect(fetch.mock.calls[0][1].credentials).toBe('include');
    expect(JSON.parse(fetch.mock.calls[0][1].body)).toEqual({ email: 'ana@example.com', senha: 'inventada123' });
  });
  it('cadastro segue campos snake_case e logout chama servidor', async () => {
    const fetch = vi.fn().mockResolvedValueOnce(new Response(JSON.stringify({ usuario: account }))).mockResolvedValueOnce(new Response(null, { status: 204 }));
    vi.stubGlobal('fetch', fetch);
    const input = { nome: 'Ana', email: 'ana@example.com', senha: 'inventada123', empresa_nome: 'Empresa', empresa_cnpj: '10.062.050/0001-57' };
    await registerAccount(input); await logoutAccount();
    expect(JSON.parse(fetch.mock.calls[0][1].body)).toEqual(input);
    expect(fetch.mock.calls[1][0]).toContain('/api/auth/logout');
    expect(fetch.mock.calls[1][1].method).toBe('POST');
  });
});
