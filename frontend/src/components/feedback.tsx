import { AlertCircle, CheckCircle2, LoaderCircle } from 'lucide-react';
import { cx } from '@/lib/format';

export function Feedback({ children, variant = 'error' }: { children: React.ReactNode; variant?: 'error' | 'success' | 'info' }) {
  return <div className={cx('feedback', `feedback-${variant}`)} role={variant === 'error' ? 'alert' : 'status'}>{variant === 'success' ? <CheckCircle2 size={20} /> : <AlertCircle size={20} />}<div>{children}</div></div>;
}
export function Loading({ label = 'Carregando…' }: { label?: string }) { return <div className="loading-state" role="status"><LoaderCircle className="spin" size={28} /><p>{label}</p></div>; }
