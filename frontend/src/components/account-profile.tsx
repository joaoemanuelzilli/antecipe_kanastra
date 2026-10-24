import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { Building2, LogOut, Monitor, ShieldCheck, UserRound } from 'lucide-react';
import { logoutAccount, useAccount } from '@/api/auth';
import { clearGestorSenha } from '@/api/client';
import { saveDemoProfile, useDemoProfile } from '@/lib/demo-profile';
import { PageHeader } from './page-header';
import { Switch } from './ui/switch';
import { Button } from './ui/button';
import { Modal } from './ui/dialog';
import Link from '@/lib/link';
import './profile-page.css';

export function AccountProfile() {
  const account = useAccount().data!;
  const preferences = useDemoProfile();
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [confirm, setConfirm] = useState(false);
  const client = useQueryClient();
  const navigate = useNavigate();
  async function logout() {
    setBusy(true); setError('');
    try { await logoutAccount(); clearGestorSenha(); client.clear(); navigate('/login', { replace: true }); }
    catch (cause) { setError(cause instanceof Error ? cause.message : 'Não foi possível sair. Tente novamente'); }
    finally { setBusy(false); }
  }
  const preference = (key: 'confirmLogout' | 'reducedMotion', value: boolean) => {
    if (!saveDemoProfile({ ...preferences, [key]: value })) setError('Não foi possível salvar a preferência neste navegador');
  };
  const initials = account.nome.trim().split(/\s+/).slice(0, 2).map(part => part[0]).join('').toLocaleUpperCase('pt-BR');
  return <main className="container page-content profile-page"><PageHeader title="Meu perfil" description="Seus dados e preferências, do seu jeito" /><section className="profile-identity"><span className="profile-avatar" aria-hidden="true">{initials}</span><div className="profile-identity-copy"><h2>{account.nome}</h2><p>{account.email}</p><span className="profile-role">{account.perfil === 'gestor' ? 'Gestor do fundo' : 'Conta da empresa'}</span></div></section>{error && <p className="profile-error" role="alert">{error}</p>}<div className="profile-grid"><div className="profile-main"><section className="profile-panel"><header className="profile-section-heading"><span className="profile-section-icon"><UserRound size={21} /></span><div><h2>Informações pessoais</h2><p>Dados vinculados à sua conta</p></div></header><div className="profile-fields"><label className="profile-field"><span>Nome completo</span><input className="profile-input" value={account.nome} readOnly /></label><label className="profile-field"><span>E-mail</span><input className="profile-input" value={account.email} readOnly /></label>{account.empresa_nome && <label className="profile-field"><span>Empresa</span><input className="profile-input" value={account.empresa_nome} readOnly /></label>}{account.empresa_cnpj && <label className="profile-field"><span>CNPJ</span><input className="profile-input" value={account.empresa_cnpj} readOnly /></label>}</div><p className="profile-small-note">A edição dos dados e a troca de senha ainda não estão disponíveis</p></section><section className="profile-panel"><header className="profile-section-heading"><span className="profile-section-icon"><ShieldCheck size={21} /></span><div><h2>Preferências</h2><p>Salvas apenas neste navegador</p></div></header><div className="profile-preference"><div><h3>Reduzir animações</h3><p>Reduza os movimentos e pause os vídeos do Fred</p></div><Switch checked={preferences.reducedMotion} onCheckedChange={value => preference('reducedMotion', value)} label="Reduzir animações" /></div><div className="profile-preference"><div><h3>Confirmar antes de sair</h3><p>Peça confirmação para encerrar a sessão</p></div><Switch checked={preferences.confirmLogout} onCheckedChange={value => preference('confirmLogout', value)} label="Confirmar antes de sair" /></div></section></div><aside className="profile-aside"><section className="profile-panel"><header className="profile-section-heading"><span className="profile-section-icon"><Monitor size={21} /></span><div><h2>Sessão atual</h2><p>Conectado com sua conta</p></div></header><div className="profile-device"><span>Navegador atual</span><span className="profile-device-badge">Em uso</span></div><button className="profile-logout" type="button" disabled={busy} onClick={() => preferences.confirmLogout ? setConfirm(true) : void logout()}><LogOut size={18} />{busy ? 'Saindo…' : 'Sair da conta'}</button></section><section className="profile-tip"><Building2 size={23} /><div><h2>Tudo da sua empresa, por aqui</h2><p>Acompanhe seus envios e consulte cada resultado</p><Link href="/lotes">Ver meus lotes</Link></div></section></aside></div><Modal open={confirm} onOpenChange={value => { if (!busy) setConfirm(value); }} title="Sair da sua conta?" description="Sua sessão será encerrada neste navegador" footer={<><Button variant="ghost" disabled={busy} onClick={() => setConfirm(false)}>Continuar por aqui</Button><Button disabled={busy} onClick={() => { void logout(); }}>{busy ? 'Saindo…' : 'Sair da conta'}</Button></>}>{error && <p role="alert">{error}</p>}</Modal></main>;
}
