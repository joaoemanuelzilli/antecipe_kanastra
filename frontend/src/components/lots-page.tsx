import { useRef, useState } from 'react';
import { useQueries } from '@tanstack/react-query';
import { ChevronDown, LoaderCircle, Search, SlidersHorizontal, X } from 'lucide-react';
import { getLote, lotesLocais } from '@/api/client';
import { countBR } from '@/lib/format';
import { filterLots, LOT_SORT_LABELS, LOT_STATUS_LABELS, type LotSort, type LotStatusFilter } from '@/lib/lot-filters';
import { LotCard } from './lot-card';
import { NewLotLink } from './new-lot-link';
import { PageHeader } from './page-header';
import './lots-page.css';

export function LotsPage() {
  const [localLots] = useState(lotesLocais);
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState<LotStatusFilter>('todos');
  const [dataBase, setDataBase] = useState('');
  const [order, setOrder] = useState<LotSort>('recentes');
  const [moreFilters, setMoreFilters] = useState(false);
  const searchRef = useRef<HTMLInputElement>(null);
  const queries = useQueries({ queries: localLots.map(local => ({
    queryKey: ['lote', local.id],
    queryFn: ({ signal }: { signal: AbortSignal }) => getLote(local.id, signal),
    refetchInterval: (query: { state: { data: Awaited<ReturnType<typeof getLote>> | undefined } }) => query.state.data?.status === 'na_fila' || query.state.data?.status === 'processando' ? 500 : false as const,
  })) });
  const allEntries = localLots.map((local, index) => ({ local, lot: queries[index].data, loading: queries[index].isPending, error: queries[index].error, result: queries[index] }));
  const entries = filterLots(allEntries, { busca: search, status, dataBase, ordenacao: order });
  const pending = allEntries.filter(entry => entry.loading).length;
  const unavailable = allEntries.filter(entry => entry.error && !entry.lot);
  const activeFilters = Number(status !== 'todos') + Number(Boolean(dataBase));
  const hasFilters = Boolean(search.trim() || activeFilters);

  function clearFilters() {
    setSearch('');
    setStatus('todos');
    setDataBase('');
    setOrder('recentes');
    searchRef.current?.focus();
  }

  return <main className="container page-content lots-page">
    <PageHeader title="Meus lotes" description="Acompanhe suas análises e consulte os detalhes de cada nota." />
    <div className="lots-overview">
      <p className="lots-total">{countBR(localLots.length)} {localLots.length === 1 ? 'lote no histórico' : 'lotes no histórico'}</p>
      <NewLotLink />
    </div>
    {localLots.length ? <>
      <form className="lots-filters" role="search" aria-label="Filtrar lotes" onSubmit={event => event.preventDefault()}>
        <div className="lots-filter-bar">
          <label className="lots-field lots-search">
            <span>Buscar lote</span>
            <span className="lots-control-wrap"><Search size={20} aria-hidden="true" /><input ref={searchRef} className="lots-control" type="search" placeholder="Buscar por nome ou arquivo" value={search} onChange={event => setSearch(event.target.value)} /></span>
          </label>
          <label className="lots-field">
            <span>Situação</span>
            <span className="lots-select-wrap"><select className="lots-control" value={status} onChange={event => setStatus(event.target.value as LotStatusFilter)}>{Object.entries(LOT_STATUS_LABELS).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select><ChevronDown size={20} aria-hidden="true" /></span>
          </label>
          <button type="button" className="lots-filter-trigger" aria-expanded={moreFilters} aria-controls="lots-more-filters" onClick={() => setMoreFilters(value => !value)}><SlidersHorizontal size={20} aria-hidden="true" />Filtros{activeFilters ? <span className="lots-filter-count" aria-label={activeFilters === 1 ? "1 filtro ativo" : `${activeFilters} filtros ativos`}>{activeFilters}</span> : null}</button>
        </div>
        <div className="lots-more-filters" id="lots-more-filters" hidden={!moreFilters}>
          <label className="lots-field"><span>Data-base</span><input className="lots-control" type="date" value={dataBase} onChange={event => setDataBase(event.target.value)} /></label>
          <label className="lots-field"><span>Ordenar por</span><span className="lots-select-wrap"><select className="lots-control" value={order} onChange={event => setOrder(event.target.value as LotSort)}>{Object.entries(LOT_SORT_LABELS).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select><ChevronDown size={20} aria-hidden="true" /></span></label>
        </div>
      </form>
      <div className="lots-results-bar">
        <p className="lots-results-count" aria-live="polite" aria-atomic="true">{pending && !entries.length ? 'Consultando seus lotes…' : `Exibindo ${countBR(entries.length)} de ${countBR(localLots.length)} ${localLots.length === 1 ? 'lote' : 'lotes'}`}{pending && entries.length ? ` · Consultando ${countBR(pending)}…` : ''}</p>
        {hasFilters ? <button className="lots-clear" type="button" onClick={clearFilters}><X size={16} aria-hidden="true" />Limpar filtros</button> : null}
      </div>
      {hasFilters && unavailable.length ? <div className="lots-summary-warning"><p>{unavailable.length === 1 ? 'O resumo de 1 lote está indisponível.' : `Os resumos de ${countBR(unavailable.length)} lotes estão indisponíveis.`} Alguns resultados podem não aparecer nos filtros.</p><button className="lots-clear" type="button" disabled={unavailable.some(entry => entry.result.isFetching)} onClick={() => { void Promise.allSettled(unavailable.map(entry => entry.result.refetch())); }}>{unavailable.some(entry => entry.result.isFetching) ? 'Consultando resumos…' : 'Consultar resumos novamente'}</button></div> : null}
      {entries.length ? <section className="lots-card-grid" aria-label="Lotes enviados neste navegador">{entries.map(({ local, lot, loading, error, result }) => <LotCard key={local.id} local={local} lot={lot} loading={loading} error={error} onRetry={() => { void result.refetch(); }} />)}</section> : <section className="lots-empty-result">{pending ? <><LoaderCircle size={28} className="spin" aria-hidden="true" /><h2>Consultando seus lotes</h2><p>Estamos carregando os resumos para aplicar os filtros.</p></> : <><Search size={28} aria-hidden="true" /><h2>{unavailable.length ? 'Sem resultados disponíveis' : 'Nenhum lote encontrado'}</h2><p>{unavailable.length ? 'Consulte os resumos indisponíveis ou ajuste os filtros.' : 'Tente outro nome ou ajuste os filtros.'}</p><button className="lots-clear" type="button" onClick={clearFilters}>Limpar filtros</button></>}</section>}
    </> : <section className="lots-first-empty"><img src="/figma/mascot-empty-box.png" width="110" height="124" alt="" /><h2>Seu primeiro lote começa aqui</h2><p>Faça um novo envio ou use o lote de exemplo para conhecer o processo.</p></section>}
    <p className="lots-history-note">Histórico dos últimos 100 envios feitos neste navegador.</p>
  </main>;
}
