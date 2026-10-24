import { useQuery } from '@tanstack/react-query';
import { lotesLocais, MOCK_MODE, request } from './client';
import type { LoteLocal } from './types';
import { useAccount } from './auth';
/** Account summaries only: never fetches the notes to build a history. */
export function useLotHistory() {
  const account = useAccount();
  return useQuery({ queryKey: ['lot-history', MOCK_MODE ? 'demo' : account.data?.id], enabled: MOCK_MODE || !!account.data, staleTime: 0, queryFn: async ({ signal }): Promise<LoteLocal[]> => {
    if (MOCK_MODE) return lotesLocais();
    const data = await request<{ lotes: { id: string; criado_em: string }[] }>('/api/lotes', { signal });
    const names = lotesLocais();
    return data.lotes.map(lot => ({ id: lot.id, criado_em: lot.criado_em, arquivo: `Lote ${lot.id.slice(0, 8)}`, nome: names.find(local => local.id === lot.id)?.nome }));
  } });
}
