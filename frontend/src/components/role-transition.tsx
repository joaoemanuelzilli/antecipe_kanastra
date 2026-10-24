import { useEffect } from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import './manager-transition.css';

export function RoleTransition({ role, onComplete }: { role: 'user' | 'manager'; onComplete: () => void }) {
  useEffect(() => {
    const timer = window.setTimeout(onComplete, 1500);
    return () => window.clearTimeout(timer);
  }, [onComplete]);
  return <Dialog.Root open><Dialog.Portal><Dialog.Overlay className="role-transition-overlay" /><Dialog.Content className="manager-transition" onEscapeKeyDown={event => event.preventDefault()} onPointerDownOutside={event => event.preventDefault()} onInteractOutside={event => event.preventDefault()} onCloseAutoFocus={event => event.preventDefault()}><div className="manager-transition-copy"><img className="role-transition-bot" src={role === 'manager' ? '/figma/role-manager.png' : '/figma/role-user.png'} width="291" height="163" alt="" /><Dialog.Title>{role === 'manager' ? 'Entrando como gestor' : 'Entrando como usuário'}</Dialog.Title><Dialog.Description>Preparando seu ambiente…</Dialog.Description><span className="manager-transition-progress" aria-hidden="true"><span /></span></div></Dialog.Content></Dialog.Portal></Dialog.Root>;
}
