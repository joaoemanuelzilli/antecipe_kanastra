import { useEffect, useRef, useState } from 'react';
import Link from '@/lib/link';
import { usePathname } from '@/lib/router';
import { cx } from '@/lib/format';
import { MOCK_MODE } from '@/api/client';
import { Modal } from './ui/dialog';
import { clearDemoAccess } from './demo-login-page';

export function Brand() {
  return <Link className="rail-brand" href="/dashboard" aria-label="Antecipe, início"><img src="/figma/logo-mark.svg" width="41" height="32" alt="" /></Link>;
}
export function RoleSwitch() {
  const manager = usePathname() === '/gestor';
  return <nav className="role-switch" aria-label="Perfil de acesso"><Link href="/dashboard" className={cx(!manager && 'selected')} aria-current={!manager ? 'page' : undefined}>Usuário</Link><Link href="/gestor" className={cx(manager && 'selected')} aria-current={manager ? 'page' : undefined}>Gestor</Link></nav>;
}
function HelpGuide() {
  const [open, setOpen] = useState(false);
  const videoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
    const updatePlayback = () => {
      const shouldPlay = !reducedMotion.matches && !document.hidden && !open;
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
  }, [open]);

  return <><button className="mascot-help" onClick={() => setOpen(true)} aria-label="Abrir ajuda sobre a antecipação"><span className="mascot-help-label">Ajuda</span><span className="mascot-help-media" aria-hidden="true"><video ref={videoRef} src="/figma/assistant-head-loop.mp4" poster="/figma/assistant-head-poster.jpg" width="100" height="100" autoPlay muted loop playsInline preload="metadata" tabIndex={-1} /></span></button><Modal open={open} onOpenChange={setOpen} title="Como podemos ajudar?" description="Um guia rápido para acompanhar suas notas."><div className="help-guide"><details open><summary>Como envio minhas notas?</summary><p>Em Novo envio, escolha um arquivo CSV e a data-base. O modelo mostra as colunas esperadas. Você também pode conhecer o fluxo com um lote de exemplo.</p><Link href="/enviar" onClick={() => setOpen(false)} className="text-link">Fazer um novo envio</Link></details><details><summary>O que acontece depois do envio?</summary><p>Acompanhe a fila e as etapas de processamento. O resultado mostra as notas aprovadas e recusadas, os valores e a explicação recebida da análise.</p></details><details><summary>Por que uma nota foi recusada?</summary><p>Abra os detalhes da nota para ver o motivo e o próximo passo. Os filtros ajudam a encontrar notas com o mesmo motivo, e você pode baixar o resultado em CSV.</p></details><details><summary>Quando a cessão é confirmada?</summary><p>Você revisa as notas aprovadas e confirma a cessão no resultado do lote. A confirmação apresenta o valor final e eventuais notas excluídas.</p><Link href="/lotes" onClick={() => setOpen(false)} className="text-link">Consultar meus lotes</Link></details><details><summary>Onde consulto as regras?</summary><p>As regras do fundo e o método de cálculo ficam disponíveis para consulta. O gestor tem uma área própria para revisar e publicar alterações.</p><Link href="/regras" onClick={() => setOpen(false)} className="text-link">Ver regras do fundo</Link></details>{MOCK_MODE && <p className="helper">Você está em uma demonstração com dados fictícios. As confirmações não realizam operações financeiras.</p>}</div></Modal></>;
}
export function Shell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  return <div className="figma-app"><aside className="app-rail"><div className="rail-inner"><Brand /><nav className="rail-nav" aria-label="Navegação principal"><Link href="/dashboard" className={cx('rail-link', (pathname === '/dashboard' || pathname === '/enviar') && 'active')} aria-label="Dashboard" aria-current={pathname === '/dashboard' ? 'page' : undefined} title="Dashboard"><img src="/figma/house.svg" width="24" height="24" alt="" /></Link><Link href="/lotes" className={cx('rail-link', (pathname === '/lotes' || pathname.startsWith('/lote/')) && 'active')} aria-label="Meus lotes" aria-current={pathname === '/lotes' ? 'page' : undefined} title="Meus lotes"><img src="/figma/file-text.svg" width="24" height="24" alt="" /></Link></nav><Link href="/regras" className={cx('rail-link', (pathname === '/regras' || pathname === '/gestor') && 'active')} aria-label="Regras do fundo" aria-current={pathname === '/regras' ? 'page' : undefined} title="Regras do fundo"><img src="/figma/sliders.svg" width="24" height="24" alt="" /></Link></div></aside><div className="app-workspace">{children}<footer className="app-environment">{MOCK_MODE ? <>Demonstração · Dados fictícios <Link href="/login" onClick={clearDemoAccess}>Sair da demonstração</Link></> : 'Antecipe · Antecipação com clareza'}</footer></div><HelpGuide /></div>;
}
