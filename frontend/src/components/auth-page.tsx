import { useState, type FormEvent } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { Eye, EyeOff } from 'lucide-react';
import { ACCOUNT_QUERY, loginAccount, registerAccount, type Registration } from '@/api/auth';
import { ApiError, MOCK_MODE } from '@/api/client';
import Link from '@/lib/link';
import { DEFAULT_DEMO_PROFILE, saveDemoProfile } from '@/lib/demo-profile';
import './demo-login.css';
import './auth-page.css';

export function AuthPage({ register = false }: { register?: boolean }) {
  const [step, setStep] = useState(1);
  const [data, setData] = useState<Registration>({ nome: '', email: '', senha: '', empresa_nome: '', empresa_cnpj: '' });
  const [confirmation, setConfirmation] = useState('');
  const [visible, setVisible] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const navigate = useNavigate();
  const location = useLocation();
  const queryClient = useQueryClient();
  const destination = ['/dashboard', '/enviar', '/perfil', '/lotes', '/gestor'].includes(location.state?.from) ? location.state.from as string : '/dashboard';
  const update = (key: keyof Registration, value: string) => { setData(old => ({ ...old, [key]: value })); setFieldErrors(old => ({ ...old, [key]: '' })); setError(''); };
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy || !event.currentTarget.reportValidity()) return;
    if (register && step === 1) {
      if (data.senha !== confirmation) { setFieldErrors({ confirmation: 'As senhas precisam ser iguais' }); return; }
      if (new TextEncoder().encode(data.senha).length > 72) { setFieldErrors({ senha: 'A senha deve ter no máximo 72 bytes' }); return; }
      setStep(2); setFieldErrors({}); return;
    }
    setBusy(true); setError(''); setFieldErrors({});
    try {
      if (MOCK_MODE) {
        saveDemoProfile({ ...DEFAULT_DEMO_PROFILE, name: data.nome.trim(), email: data.email.trim(), company: data.empresa_nome.trim() });
        sessionStorage.setItem('antecipe.demo.access.v1', 'true');
        setData(old => ({ ...old, senha: '' })); setConfirmation('');
        navigate(destination, { replace: true }); return;
      }
      const account = register ? await registerAccount({ ...data, nome: data.nome.trim(), email: data.email.trim(), empresa_nome: data.empresa_nome.trim() }) : await loginAccount(data.email.trim(), data.senha);
      queryClient.clear(); queryClient.setQueryData(ACCOUNT_QUERY, account);
      setData(old => ({ ...old, senha: '' })); setConfirmation('');
      navigate(account.perfil === 'gestor' ? '/gestor' : destination === '/gestor' ? '/dashboard' : destination, { replace: true });
    } catch (cause) {
      if (cause instanceof ApiError && cause.erro.detalhes?.length) {
        const errors = Object.fromEntries(cause.erro.detalhes.map(item => [item.campo, item.problema]));
        setFieldErrors(errors);
        if (errors.nome || errors.email || errors.senha) setStep(1);
      }
      setError(cause instanceof Error ? cause.message : 'Não foi possível concluir. Tente novamente');
    } finally { setBusy(false); }
  }
  const field = (key: keyof Registration, label: string, type = 'text', extra = {}) => <label className="auth-field"><span>{label}</span><input className="demo-login-input" type={type} value={data[key]} required disabled={busy} onChange={event => update(key, event.target.value)} aria-invalid={!!fieldErrors[key]} aria-describedby={fieldErrors[key] ? `auth-${key}-error` : undefined} {...extra} />{fieldErrors[key] && <small id={`auth-${key}-error`} role="alert">{fieldErrors[key]}</small>}</label>;
  return <main className={`demo-login-page auth-page${register ? ' auth-page-register' : ''}`}><img className="demo-login-background" src="/figma/login-background.png" alt="" /><Link href="/" className="demo-login-logo" aria-label="Antecipe — voltar à página inicial"><img src="/figma/logo-white.svg" alt="Antecipe" width={296} height={48} /></Link><div className="demo-login-stack"><section className="demo-login-card"><header className="demo-login-intro"><h1>{register ? step === 1 ? 'Crie sua conta' : 'Agora, sua empresa' : 'Que bom te ver aqui!'}</h1><p>{register ? step === 1 ? 'Seus dados de acesso para começar' : 'Precisamos destes dados para concluir o cadastro' : 'Entre na sua conta do Antecipe'}</p></header><form className="demo-login-form" onSubmit={submit}>
    {MOCK_MODE && <p className="demo-login-notice">Demonstração · Use dados fictícios, nenhuma conta real será criada</p>}
    {register && <ol className="auth-progress" aria-label={`Etapa ${step} de 2`}><li aria-current={step === 1 ? 'step' : undefined} data-complete={step > 1}><span>1</span>Seus dados</li><li aria-current={step === 2 ? 'step' : undefined}><span>2</span>Sua empresa</li></ol>}
    <div className="auth-fields">{step === 1 ? <>{register && field('nome', 'Nome completo', 'text', { minLength: 2, maxLength: 100, autoComplete: 'name' })}{field('email', 'E-mail', 'email', { maxLength: 200, autoComplete: 'email' })}<div className="auth-password">{field('senha', 'Senha', visible ? 'text' : 'password', { minLength: register ? 8 : undefined, maxLength: 72, autoComplete: register ? 'new-password' : 'current-password' })}<button type="button" className="auth-eye" aria-label={visible ? 'Ocultar senhas' : 'Mostrar senhas'} aria-pressed={visible} onClick={() => setVisible(old => !old)}>{visible ? <EyeOff size={20} /> : <Eye size={20} />}</button></div>{register && <><p className="auth-hint">Use de 8 a 72 caracteres</p><label className="auth-field"><span>Repetir senha</span><input className="demo-login-input" type={visible ? 'text' : 'password'} autoComplete="new-password" required value={confirmation} disabled={busy} onChange={event => { setConfirmation(event.target.value); setFieldErrors(old => ({ ...old, confirmation: '' })); }} aria-invalid={!!fieldErrors.confirmation} aria-describedby={fieldErrors.confirmation ? 'auth-confirmation-error' : undefined} />{fieldErrors.confirmation && <small id="auth-confirmation-error" role="alert">{fieldErrors.confirmation}</small>}</label></>}</> : <>{field('empresa_nome', 'Nome da empresa', 'text', { minLength: 2, maxLength: 150, autoComplete: 'organization' })}{field('empresa_cnpj', 'CNPJ', 'text', { maxLength: 18, inputMode: 'numeric', placeholder: '00.000.000/0000-00' })}</>}</div>
    {error && <p className="demo-login-error" role="alert">{error}</p>}
    <button className="demo-login-submit" type="submit" disabled={busy}>{busy ? 'Aguarde…' : register ? step === 1 ? 'Continuar' : MOCK_MODE ? 'Explorar com estes dados' : 'Criar conta' : 'Entrar'}</button>
    {register && step === 2 && <button className="auth-back" type="button" disabled={busy} onClick={() => { setStep(1); setError(''); }}>Voltar aos meus dados</button>}
    <div className="auth-footer"><p className="auth-switch">{register ? 'Já tem uma conta? ' : 'Ainda não tem conta? '}<Link href={register ? '/login' : '/cadastro'} state={location.state}>{register ? 'Entrar' : 'Criar conta'}</Link></p></div>
  </form></section></div></main>;
}
