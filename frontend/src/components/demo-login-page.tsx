import { useState, type FormEvent } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { clearGestorSenha, MOCK_MODE } from '@/api/client';
import Link from '@/lib/link';
import { Modal } from './ui/dialog';
import { Button } from './ui/button';
import './demo-login.css';

const DEMO_ACCESS_KEY = 'antecipe.demo.access.v1';

export function isDemoAccessGranted(): boolean {
  if (!MOCK_MODE) return false;
  try { return sessionStorage.getItem(DEMO_ACCESS_KEY) === 'true'; }
  catch { return false; }
}

export function clearDemoAccess(): void {
  clearGestorSenha();
  try { sessionStorage.removeItem(DEMO_ACCESS_KEY); }
  catch { /* An unavailable session store already means access is not granted. */ }
}

export function DemoLoginPage({ onEnter }: { onEnter?: () => void } = {}) {
  const navigate = useNavigate();
  const location = useLocation();
  const destination = ['/dashboard', '/enviar', '/perfil', '/regras', '/lotes', '/gestor'].includes(location.state?.from) ? location.state.from as string : '/dashboard';
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [passwordVisible, setPasswordVisible] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);
  const [error, setError] = useState('');

  function enterDemo() {
    if (!MOCK_MODE) return;
    try { sessionStorage.setItem(DEMO_ACCESS_KEY, 'true'); }
    catch {
      setError('Não foi possível iniciar a demonstração. Permita o armazenamento da sessão neste navegador e tente novamente.');
      return;
    }
    // Only a boolean marker survives this screen; fictional form values are discarded.
    setEmail(''); setPassword('');
    onEnter?.();
    navigate(destination, { replace: true });
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!event.currentTarget.reportValidity()) return;
    enterDemo();
  }

  return <main className="demo-login-page">
    <img className="demo-login-background" src="/figma/login-background.png" alt="" aria-hidden="true" />
    <Link href="/" className="demo-login-logo" aria-label="Antecipe — voltar à página inicial"><img src="/figma/logo-white.svg" alt="Antecipe" width={296} height={48} /></Link>
    {MOCK_MODE ? <div className="demo-login-stack">
      <section className="demo-login-card" aria-labelledby="demo-login-title">
        <header className="demo-login-intro"><h1 id="demo-login-title">Que bom te ver aqui!</h1><p>Entre na demonstração e conheça o Antecipe.</p></header>
        <form className="demo-login-form" onSubmit={submit} autoComplete="off">
          <p className="demo-login-notice">Acesso de demonstração · Use dados fictícios</p>
          <div className="demo-login-fields">
            <label className="sr-only" htmlFor="demo-login-email">E-mail fictício</label>
            <input id="demo-login-email" name="demo-email" className="demo-login-input" type="email" placeholder="Insira seu e-mail fictício" value={email} onChange={(event) => { setEmail(event.target.value); setError(''); }} required autoComplete="off" spellCheck={false} autoCapitalize="none" aria-describedby="demo-login-guidance" />
            <div className="demo-login-password">
              <label className="sr-only" htmlFor="demo-login-password">Senha fictícia</label>
              <input id="demo-login-password" name="demo-password" className="demo-login-input" type={passwordVisible ? 'text' : 'password'} placeholder="Insira uma senha fictícia" value={password} onChange={(event) => { setPassword(event.target.value); setError(''); }} required autoComplete="off" aria-describedby="demo-login-guidance" />
              <button className="demo-login-eye" type="button" onClick={() => setPasswordVisible((visible) => !visible)} aria-label={passwordVisible ? 'Ocultar senha' : 'Mostrar senha'} aria-pressed={passwordVisible}><img src="/figma/eye-closed.svg" alt="" width={24} height={24} /></button>
            </div>
          </div>
          <button className="demo-login-help" type="button" onClick={() => setHelpOpen(true)}>Como acessar a demonstração?</button>
          <button className="demo-login-submit" type="submit">Acessar demonstração</button>
          <p className="demo-login-guidance" id="demo-login-guidance">Nenhum cadastro ou senha real é necessário.</p>
          {error ? <p className="demo-login-error" role="alert">{error}</p> : null}
        </form>
      </section>
      <button className="demo-login-explore" type="button" onClick={enterDemo}>
        <span className="demo-login-mascot" aria-hidden="true"><img src="/figma/mascot-welcome.png" alt="" /></span>
        <span className="demo-login-explore-content"><span>Explorar demonstração</span><img src="/figma/arrow-right.svg" alt="" width={24} height={24} /></span>
      </button>
    </div> : <section className="demo-login-card demo-login-unavailable" aria-labelledby="demo-login-title"><header className="demo-login-intro"><h1 id="demo-login-title">Acesso de demonstração</h1><p>Esta tela está disponível apenas no ambiente de demonstração.</p></header><Button onClick={() => navigate('/', { replace: true })}>Voltar ao início</Button></section>}
    <Modal open={helpOpen} onOpenChange={setHelpOpen} title="Acesso de demonstração" description="Use qualquer e-mail fictício e uma senha inventada para explorar o Antecipe." className="demo-login-help-dialog" footer={<Button onClick={() => setHelpOpen(false)}>Entendi</Button>}><p>Nenhum cadastro ou senha real é necessário.</p><p className="demo-login-help-copy">Os dados preenchidos são descartados ao entrar. Você também pode escolher “Explorar demonstração” para acessar sem preencher o formulário.</p></Modal>
  </main>;
}
