import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { QueryClient,QueryClientProvider } from '@tanstack/react-query';
import '@fontsource/parkinsans/latin-400.css';
import '@fontsource/parkinsans/latin-500.css';
import '@fontsource/parkinsans/latin-600.css';
import '@fontsource/parkinsans/latin-700.css';
import '@fontsource/parkinsans/latin-800.css';
import './styles.css';
import './figma-theme.css';
import './app-layout.css';
import App from './App';
import './mobile.css';
const queryClient=new QueryClient({defaultOptions:{queries:{retry:1,refetchOnWindowFocus:false,staleTime:1000}}});
async function start(){if(import.meta.env.VITE_MOCK_API==='true'){const {startMockWorker}=await import('./mocks/browser');await startMockWorker();}const container=document.getElementById('root');if(!container)throw new Error('Não foi possível iniciar a interface.');createRoot(container).render(<StrictMode><QueryClientProvider client={queryClient}><App/></QueryClientProvider></StrictMode>);}
start().catch(()=>{const root=document.getElementById('root');if(root)root.innerHTML='<main style="padding:40px;font-family:Parkinsans,sans-serif"><h1>Não foi possível iniciar a demonstração</h1><p>Atualize a página. Se o problema continuar, verifique se o navegador permite o serviço de demonstração local.</p></main>';});
