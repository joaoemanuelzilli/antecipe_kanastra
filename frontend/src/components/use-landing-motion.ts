import { useEffect, useRef } from 'react';

/** Motion is progressive enhancement: no stylesheet hides the resting content. */
export function useLandingMotion() {
  const root = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const page = root.current;
    if (!page) return;
    const preference = window.matchMedia('(prefers-reduced-motion: reduce)');
    let dispose = () => {};

    const configure = () => {
      dispose();
      if (preference.matches || !('animate' in Element.prototype) || !('IntersectionObserver' in window)) return;
      const animations = new Set<Animation>();
      const ease = 'cubic-bezier(.16,1,.3,1)';
      const play = (element: Element | null, frames: Keyframe[], duration = 850, delay = 0) => {
        if (!element) return;
        const animation = element.animate(frames, { duration, delay, easing: ease, fill: 'backwards' });
        animations.add(animation);
        animation.onfinish = () => animations.delete(animation);
      };
      const select = (selector: string) => page.querySelector(selector);
      play(select('.lp-hero-photo'), [{ transform: 'scale(1.12)', filter: 'brightness(.7)' }, { transform: 'scale(1)', filter: 'brightness(1)' }], 1500);
      play(select('.lp-hero h1'), [{ clipPath: 'inset(0 0 100% 0)', transform: 'translateY(30px)' }, { clipPath: 'inset(0 0 0% 0)', transform: 'translateY(0)' }], 1000, 100);
      play(select('.lp-hero-content>p'), [{ opacity: 0, transform: 'translateY(16px)' }, { opacity: 1, transform: 'translateY(0)' }], 700, 360);
      play(select('.lp-actions'), [{ opacity: 0, transform: 'translateY(12px)' }, { opacity: 1, transform: 'translateY(0)' }], 650, 480);

      const observer = new IntersectionObserver(entries => {
        for (const entry of entries) {
          if (!entry.isIntersecting) continue;
          const element = entry.target;
          observer.unobserve(element);
          if (element.matches('.lp-photo-panel')) {
            play(element, [{ clipPath: 'inset(5% 3% 5% 3% round 32px)' }, { clipPath: 'inset(0% 0% 0% 0% round 32px)' }], 1100);
            play(element.querySelector('.lp-cover'), [{ transform: 'scale(1.12)' }, { transform: 'scale(1)' }], 1300);
            play(element.querySelector('.lp-lot-preview, .lp-decision-note'), [{ opacity: 0, transform: 'translateY(45px) rotate(-3deg)' }, { opacity: 1, transform: 'translateY(0) rotate(0)' }], 1000, 180);
          } else if (element.matches('.lp-steps')) {
            element.querySelectorAll('li').forEach((step, index) => {
              play(step, [{ opacity: 0, transform: 'translateX(24px)' }, { opacity: 1, transform: 'translateX(0)' }], 700, index * 130);
              play(step.querySelector('.lp-step-icon'), [{ transform: 'scale(.65) rotate(-25deg)' }, { transform: 'scale(1) rotate(0)' }], 800, index * 130);
            });
          } else if (element.matches('.lp-analysis')) {
            play(element.querySelector('.lp-analysis-copy'), [{ opacity: 0, transform: 'translateY(32px)' }, { opacity: 1, transform: 'translateY(0)' }], 950);
            play(element.querySelector('.lp-analysis-status'), [{ opacity: 0, transform: 'translateX(45px)' }, { opacity: 1, transform: 'translateX(0)' }], 850, 200);
          } else if (element.matches('.lp-help-grid')) {
            element.querySelectorAll('article').forEach((card, index) => play(card, [{ opacity: 0, transform: 'perspective(900px) rotateX(12deg) translateY(36px)' }, { opacity: 1, transform: 'perspective(900px) rotateX(0) translateY(0)' }], 1000, index * 120));
          } else if (element.matches('.lp-wordmark')) {
            // Fade first, then illuminate the lettering and settle into the footer.
            play(element, [
              { opacity: .08, filter: 'brightness(1)', offset: 0 },
              { opacity: .9, filter: 'brightness(2)', offset: .45 },
              { opacity: 1, filter: 'brightness(5)', offset: .7 },
              { opacity: .9, filter: 'brightness(2)', offset: 1 },
            ], 2600);
          }
        }
      }, { threshold: .15 });
      page.querySelectorAll('.lp-photo-panel, .lp-steps, .lp-analysis, .lp-help-grid, .lp-wordmark').forEach(element => observer.observe(element));

      // Only the large photographic feature follows scroll; mobile stays still.
      const desktop = window.matchMedia('(min-width: 761px) and (pointer: fine)');
      const feature = page.querySelector<HTMLElement>('.lp-analysis');
      const photo = feature?.querySelector<HTMLElement>('.lp-cover');
      let visible = false;
      let frame = 0;
      const update = () => {
        frame = 0;
        if (!feature || !photo) return;
        if (!desktop.matches || document.hidden || !visible) {
          photo.style.removeProperty('transform');
          return;
        }
        const bounds = feature.getBoundingClientRect();
        const offset = Math.max(-1, Math.min(1, (innerHeight / 2 - bounds.top - bounds.height / 2) / innerHeight));
        photo.style.transform = `scale(1.1) translateY(${offset * 45}px)`;
      };
      const schedule = () => { if (!frame) frame = requestAnimationFrame(update); };
      const visibilityObserver = new IntersectionObserver(entries => { visible = entries[0].isIntersecting; schedule(); });
      if (feature) visibilityObserver.observe(feature);
      const onScroll = () => { if (visible && desktop.matches && !document.hidden) schedule(); };
      const onVisibility = () => {
        for (const animation of animations) {
          if (document.hidden) animation.pause();
          else animation.play();
        }
        schedule();
      };
      window.addEventListener('scroll', onScroll, { passive: true });
      window.addEventListener('resize', schedule, { passive: true });
      document.addEventListener('visibilitychange', onVisibility);
      dispose = () => {
        observer.disconnect();
        visibilityObserver.disconnect();
        animations.forEach(animation => animation.cancel());
        animations.clear();
        cancelAnimationFrame(frame);
        photo?.style.removeProperty('transform');
        window.removeEventListener('scroll', onScroll);
        window.removeEventListener('resize', schedule);
        document.removeEventListener('visibilitychange', onVisibility);
      };
    };
    configure();
    preference.addEventListener('change', configure);
    return () => { dispose(); preference.removeEventListener('change', configure); };
  }, []);

  return root;
}
