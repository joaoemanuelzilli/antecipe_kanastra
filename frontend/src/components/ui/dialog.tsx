'use client';

import * as DialogPrimitive from '@radix-ui/react-dialog';
import { X } from 'lucide-react';
import type { ReactNode } from 'react';
import { useRef } from 'react';
import { cx } from '@/lib/format';

export function Modal({ open, onOpenChange, title, description, children, footer, className }: { open: boolean; onOpenChange: (open: boolean) => void; title: string; description?: string; children: ReactNode; footer?: ReactNode; className?: string }) {
  const previousFocus = useRef<HTMLElement | null>(null);
  return <DialogPrimitive.Root open={open} onOpenChange={onOpenChange}><DialogPrimitive.Portal><DialogPrimitive.Overlay className="dialog-overlay" /><DialogPrimitive.Content className={cx('dialog-content', className)} onOpenAutoFocus={() => { previousFocus.current = document.activeElement instanceof HTMLElement ? document.activeElement : null; }} onCloseAutoFocus={event => { event.preventDefault(); previousFocus.current?.focus(); }}><div className="dialog-heading"><DialogPrimitive.Title>{title}</DialogPrimitive.Title><DialogPrimitive.Close className="icon-button" aria-label="Fechar"><X size={20} /></DialogPrimitive.Close></div>{description ? <DialogPrimitive.Description className="dialog-description">{description}</DialogPrimitive.Description> : <DialogPrimitive.Description className="sr-only">Revise as informações antes de continuar.</DialogPrimitive.Description>}<div className="dialog-body">{children}</div>{footer ? <div className="dialog-footer">{footer}</div> : null}</DialogPrimitive.Content></DialogPrimitive.Portal></DialogPrimitive.Root>;
}
