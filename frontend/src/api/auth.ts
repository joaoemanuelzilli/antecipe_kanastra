import { useQuery } from '@tanstack/react-query';
import { ApiError, jsonRequest, MOCK_MODE, request } from './client';
export interface Account { id: string; nome: string; email: string; perfil: 'empresa' | 'gestor'; empresa_nome?: string; empresa_cnpj?: string; criado_em: string }
export interface Registration { nome: string; email: string; senha: string; empresa_nome: string; empresa_cnpj: string }
export const ACCOUNT_QUERY = ['auth-user'] as const;
export async function currentAccount(signal?: AbortSignal): Promise<Account | null> {
  try { return (await request<{ usuario: Account }>('/api/auth/eu', { signal })).usuario; }
  catch (cause) { if (cause instanceof ApiError && cause.status === 401) return null; throw cause; }
}
export function useAccount() { return useQuery({ queryKey: ACCOUNT_QUERY, queryFn: ({ signal }) => currentAccount(signal), enabled: !MOCK_MODE, retry: false, staleTime: 30000, refetchOnWindowFocus: true }); }
export async function loginAccount(email: string, senha: string) { return (await jsonRequest<{ usuario: Account }>('/api/auth/login', { email, senha })).usuario; }
export async function registerAccount(input: Registration) { return (await jsonRequest<{ usuario: Account }>('/api/auth/cadastro', input)).usuario; }
export function logoutAccount() { return request<void>('/api/auth/logout', { method: 'POST' }); }
