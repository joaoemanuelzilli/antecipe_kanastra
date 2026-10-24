import { useEffect, useRef, useState } from 'react';
import { Play } from 'lucide-react';

export function LandingVideo() {
  const video = useRef<HTMLVideoElement>(null);
  const [started, setStarted] = useState(false);
  const [error, setError] = useState(false);

  useEffect(() => {
    const player = video.current;
    if (!player || !('IntersectionObserver' in window)) return;
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
    let visible = false;
    let disposed = false;
    player.defaultMuted = true;
    player.muted = true;
    const update = () => {
      if (!visible || document.hidden || reducedMotion.matches) {
        player.pause();
        return;
      }
      if (player.ended) return;
      void player.play().then(() => {
        // A pending play request must not restart a video after leaving the section.
        if (disposed || !visible || document.hidden || reducedMotion.matches) player.pause();
      }).catch(() => { /* Manual play remains available if autoplay is blocked. */ });
    };
    const observer = new IntersectionObserver(entries => {
      const entry = entries[0];
      visible = entry.isIntersecting && entry.intersectionRatio >= .35;
      update();
    }, { threshold: [0, .35], rootMargin: '-84px 0px 0px 0px' });
    observer.observe(player);
    document.addEventListener('visibilitychange', update);
    reducedMotion.addEventListener('change', update);
    return () => {
      disposed = true;
      observer.disconnect();
      document.removeEventListener('visibilitychange', update);
      reducedMotion.removeEventListener('change', update);
      player.pause();
    };
  }, []);

  async function play() {
    try {
      await video.current?.play();
      video.current?.focus({ preventScroll: true });
    } catch {
      // Keep native controls available if the browser declines playback.
      setStarted(true);
    }
  }

  return <section className="lp-video-section lp-section" id="apresentacao" aria-labelledby="lp-video-title">
    <div className="lp-video-heading"><h2 id="lp-video-title">Veja o Antecipe em ação</h2><p>Do envio à decisão, em menos de um minuto</p></div>
    <div className="lp-video-player">
      <video ref={video} controls muted playsInline preload="none" poster="/media/antecipe-apresentacao.jpg" width="1920" height="1080" aria-label="Apresentação do Antecipe" aria-describedby="lp-video-caption" onPlay={() => setStarted(true)} onError={() => setError(true)}>
        <source src="/media/antecipe-apresentacao.mp4" type="video/mp4" />
        Seu navegador não reproduz este vídeo — <a href="/media/antecipe-apresentacao.mp4">Abrir apresentação</a>
      </video>
      {!started && !error && <button type="button" className="lp-video-play" onClick={() => { void play(); }} aria-label="Reproduzir apresentação do Antecipe"><Play size={28} fill="currentColor" aria-hidden="true" /><span>Assistir vídeo <small>0:49</small></span></button>}
    </div>
    <p className="lp-video-caption" id="lp-video-caption">Apresentação com dados ilustrativos</p>
    {error && <p className="lp-video-error" role="alert">Não foi possível reproduzir o vídeo — <a href="/media/antecipe-apresentacao.mp4" download>Baixar para assistir</a></p>}
  </section>;
}
