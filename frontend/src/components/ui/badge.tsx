import { Check, X } from 'lucide-react';
import { cx } from '@/lib/format';

export function StatusBadge({ status, children }: { status: 'aprovada' | 'recusada'; children?: React.ReactNode }) {
  const approved = status === 'aprovada';
  return <span className={cx('status-badge', approved ? 'status-approved' : 'status-rejected')}><span className="status-icon">{approved ? <Check size={10} strokeWidth={3} /> : <X size={10} strokeWidth={3} />}</span>{children ?? (approved ? 'Aprovada' : 'Recusada')}</span>;
}
