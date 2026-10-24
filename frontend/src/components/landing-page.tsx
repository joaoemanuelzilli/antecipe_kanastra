import { useState } from 'react';
import { ArrowRight, ArrowUpRight, Plus } from 'lucide-react';
import Link from '@/lib/link';
import { MOCK_MODE } from '@/api/client';
import { Button } from './ui/button';
import { Modal } from './ui/dialog';
import './landing-page.css';

const assets = '/figma/landing/';
const repository = 'https://github.com/joaoemanuelzilli/antecipe_kanastra';
const steps = [
  { icon: 'upload.svg', title: 'Envie suas notas', copy: 'Importe seu CSV e escolha a data-base da simulação.' },
  { icon: 'result.svg', title: 'Entenda o resultado', copy: 'Veja o valor líquido, os descontos e o motivo de cada recusa.' },
  { icon: 'confirm.svg', title: 'Confirme a cessão', copy: 'Revise as notas aprovadas e confirme quando estiver tudo certo.' },
];
const questions = [
  { title: 'O que preciso para começar?', copy: 'Um arquivo CSV com suas notas fiscais. Na tela de envio, você encontra o modelo e pode escolher a data-base da simulação.' },
  { title: 'Por que uma nota pode não entrar?', copy: 'Cada nota é analisada conforme as regras do fundo. No resultado, abra os detalhes para consultar o motivo da recusa e o próximo passo.' },
  { title: 'A simulação já confirma a cessão?', copy: 'Não. Primeiro você consulta o resultado. A cessão só acontece depois da sua confirmação, com a revisão do valor final e das notas incluídas.' },
];

function IllustrationCard() {
  return <div className="lp-lot-preview" aria-label="Exemplo ilustrativo de resumo de um lote">
    <div className="lp-lot-title"><div><span>Exemplo ilustrativo</span><h3>Pagamentos de dezembro</h3></div><img src={`${assets}caret-right.svg`} width="24" height="24" alt="" /></div>
    <p className="lp-lot-value">R$ 103.264,84</p>
    <div className="lp-lot-tags"><span>30 notas <img src={`${assets}notepad.svg`} width="18" height="18" alt="" /></span><div><span className="lp-approved">19 <img src={`${assets}check.svg`} width="18" height="18" alt="aprovadas" /></span><span className="lp-rejected">11 <img src={`${assets}reject.svg`} width="18" height="18" alt="recusadas" /></span></div></div>
  </div>;
}

