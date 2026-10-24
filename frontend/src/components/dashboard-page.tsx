import { useQueries } from '@tanstack/react-query';
import { getLote, lotesLocais } from '@/api/client';
import type { Lote } from '@/api/types';
import Link from '@/lib/link';
import { PageHeader } from './page-header';
import { LotCard } from './lot-card';
import { NewLotLink } from './new-lot-link';
import './dashboard.css';

export function DashboardPage() {
  const recentLots = lotesLocais().slice(0, 3);
  const summaries = useQueries({ queries: recentLots.map(local => ({ queryKey: ['lote', local.id], queryFn: ({ signal }: { signal: AbortSignal }) => getLote(local.id, signal), refetchInterval: (query: { state: { data: Lote | undefined } }) => query.state.data?.status === 'na_fila' || query.state.data?.status === 'processando' ? 500 : false as const })) });

  return <main className="container dashboard-page">
    <PageHeader title="Dashboard" description="Veja os resumos das suas notas e da sua conta." />
    <section className="dashboard-hero" aria-labelledby="dashboard-upload-heading"><img className="dashboard-hero-shape" src="/figma/hero-shape.svg" alt="" /><div className="dashboard-hero-mascot" aria-hidden="true"><img src="/figma/mascot-magnify.png" alt="" /></div><div className="dashboard-hero-content"><div className="dashboard-hero-copy"><h2 id="dashboard-upload-heading">Antecipe suas notas</h2><p>Envie o CSV e veja imediatamente quanto você<br className="dashboard-hero-line-break" /> recebe e o motivo de alguma nota não ter sido incluída.</p></div><NewLotLink className="dashboard-upload-link" /></div></section>
    <section className="dashboard-lots" aria-labelledby="dashboard-lots-heading"><div className="dashboard-section-heading"><h2 id="dashboard-lots-heading">Meus lotes</h2><Link href="/lotes">Ver todos</Link></div><div className="dashboard-lots-grid">{recentLots.length ? recentLots.map((local, index) => <LotCard key={local.id} local={local} lot={summaries[index].data} loading={summaries[index].isPending} error={summaries[index].error} onRetry={() => { void summaries[index].refetch(); }} />) : <div className="dashboard-empty"><div className="dashboard-empty-mascot" aria-hidden="true"><img src="/figma/mascot-empty-box.png" alt="" /></div><p>Você ainda não cadastrou nenhum lote.</p></div>}</div></section>
  </main>;
}
