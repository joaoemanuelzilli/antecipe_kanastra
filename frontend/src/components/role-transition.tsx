import { useEffect, useState } from 'react';
import './manager-transition.css';

export function RoleTransition({ role }: { role: 'user' | 'manager' }) {
  const [visible, setVisible] = useState(true);
  useEffect(() => {
    const timer = window.setTimeout(() => setVisible(false), 1900);
    return () => window.clearTimeout(timer);
  }, []);
  if (!visible) return null;
  return <div className="manager-transition" role="status" aria-live="polite" aria-label={role === 'manager' ? 'Entrando como gestor' : 'Entrando como usuário'}><div className="manager-transition-fill" /><div className="manager-transition-copy"><img className="role-transition-bot" src={role === 'manager' ? '/figma/role-manager.png' : '/figma/role-user.png'} width="291" height="163" alt="" /><h2>{role === 'manager' ? 'Entrando como gestor' : 'Entrando como usuário'}</h2><p>Preparando seu ambiente…</p><span className="manager-transition-progress" aria-hidden="true"><span /></span></div></div>;
}
