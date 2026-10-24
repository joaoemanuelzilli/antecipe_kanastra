import { LoaderCircle, NotepadText, CircleCheck, CircleX } from 'lucide-react';
import type { Lote, LoteLocal } from '@/api/types';
import Link from '@/lib/link';
import { countBR, moneyCentavos } from '@/lib/format';
import './lot-card.css';

export interface LotCardProps {
  local: LoteLocal;
  lot?: Lote;
  loading?: boolean;
  error?: Error | null;
  onRetry: () => void;
}

export function LotCard({ local, lot, loading, error, onRetry }: LotCardProps) {
  const displayName = local.nome || local.arquivo;
  const liquid = lot?.cessao?.liquido_centavos ?? lot?.totais?.liquido_centavos;
  if (error && !lot) return <article className="dashboard-lot-card dashboard-lot-error"><h3 title={`Arquivo original: ${local.arquivo}`}>{displayName}</h3><p>{error.message}</p><button type="button" className="dashboard-retry" aria-label={`Tentar novamente o resumo de ${displayName}`} onClick={onRetry}>Tentar novamente</button></article>;
  return <Link className="dashboard-lot-card" href={`/lote/${local.id}`} aria-label={`Abrir lote ${displayName}${local.nome ? `. Arquivo original: ${local.arquivo}` : ''}${error ? '. Não foi possível atualizar. Exibindo o último resumo disponível.' : ''}`}>
    <div className="dashboard-lot-top"><div className="dashboard-lot-copy"><h3 title={`Arquivo original: ${local.arquivo}`}>{displayName}</h3><p className="dashboard-lot-value">{loading && !lot ? 'Carregando resumo…' : liquid === undefined ? lot?.status === 'erro' || lot?.status === 'concluido' ? 'Resumo indisponível' : 'Valor em análise' : moneyCentavos(liquid)}</p></div><img className="dashboard-card-caret" src="/figma/caret-right.svg" alt="" /></div>
    {error && <p className="dashboard-lot-stale">Não foi possível atualizar</p>}
    {lot?.totais ? <div className="dashboard-lot-counts"><span className="dashboard-count dashboard-count-total"><span>{countBR(lot.totais.linhas)} notas</span><NotepadText size={16} strokeWidth={1.75} aria-hidden="true" /></span><div className="dashboard-lot-decisions"><span className="dashboard-count dashboard-count-approved" aria-label={`${countBR(lot.totais.aprovadas)} notas aprovadas`}><span>{countBR(lot.totais.aprovadas)}</span><CircleCheck size={16} strokeWidth={1.75} aria-hidden="true" /></span><span className="dashboard-count dashboard-count-rejected" aria-label={`${countBR(lot.totais.recusadas)} notas recusadas`}><span>{countBR(lot.totais.recusadas)}</span><CircleX size={16} strokeWidth={1.75} aria-hidden="true" /></span></div></div> : <div className="dashboard-lot-processing" role={loading ? 'status' : undefined}>{loading ? <><LoaderCircle size={16} className="spin" />Consultando notas</> : lot?.status === 'na_fila' ? `Na fila${lot.posicao_fila === null ? '' : ` · posição ${countBR(lot.posicao_fila)}`}` : lot?.status === 'processando' ? `${lot.progresso.percentual.toLocaleString('pt-BR')}% processado` : lot?.erro?.mensagem || 'Resumo indisponível'}</div>}
  </Link>;
}
