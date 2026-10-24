import { useEffect, useState } from 'react';
import { Check, Clock3, LoaderCircle } from 'lucide-react';
import type { Lote } from '@/api/types';

const number = (value: number) => value.toLocaleString('pt-BR', { maximumFractionDigits: 1 });
export function ProcessingStatus({ lot }: { lot: Lote }) {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => { const timer = window.setInterval(() => setNow(Date.now()), 1000); return () => window.clearInterval(timer); }, []);
  const duration = lot.duracao_ms ?? (lot.iniciado_em ? Math.max(0, now - new Date(lot.iniciado_em).getTime()) : null);
  const seconds = duration === null ? null : Math.floor(duration / 1000);
  const current = lot.etapas.find(step => step.codigo === lot.etapa || step.status === 'em_andamento');
  const percent = Math.min(100, Math.max(0, lot.progresso.percentual));
  return <section className="panel processing-panel" aria-labelledby="processing-title"><div className="processing-heading"><div className="processing-symbol">{lot.status === 'na_fila' ? <Clock3 size={27} /> : <LoaderCircle size={27} className="spin" />}</div><div><h2 id="processing-title">{lot.status === 'na_fila' ? 'Seu lote está na fila' : 'Analisando suas notas'}</h2><p>{lot.status === 'na_fila' ? lot.posicao_fila === null ? 'Aguardando o início do processamento.' : `Seu lote é o ${lot.posicao_fila.toLocaleString('pt-BR')}º da fila.` : current?.titulo || 'O processamento está em andamento.'}</p></div><strong className="processing-percentage">{number(percent)}%</strong></div><div className="processing-progress-track" role="progressbar" aria-label="Progresso do processamento" aria-valuemin={0} aria-valuemax={100} aria-valuenow={percent}><span style={{ width: `${percent}%` }} /></div><div className="processing-metadata"><span>{number(lot.progresso.linhas_processadas)} de {number(lot.progresso.linhas_total)} linhas processadas</span><span>{seconds === null ? 'Aguardando início' : `${Math.floor(seconds / 60) > 0 ? `${Math.floor(seconds / 60)} min ` : ''}${seconds % 60} s de processamento`}</span></div><ol className="processing-steps">{lot.etapas.map((step, index) => <li key={step.codigo} className={`processing-step step-${step.status}`} aria-current={step.status === 'em_andamento' ? 'step' : undefined}><span className="processing-step-number">{step.status === 'concluida' ? <Check size={15} /> : step.status === 'em_andamento' ? <LoaderCircle size={15} className="spin" /> : String(index + 1).padStart(2, '0')}</span><span>{step.titulo}</span><small>{step.status === 'concluida' ? 'Concluída' : step.status === 'em_andamento' ? 'Em andamento' : 'Pendente'}</small></li>)}</ol><p className="helper">Os resultados aparecerão aqui quando todas as etapas terminarem.</p></section>;
}
