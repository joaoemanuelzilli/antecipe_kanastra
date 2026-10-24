import Link from '@/lib/link';
import { cx } from '@/lib/format';
import './new-lot-link.css';

export interface NewLotLinkProps {
  className?: string;
}

export function NewLotLink({ className }: NewLotLinkProps) {
  return <Link className={cx('new-lot-link', className)} href="/enviar">Novo envio<img src="/figma/plus.svg" alt="" /></Link>;
}
