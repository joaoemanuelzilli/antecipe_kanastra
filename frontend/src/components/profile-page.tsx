import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowRight, Building2, Check, Eye, EyeOff, KeyRound, LogOut, Monitor, ShieldCheck, UserRound } from 'lucide-react';
import { clearGestorSenha, MOCK_MODE } from '@/api/client';
import { saveDemoProfile, useDemoProfile, type DemoProfile } from '@/lib/demo-profile';
import Link from '@/lib/link';
import { clearDemoAccess } from './demo-login-page';
import { notify } from './notifications';
import { PageHeader } from './page-header';
import { Button } from './ui/button';
import { Modal } from './ui/dialog';
import { Switch } from './ui/switch';
import './profile-page.css';

type ProfileFields = Pick<DemoProfile, 'name' | 'email' | 'company' | 'role'>;
const fields = ({ name, email, company, role }: DemoProfile): ProfileFields => ({ name, email, company, role });

export function ProfilePage() {
  const profile = useDemoProfile();
  const navigate = useNavigate();
  const [draft, setDraft] = useState(() => fields(profile));
  const [error, setError] = useState('');
  const [passwordOpen, setPasswordOpen] = useState(false);
  const [logoutOpen, setLogoutOpen] = useState(false);
  const dirty = Object.keys(draft).some(key => draft[key as keyof ProfileFields] !== profile[key as keyof ProfileFields]);
  const initials = profile.name.trim().split(/\s+/).filter(Boolean).slice(0, 2).map(word => word[0]).join('').toLocaleUpperCase('pt-BR') || 'MC';

  function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!event.currentTarget.reportValidity()) return;
    const next = Object.fromEntries(Object.entries(draft).map(([key, value]) => [key, value.trim()])) as ProfileFields;
    if (!next.name || !next.company) { setError('Preencha seu nome e o nome da empresa.'); return; }
    if (!saveDemoProfile({ ...profile, ...next })) { setError('Não foi possível salvar nesta sessão. Tente novamente.'); return; }
    setDraft(next); setError(''); notify.success('Suas alterações foram salvas nesta sessão.');
  }
  function preference(key: 'confirmLogout' | 'reducedMotion', value: boolean) {
    if (!saveDemoProfile({ ...profile, [key]: value })) { setError('Não foi possível salvar sua preferência.'); return; }
    setError(''); notify.success('Preferência atualizada.');
  }
  function logout() { clearDemoAccess(); clearGestorSenha(); navigate(MOCK_MODE ? '/login' : '/', { replace: true }); }

  return <main className="container page-content profile-page">
    <PageHeader title="Meu perfil" description="Seus dados e preferências, do seu jeito." />
    {MOCK_MODE ? <>
      <section className="profile-identity" aria-label="Resumo do perfil"><span className="profile-avatar" aria-hidden="true">{initials}</span><div className="profile-identity-copy"><h2>{profile.name}</h2><p>{profile.email}</p><span className="profile-role">{profile.role || 'Responsável pela conta'}</span></div><div className="profile-demo"><span><span />Demonstração</span><p>Dados fictícios nesta sessão.</p></div></section>
      {error ? <p className="profile-error" role="alert">{error}</p> : null}
      <div className="profile-grid"><div className="profile-main">
        <section className="profile-panel"><header className="profile-section-heading"><span className="profile-section-icon"><UserRound size={21} aria-hidden="true" /></span><div><h2>Informações pessoais</h2><p>Como você se apresenta por aqui.</p></div></header><form onSubmit={save}>
          <div className="profile-fields"><label className="profile-field"><span>Nome completo</span><input className="profile-input" autoComplete="off" value={draft.name} maxLength={80} required onChange={event => setDraft({ ...draft, name: event.target.value })} /></label><label className="profile-field"><span>E-mail</span><input className="profile-input" type="email" autoComplete="off" value={draft.email} maxLength={120} required onChange={event => setDraft({ ...draft, email: event.target.value })} /></label><label className="profile-field"><span>Empresa</span><input className="profile-input" autoComplete="off" value={draft.company} maxLength={80} required onChange={event => setDraft({ ...draft, company: event.target.value })} /></label><label className="profile-field"><span>Cargo</span><input className="profile-input" autoComplete="off" placeholder="Seu cargo na empresa" value={draft.role} maxLength={80} onChange={event => setDraft({ ...draft, role: event.target.value })} /></label></div>
          <div className="profile-form-footer"><p>Use dados fictícios para explorar.</p><div><Button type="button" variant="ghost" disabled={!dirty} onClick={() => { setDraft(fields(profile)); setError(''); }}>Cancelar</Button><Button type="submit" disabled={!dirty}>Salvar alterações<Check size={18} aria-hidden="true" /></Button></div></div>
        </form></section>
        <section className="profile-panel"><header className="profile-section-heading"><span className="profile-section-icon"><ShieldCheck size={21} aria-hidden="true" /></span><div><h2>Preferências</h2><p>Deixe sua experiência mais confortável.</p></div></header><div className="profile-preference"><div id="profile-motion-label"><h3>Reduzir animações</h3><p>Pause os vídeos do Fred e reduza o movimento da interface.</p></div><Switch checked={profile.reducedMotion} onCheckedChange={value => preference('reducedMotion', value)} label="Reduzir animações" /></div><div className="profile-preference"><div><h3>Confirmar antes de sair</h3><p>Peça uma confirmação para encerrar sua sessão.</p></div><Switch checked={profile.confirmLogout} onCheckedChange={value => preference('confirmLogout', value)} label="Confirmar antes de sair" /></div></section>
      </div><aside className="profile-aside" aria-label="Configurações da conta">
        <section className="profile-panel"><span className="profile-section-icon"><KeyRound size={22} aria-hidden="true" /></span><h2>Senha e acesso</h2><p className="profile-panel-copy">Experimente a troca de senha no seu perfil de demonstração.</p><Button type="button" variant="outline" onClick={() => setPasswordOpen(true)}>Alterar senha<ArrowRight size={17} aria-hidden="true" /></Button><p className="profile-small-note">Nenhuma credencial real é alterada.</p></section>
        <section className="profile-panel"><header className="profile-section-heading"><span className="profile-section-icon"><Monitor size={21} aria-hidden="true" /></span><div><h2>Sessão atual</h2><p>Você está conectado por aqui.</p></div></header><div className="profile-device"><span>Navegador atual</span><span className="profile-device-badge">Em uso</span></div><p className="profile-small-note">Seus lotes ficam disponíveis no histórico deste navegador.</p><button type="button" className="profile-logout" onClick={() => { if (profile.confirmLogout) setLogoutOpen(true); else logout(); }}><LogOut size={18} aria-hidden="true" />Sair da conta</button></section>
        <section className="profile-tip"><Building2 size={23} aria-hidden="true" /><div><h2>Tudo da sua empresa, por aqui.</h2><p>Acompanhe seus envios e consulte o resultado de cada lote.</p><Link href="/lotes">Ver meus lotes<ArrowRight size={16} aria-hidden="true" /></Link></div></section>
      </aside></div>
      <PasswordDemo open={passwordOpen} onOpenChange={setPasswordOpen} onSuccess={() => notify.success('Troca de senha simulada. Nenhuma credencial real foi alterada.')} />
      <Modal open={logoutOpen} onOpenChange={setLogoutOpen} title="Sair da sua conta?" description="Você pode voltar quando quiser. Seus lotes continuam no histórico deste navegador." footer={<><Button type="button" variant="ghost" onClick={() => setLogoutOpen(false)}>Continuar por aqui</Button><Button type="button" onClick={logout}><LogOut size={17} />Sair da conta</Button></>}><p>Vamos encerrar seu acesso à demonstração.</p></Modal>
    </> : <section className="profile-panel profile-unavailable"><span className="profile-section-icon"><UserRound size={24} /></span><h2>Seu perfil estará aqui</h2><p>As configurações da conta estarão disponíveis quando seu acesso for configurado.</p><Button asChild variant="outline"><Link href="/dashboard">Voltar ao início</Link></Button></section>}
  </main>;
}

