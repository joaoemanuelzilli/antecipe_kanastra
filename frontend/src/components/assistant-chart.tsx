import { useQuery } from '@tanstack/react-query';
import { getLote } from '@/api/client';
import { moneyCentavos } from '@/lib/format';
import type { AssistantChartSpec } from '@/lib/assistant-chart';
import type { Lote } from '@/api/types';
import './assistant-chart.css';
const colors = ['#008d50', '#bc4557', '#3c7896', '#97722c', '#7c5ca5', '#517362'];
export function chartRows(lot: Lote, metric: AssistantChartSpec['metrica']) {
  if (!lot.totais) return [];
  if (metric === 'status') return [{ label: 'Aprovadas', value: lot.totais.aprovadas }, { label: 'Recusadas', value: lot.totais.recusadas }];
  if (metric === 'valores') return [{ label: 'Valor de face', value: lot.totais.face_centavos }, { label: 'Desconto', value: lot.totais.desconto_centavos }, { label: 'Valor líquido', value: lot.totais.liquido_centavos }];
  return lot.motivos.map(reason => ({ label: reason.titulo, value: reason.quantidade })).sort((a, b) => b.value - a.value);
}

export function AssistantChart({ spec, contextLotId }: { spec: AssistantChartSpec; contextLotId?: string }) {
  const id = spec.lote_id ?? contextLotId;
  const lot = useQuery({ queryKey: ['lote', id], enabled: !!id, queryFn: ({ signal }) => getLote(id!, signal), retry: false });
  const format = (value: number) => spec.metrica === 'valores' ? moneyCentavos(value) : value.toLocaleString('pt-BR');
  if (!id) return <p className="assistant-chart-notice">Abra um lote ou informe o identificador para exibir o gráfico</p>;
  if (lot.isPending) return <p className="assistant-chart-notice" role="status">Consultando os dados do gráfico…</p>;
  if (lot.error) return <div className="assistant-chart-notice" role="alert">Não foi possível consultar os dados do gráfico <button type="button" onClick={() => { void lot.refetch(); }}>Tentar novamente</button></div>;
  const rows = chartRows(lot.data, spec.metrica);
  if (lot.data.status !== 'concluido' || !lot.data.totais) return <p className="assistant-chart-notice">O gráfico ficará disponível quando a análise terminar</p>;
  if (!rows.length || rows.every(row => row.value === 0)) return <p className="assistant-chart-notice">Não há valores para exibir neste gráfico</p>;
  if (rows.some(row => !Number.isSafeInteger(row.value) || row.value < 0)) return <p className="assistant-chart-notice">Os dados recebidos não permitem montar este gráfico</p>;
  const total = rows.reduce((sum, row) => sum + row.value, 0);
  const max = Math.max(...rows.map(row => row.value), 1);
  return <figure className="assistant-chart"><figcaption>{spec.titulo}</figcaption><p className="assistant-chart-source">Dados da análise · Lote {id.slice(0, 8)}</p>
    {spec.tipo === 'rosca' ? <div className="assistant-chart-donut"><svg viewBox="0 0 160 160" role="img" aria-label={`${spec.titulo}: ${rows.map(row => `${row.label}, ${format(row.value)}`).join('; ')}`}><circle cx="80" cy="80" r="58" fill="none" stroke="#edf3ef" strokeWidth="20" />{rows.map((row, index) => { const start = rows.slice(0, index).reduce((sum, previous) => sum + previous.value, 0) / total * 100; return <circle key={row.label} cx="80" cy="80" r="58" fill="none" stroke={colors[index % colors.length]} strokeWidth="20" pathLength="100" strokeDasharray={`${row.value / total * 100} 100`} strokeDashoffset={-start} transform="rotate(-90 80 80)" />; })}</svg><ul>{rows.map((row, index) => <li key={row.label}><span><i style={{ background: colors[index % colors.length] }} />{row.label}</span><strong>{format(row.value)}</strong></li>)}</ul></div> : <div className="assistant-chart-bars">{rows.slice(0, 12).map((row, index) => <div key={row.label}><div><span>{row.label}</span><strong>{format(row.value)}</strong></div><div className="assistant-chart-track" role="img" aria-label={`${row.label}: ${format(row.value)}`}><span style={{ width: `${row.value / max * 100}%`, background: colors[index % colors.length] }} /></div></div>)}</div>}
    {rows.length > 12 && spec.tipo === 'barras' && <p className="assistant-chart-source">Mostrando os 12 maiores motivos</p>}
    <details><summary>Ver dados do gráfico</summary><div className="assistant-table-scroll" tabIndex={0} role="region" aria-label="Dados do gráfico"><table><thead><tr><th>Categoria</th><th>Valor</th></tr></thead><tbody>{rows.map(row => <tr key={row.label}><td>{row.label}</td><td>{format(row.value)}</td></tr>)}</tbody></table></div></details>
  </figure>;
}
