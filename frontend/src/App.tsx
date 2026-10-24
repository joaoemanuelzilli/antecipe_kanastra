import { useEffect } from 'react';
import { BrowserRouter, Navigate, Routes, Route, useParams, useLocation } from 'react-router-dom';
import { LandingPage } from '@/components/landing-page';
import { Shell } from '@/components/shell';
import { DashboardPage } from '@/components/dashboard-page';
import { DemoLoginPage, isDemoAccessGranted } from '@/components/demo-login-page';
import { UploadPage } from '@/components/upload-page';
import { ResultPage } from '@/components/result-page';
import { LotsPage } from '@/components/lots-page';
import { RulesPage } from '@/components/rules-page';
import { ManagerPage } from '@/components/manager-page';
import { PageHeader } from '@/components/page-header';
import { MOCK_MODE } from '@/api/client';
import Link from '@/lib/link';
function RouteScroll() { const { pathname } = useLocation(); useEffect(() => { window.scrollTo({ top: 0, behavior: 'instant' }); }, [pathname]); return null; }
function ResultRoute() { const { id } = useParams(); return <ResultPage key={id} id={id ?? ''} />; }
function AppContent() {
  const { pathname } = useLocation();
  if (pathname === '/') return <LandingPage />;
  if (MOCK_MODE && pathname === '/login') return <DemoLoginPage />;
  if (MOCK_MODE && !isDemoAccessGranted()) return <Navigate to="/login" state={{ from: pathname }} replace />;
  return <Shell><Routes><Route path="/dashboard" element={<DashboardPage />} /><Route path="/enviar" element={<><DashboardPage /><UploadPage /></>} /><Route path="/lotes" element={<LotsPage />} /><Route path="/lote/:id" element={<ResultRoute />} /><Route path="/regras" element={<RulesPage />} /><Route path="/gestor" element={<ManagerPage />} /><Route path="/login" element={<Navigate to="/dashboard" replace />} /><Route path="*" element={<main className="container page-content"><PageHeader title="Página não encontrada" description="Não encontramos o endereço que você abriu." /><div className="empty-state"><Link href="/dashboard" className="button button-primary">Voltar ao início</Link></div></main>} /></Routes></Shell>;
}
export default function App() { return <BrowserRouter><RouteScroll /><AppContent /></BrowserRouter>; }
