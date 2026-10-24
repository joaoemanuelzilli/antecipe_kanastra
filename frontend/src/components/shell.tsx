import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { RoleTransition } from './role-transition';
import Link from '@/lib/link';
import { Settings2, UserRound } from 'lucide-react';
import { useDemoProfile } from '@/lib/demo-profile';
import { usePathname } from '@/lib/router';
import { cx } from '@/lib/format';
import { MOCK_MODE, setGestorSenha } from '@/api/client';
import './compact-rail.css';
import { AssistantChat } from './assistant-chat';
import { clearDemoAccess } from './demo-login-page';

export function Brand() {
  return <Link className="rail-brand" href="/dashboard" aria-label="Antecipe, início"><img src="/figma/logo-mark.svg" width="41" height="32" alt="" /></Link>;
}
export function RoleSwitch() {
  const manager = usePathname() === '/gestor';
  const navigate = useNavigate();
  const [pending, setPending] = useState<'user' | 'manager' | null>(null);
  function switchRole(role: 'user' | 'manager') {
    if (pending || (role === 'manager') === manager) return;
    setPending(role);
  }
  return <><nav className="role-switch" aria-label="Perfil de acesso"><Link href="/dashboard" onClick={event => { event.preventDefault(); switchRole('user'); }} className={cx(!manager && 'selected')} aria-current={!manager ? 'page' : undefined}>Usuário</Link><Link href="/gestor" onClick={event => { event.preventDefault(); switchRole('manager'); }} className={cx(manager && 'selected')} aria-current={manager ? 'page' : undefined}>Gestor</Link></nav>{pending ? <RoleTransition role={pending} onComplete={() => { if (pending === 'manager' && MOCK_MODE) setGestorSenha('demo-gestor'); navigate(pending === 'manager' ? '/gestor' : '/dashboard'); setPending(null); }} /> : null}</>;
}

function HelpGuide() {
  const pathname = usePathname();
  const [open, setOpen] = useState(false);
  const [question, setQuestion] = useState('');
  const [questionRequest, setQuestionRequest] = useState(0);
  useEffect(() => {
    const openChat = (event: Event) => { setQuestion((event as CustomEvent<string>).detail || ''); setQuestionRequest(value => value + 1); setOpen(true); };
    window.addEventListener('antecipe:chat-open', openChat);
    return () => window.removeEventListener('antecipe:chat-open', openChat);
  }, []);
  const videoRef = useRef<HTMLVideoElement>(null);
  const { reducedMotion: userReducedMotion } = useDemoProfile();

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
    const updatePlayback = () => {
      const shouldPlay = !reducedMotion.matches && !userReducedMotion && !document.hidden && !open;
      video.autoplay = shouldPlay;
      if (shouldPlay) {
        void video.play().catch(() => { /* The poster stays visible if autoplay is unavailable. */ });
      } else {
        video.pause();
      }
    };
    updatePlayback();
    reducedMotion.addEventListener('change', updatePlayback);
    document.addEventListener('visibilitychange', updatePlayback);
    return () => {
      reducedMotion.removeEventListener('change', updatePlayback);
      document.removeEventListener('visibilitychange', updatePlayback);
      video.pause();
    };
  }, [open, userReducedMotion]);

  return <><button className="mascot-help" onClick={() => { setQuestion(''); setOpen(true); }} aria-label="Abrir chat com a IA"><span className="mascot-help-label">Fale com a IA</span><span className="mascot-help-media" aria-hidden="true"><video ref={videoRef} src="/figma/assistant-head-loop.mp4" poster="/figma/assistant-head-poster.jpg" width="100" height="100" autoPlay muted loop playsInline preload="metadata" tabIndex={-1} /></span></button><AssistantChat key={`${pathname}:${question}:${questionRequest}`} open={open} onOpenChange={setOpen} initialQuestion={question} /></>;
}
export function Shell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const { reducedMotion } = useDemoProfile();
  useEffect(() => { document.documentElement.dataset.profileReducedMotion = String(reducedMotion); return () => { delete document.documentElement.dataset.profileReducedMotion; }; }, [reducedMotion]);
  const manager = pathname === '/gestor';
  return <div className="figma-app"><aside className="app-rail"><div className="rail-inner"><Brand />{manager ? <nav className="rail-nav" aria-label="Navegação do gestor"><Link href="/gestor" className="rail-link active" aria-label="Configuração do fundo" aria-current="page" title="Configuração do fundo"><Settings2 size={24} strokeWidth={1.5} aria-hidden="true" /><span className="rail-label">Configuração</span></Link></nav> : <><nav className="rail-nav" aria-label="Navegação principal"><Link href="/dashboard" className={cx('rail-link', (pathname === '/dashboard' || pathname === '/enviar') && 'active')} aria-label="Dashboard" aria-current={pathname === '/dashboard' ? 'page' : undefined} title="Dashboard"><img src="/figma/house.svg" width="24" height="24" alt="" /><span className="rail-label">Início</span></Link><Link href="/lotes" className={cx('rail-link', (pathname === '/lotes' || pathname.startsWith('/lote/')) && 'active')} aria-label="Meus lotes" aria-current={pathname === '/lotes' ? 'page' : undefined} title="Meus lotes"><img src="/figma/file-text.svg" width="24" height="24" alt="" /><span className="rail-label">Lotes</span></Link></nav><Link href="/perfil" className={cx('rail-link', pathname === '/perfil' && 'active')} aria-label="Meu perfil" aria-current={pathname === '/perfil' ? 'page' : undefined} title="Meu perfil"><UserRound size={24} strokeWidth={1.5} aria-hidden="true" /><span className="rail-label">Perfil</span></Link></>}</div></aside><div className="app-workspace">{children}<footer className="app-environment">{MOCK_MODE ? <>Demonstração · Dados fictícios <Link href="/login" onClick={clearDemoAccess}>Sair da demonstração</Link></> : 'Antecipe · Antecipação com clareza'}</footer></div><HelpGuide /></div>;
}
