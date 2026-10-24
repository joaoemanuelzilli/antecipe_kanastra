import { useEffect, useRef } from 'react';

export function LandingFred() {
  const ref = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    const video = ref.current;
    if (!video) return;
    video.defaultMuted = true;
    video.muted = true;
    video.volume = 0;
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
    let visible = false;
    const update = () => {
      video.muted = true;
      video.volume = 0;
      if (visible && !document.hidden && !reducedMotion.matches) {
        void video.play().catch(() => { /* Keep the poster if playback is unavailable. */ });
      } else {
        video.pause();
        if (reducedMotion.matches) video.currentTime = 0;
      }
    };
    const observer = new IntersectionObserver(entries => {
      visible = entries[0].isIntersecting;
      update();
    });
    observer.observe(video);
    reducedMotion.addEventListener('change', update);
    document.addEventListener('visibilitychange', update);
    return () => {
      observer.disconnect();
      reducedMotion.removeEventListener('change', update);
      document.removeEventListener('visibilitychange', update);
      video.pause();
    };
  }, []);

  return <span className="lp-fred-avatar" role="img" aria-label="Fred, mascote do Antecipe"><video ref={ref} src="/figma/assistant-head-loop.mp4" poster="/figma/assistant-head-poster.jpg" width="50" height="50" muted loop playsInline preload="metadata" tabIndex={-1} aria-hidden="true" /></span>;
}