function PasswordDemo({ open, onOpenChange, onSuccess }: { open: boolean; onOpenChange: (open: boolean) => void; onSuccess: () => void }) {
  const [current, setCurrent] = useState('');
  const [next, setNext] = useState('');
  const [confirmation, setConfirmation] = useState('');
  const [visible, setVisible] = useState(false);
  const [error, setError] = useState('');
  function changeOpen(value: boolean) { setCurrent(''); setNext(''); setConfirmation(''); setVisible(false); setError(''); onOpenChange(value); }
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!event.currentTarget.reportValidity()) return;
    if (next !== confirmation) { setError('As novas senhas precisam ser iguais.'); return; }
    if (next === current) { setError('Escolha uma senha fictícia diferente da atual.'); return; }
    changeOpen(false); onSuccess();
  }
  return <Modal open={open} onOpenChange={changeOpen} title="Alterar senha" description="Simulação: use apenas senhas fictícias. Nenhuma credencial será salva ou enviada." className="profile-password-dialog"><form className="profile-password-form" onSubmit={submit} autoComplete="off"><label className="profile-field"><span>Senha atual fictícia</span><input className="profile-input" type={visible ? 'text' : 'password'} autoComplete="off" required maxLength={128} value={current} onChange={event => setCurrent(event.target.value)} /></label><div className="profile-password-new"><label className="profile-field"><span>Nova senha fictícia</span><input className="profile-input" type={visible ? 'text' : 'password'} autoComplete="off" required minLength={8} maxLength={128} aria-describedby="profile-new-password-hint" value={next} onChange={event => setNext(event.target.value)} /></label><p className="profile-small-note" id="profile-new-password-hint">Use pelo menos 8 caracteres.</p></div><label className="profile-field"><span>Confirmar nova senha fictícia</span><input className="profile-input" type={visible ? 'text' : 'password'} autoComplete="off" required minLength={8} maxLength={128} value={confirmation} onChange={event => setConfirmation(event.target.value)} /></label><button type="button" className="profile-password-visibility" onClick={() => setVisible(value => !value)}>{visible ? <EyeOff size={18} /> : <Eye size={18} />}{visible ? 'Ocultar senhas' : 'Mostrar senhas'}</button>{error ? <p className="profile-error" role="alert">{error}</p> : null}<Button type="submit">Simular alteração<Check size={18} /></Button></form></Modal>;
}
