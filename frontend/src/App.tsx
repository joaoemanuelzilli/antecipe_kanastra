import { AuthPage } from '@/components/auth-page';
import { useAccount } from '@/api/auth';
import { Loading } from '@/components/feedback';
import { Notifications } from '@/components/notifications';
import { useEffect } from 'react';
import { BrowserRouter, Navigate, Routes, Route, useParams, useLocation } from 'react-router-dom';
import { LandingPage } from '@/components/landing-page';
import { Shell } from '@/components/shell';
import { DashboardPage } from '@/components/dashboard-page';
import { DemoLoginPage, isDemoAccessGranted } from '@/components/demo-login-page';
import { UploadPage } from '@/components/upload-page';
import { ResultPage } from '@/components/result-page';
import { LotsPage } from '@/components/lots-page';
import { ProfilePage } from '@/components/profile-page';
import { ManagerPage } from '@/components/manager-page';
import { PageHeader } from '@/components/page-header';
import { MOCK_MODE } from '@/api/client';
import Link from '@/lib/link';
function RouteScroll() { const { pathname } = useLocation(); useEffect(() => { window.scrollTo({ top: 0, behavior: 'instant' }); }, [pathname]); return null; }
function ResultRoute() { const { id } = useParams(); return <ResultPage key={id} id={id ?? ''} />; }
function ConnectedPortal({ children }: { children: React.ReactNode }) {
  const account = useAccount();
  const location = useLocation();
  if (account.isPending) return <main className="container"><Loading label="Conferindo sua sessão…" /></main>;
  if (account.error) return <main className="container page-content"><h1>Não foi possível conferir seu acesso</h1><p>Verifique a conexão com a API e tente novamente</p><button className="button button-primary" onClick={() => { void account.refetch(); }}>Tentar novamente</button></main>;
  if (!account.data) return <Navigate to="/login" state={{ from: location.pathname }} replace />;
  if (location.pathname === '/gestor' && account.data.perfil !== 'gestor') return <Navigate to="/dashboard" replace />;
  return children;
}
function AppContent() {
  const { pathname } = useLocation();
  if (pathname === '/') return <LandingPage />;
  if (pathname === '/cadastro') return <AuthPage key="register" register />;
  if (!MOCK_MODE && pathname === '/login') return <AuthPage key="login" />;
  if (MOCK_MODE && pathname === '/login') return <DemoLoginPage />;
  if (MOCK_MODE && !isDemoAccessGranted()) return <Navigate to="/login" state={{ from: pathname }} replace />;
  const portal = <Shell><Routes><Route path="/dashboard" element={<DashboardPage />} /><Route path="/enviar" element={<><DashboardPage /><UploadPage /></>} /><Route path="/lotes" element={<LotsPage />} /><Route path="/lote/:id" element={<ResultRoute />} /><Route path="/perfil" element={<ProfilePage />} /><Route path="/regras" element={<Navigate to="/perfil" replace />} /><Route path="/gestor" element={<ManagerPage />} /><Route path="/login" element={<Navigate to="/dashboard" replace />} /><Route path="*" element={<main className="container page-content"><PageHeader title="Página não encontrada" description="Não encontramos o endereço que você abriu." /><div className="empty-state"><Link href="/dashboard" className="button button-primary">Voltar ao início</Link></div></main>} /></Routes></Shell>;
  return MOCK_MODE ? portal : <ConnectedPortal>{portal}</ConnectedPortal>;
}
export default function App() { return <BrowserRouter><RouteScroll /><AppContent /><Notifications /></BrowserRouter>; }