export function LandingPage() {
  const [termsOpen, setTermsOpen] = useState(false);
  const entry = '/enviar';
  return <div className="landing-page">
    <a className="lp-skip" href="#conteudo">Ir para o conteúdo</a>
    <header className="lp-header"><Link href="/" aria-label="Antecipe, início"><img src={`${assets}logo.svg`} width="148" height="24" alt="Antecipe" /></Link><nav aria-label="Navegação da apresentação"><a className="lp-repository" aria-label="Ver repositório (abre em nova aba)" href={repository} target="_blank" rel="noreferrer"><span>Ver repositório</span><img src={`${assets}repository.svg`} width="24" height="24" alt="" /><span className="sr-only"> (abre em nova aba)</span></a><Button asChild className="lp-login"><Link href={MOCK_MODE ? '/login' : '/dashboard'}>{MOCK_MODE ? 'Fazer login' : 'Acessar portal'}</Link></Button></nav></header>
    <main id="conteudo">
      <section className="lp-hero-wrap" aria-labelledby="lp-title"><div className="lp-hero"><img className="lp-cover lp-hero-photo" src={`${assets}hero.webp`} alt="Empreendedor em seu negócio, com um celular na mão" width="1376" height="736" fetchPriority="high" /><div className="lp-hero-shade" /><div className="lp-hero-content"><h1 id="lp-title"><span>Antecipe suas notas</span> e entenda cada resultado.</h1><p>Envie seu CSV. Veja quanto você recebe, quais notas foram aceitas e o que fazer com as que não passaram.</p><div className="lp-actions"><Button asChild className="lp-cta"><Link href={entry}>Simular com meu lote<img src={`${assets}arrow-right.svg`} width="24" height="24" alt="" /></Link></Button><a className="lp-secondary" href="#como-funciona">Conhecer o fluxo<img src={`${assets}arrow-up-right.svg`} width="24" height="24" alt="" /></a></div></div></div></section>
      <section className="lp-how lp-section" id="como-funciona" aria-labelledby="lp-how-title"><div className="lp-photo-panel"><img className="lp-cover" src={`${assets}shop.webp`} alt="Empreendedora em sua loja consultando o celular" width="768" height="723" loading="lazy" /><IllustrationCard /></div><div className="lp-how-copy"><div><h2 id="lp-how-title">Do CSV à decisão.<br />Tudo às claras.</h2><p>Uma jornada simples para antecipar suas notas com informação em cada etapa.</p></div><ol className="lp-steps">{steps.map(step => <li key={step.title}><span className="lp-step-icon"><img src={`${assets}${step.icon}`} width="24" height="24" alt="" /></span><div><h3>{step.title}</h3><p>{step.copy}</p></div></li>)}</ol><div className="lp-assurance"><strong>Nada de “recusada” sem explicação.</strong><p>O motivo e o próximo passo acompanham cada nota que não entra.</p></div></div></section>
      <section className="lp-analysis-wrap" aria-labelledby="lp-analysis-title"><div className="lp-analysis lp-section"><img className="lp-cover" src={`${assets}bakery.webp`} alt="Dono de uma padaria acompanhando suas notas pelo celular" width="1200" height="561" loading="lazy" /><div className="lp-analysis-shade" /><div className="lp-analysis-copy"><h2 id="lp-analysis-title"><span>Seu negócio anda.</span><br />Você acompanha.</h2><p>Do envio ao resultado, acompanhe o andamento da análise no próprio portal.</p></div><div className="lp-analysis-status"><img src={`${assets}confirm.svg`} width="24" height="24" alt="" /><div><strong>Cada etapa à vista.</strong><p>Envio, análise e resultado em um só lugar.</p></div></div></div></section>
      <section className="lp-clarity lp-section" aria-labelledby="lp-clarity-title"><div className="lp-clarity-copy"><h2 id="lp-clarity-title">Você vê os detalhes.<br />Você decide.</h2><p>Saiba o que compõe o valor da sua antecipação antes de confirmar.</p><ul><li><img src={`${assets}check.svg`} width="18" height="18" alt="" />Valor líquido e descontos por nota</li><li><img src={`${assets}check.svg`} width="18" height="18" alt="" />Motivos de recusa e orientações</li><li><img src={`${assets}check.svg`} width="18" height="18" alt="" />Resultado disponível para baixar em CSV</li></ul><Link href="/regras" className="lp-text-link">Conhecer as regras do fundo<ArrowUpRight size={20} /></Link></div><div className="lp-photo-panel lp-market"><img className="lp-cover" src={`${assets}market.webp`} alt="Comerciantes conversando no balcão de um mercado" width="768" height="717" loading="lazy" /><div className="lp-decision-note"><img src={`${assets}confirm.svg`} width="24" height="24" alt="" /><div><strong>A última palavra é sua.</strong><p>A simulação vem antes da confirmação.</p></div></div></div></section>
      <section className="lp-help" aria-labelledby="lp-help-title"><div className="lp-section"><div className="lp-help-heading"><h2 id="lp-help-title">Dúvidas? O Fred te ajuda.</h2><span>Seu guia no portal<img src="/figma/assistant-head-poster.jpg" width="50" height="50" alt="Fred, mascote do Antecipe" loading="lazy" /></span></div><div className="lp-help-grid">{questions.map((question, index) => <article key={question.title}><img className="lp-fred-photo" src={`${assets}${['fred-help', 'fred-fast', 'fred-guide'][index]}.webp`} alt={['Fred sentado à mesa, pronto para ajudar', 'Fred voando para acompanhar a análise', 'Fred no mercado, orientando o próximo passo'][index]} width="379" height="377" loading="lazy" /><details><summary>{question.title}<Plus size={20} aria-hidden="true" /></summary><p>{question.copy}</p></details></article>)}</div><div className="lp-final-cta"><div><h2>Pronto para ver com clareza?</h2><p>{MOCK_MODE ? 'Explore a demonstração com dados fictícios.' : 'Comece pela simulação do seu lote.'}</p></div><Button asChild className="lp-cta"><Link href={entry}>{MOCK_MODE ? 'Explorar demonstração' : 'Simular meu lote'}<ArrowRight size={24} /></Link></Button></div></div></section>
    </main>
    <footer className="lp-footer"><img className="lp-wordmark" src={`${assets}wordmark.svg`} alt="Antecipe" loading="lazy" /><div className="lp-footer-bottom"><a href={repository} target="_blank" rel="noreferrer" aria-label="Repositório do projeto Antecipe Kanastra (abre em nova aba)"><img src={`${assets}kanastra.svg`} width="92" height="16" alt="Kanastra" /></a><span>{MOCK_MODE ? 'Demonstração · Dados fictícios' : 'Antecipação com clareza'}</span><div><button type="button" onClick={() => setTermsOpen(true)}>Sobre o uso</button><a href="#como-funciona">Saiba mais</a></div></div></footer>
    <Modal open={termsOpen} onOpenChange={setTermsOpen} title="Sobre o uso do Antecipe" description="Entenda a simulação antes de confirmar uma cessão." footer={<Button onClick={() => setTermsOpen(false)}>Entendi</Button>}><p>A simulação apresenta os valores e as decisões conforme as regras do fundo. Revise o resultado e os detalhes das notas antes de confirmar a cessão.</p>{MOCK_MODE && <p className="lp-terms-copy">Neste ambiente de demonstração, os dados são fictícios e as confirmações não realizam operações financeiras. Nenhum cadastro ou senha real é necessário.</p>}</Modal>
  </div>;
}
