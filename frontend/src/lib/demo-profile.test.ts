import { afterEach, describe, expect, it, vi } from 'vitest';
import { DEFAULT_DEMO_PROFILE, parseDemoProfile, saveDemoProfile } from './demo-profile';

afterEach(() => vi.unstubAllGlobals());
describe('perfil de demonstração', () => {
  it('recupera os padrões quando o armazenamento está ausente ou inválido', () => {
    for (const value of [null, '', '{', '[]', '42']) expect(parseDemoProfile(value)).toEqual(DEFAULT_DEMO_PROFILE);
  });
  it('aceita somente campos conhecidos e preferências booleanas', () => {
    const result = parseDemoProfile(JSON.stringify({ name: 'Nome de teste', email: 42, confirmLogout: false, reducedMotion: 'false', password: 'nao-guardar', token: 'nao-guardar' }));
    expect(result.name).toBe('Nome de teste'); expect(result.email).toBe(DEFAULT_DEMO_PROFILE.email); expect(result.confirmLogout).toBe(false); expect(result.reducedMotion).toBe(false); expect(result).not.toHaveProperty('password'); expect(result).not.toHaveProperty('token');
  });
  it('nunca salva senhas ou credenciais extras recebidas por engano', () => {
    const setItem = vi.fn(); const dispatchEvent = vi.fn();
    vi.stubGlobal('sessionStorage', { setItem }); vi.stubGlobal('window', { dispatchEvent });
    expect(saveDemoProfile({ ...DEFAULT_DEMO_PROFILE, password: 'nao-guardar', gestor_senha: 'nao-guardar' } as typeof DEFAULT_DEMO_PROFILE)).toBe(true);
    expect(JSON.parse(setItem.mock.calls[0][1])).toEqual(DEFAULT_DEMO_PROFILE); expect(dispatchEvent).toHaveBeenCalledOnce();
  });
  it('informa falha sem prometer salvamento quando o armazenamento está bloqueado', () => {
    const dispatchEvent = vi.fn(); vi.stubGlobal('window', { dispatchEvent }); vi.stubGlobal('sessionStorage', { setItem: () => { throw new Error('blocked'); } });
    expect(saveDemoProfile(DEFAULT_DEMO_PROFILE)).toBe(false); expect(dispatchEvent).not.toHaveBeenCalled();
  });
});
