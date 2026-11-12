export function installLandingMotion(scene: HTMLElement) {
  const media = matchMedia(
    "(max-width: 760px) and (min-height: 501px) and (prefers-reduced-motion: no-preference)",
  );
  const image = scene.querySelector<HTMLElement>(".hero-image");
  const captions = [...scene.querySelectorAll<HTMLElement>(".hero-toolbar, .hero-photo-caption")];
  if (!image) return () => {};
  const originals = [image, ...captions].map((element) => ({
    element,
    style: element.getAttribute("style"),
  }));
  let frame = 0,
    progress = 0,
    previousTime = 0,
    disposed = false;
  const clamp = (value: number) => Math.max(0, Math.min(1, value));
  function restore() {
    for (const { element, style } of originals) {
      if (style === null) element.removeAttribute("style");
      else element.setAttribute("style", style);
    }
  }
  function render(time: number) {
    frame = 0;
    if (disposed || !media.matches) return;
    const target = clamp(-scene.getBoundingClientRect().top / (innerHeight * 0.55));
    const elapsed = previousTime ? Math.min(time - previousTime, 64) : 16;
    previousTime = time;
    progress += (target - progress) * (1 - Math.exp(-elapsed / 100));
    if (Math.abs(target - progress) < 0.0001) progress = target;
    image!.style.transform = `translateY(${-0.5 * progress}%) scale(${1.08 - 0.06 * progress})`;
    const fade = clamp(progress / 0.3);
    for (const caption of captions) {
      caption.style.opacity = String(1 - fade);
      caption.style.visibility = fade === 1 ? "hidden" : "visible";
      caption.style.transform = `translateY(${-12 * fade}px)`;
    }
    if (target !== progress) frame = requestAnimationFrame(render);
  }
  function schedule() {
    if (!frame && media.matches) frame = requestAnimationFrame(render);
  }
  function changeMedia() {
    cancelAnimationFrame(frame);
    frame = 0;
    previousTime = 0;
    restore();
    schedule();
  }
  addEventListener("scroll", schedule, { passive: true });
  addEventListener("resize", schedule, { passive: true });
  media.addEventListener("change", changeMedia);
  schedule();
  return () => {
    disposed = true;
    cancelAnimationFrame(frame);
    removeEventListener("scroll", schedule);
    removeEventListener("resize", schedule);
    media.removeEventListener("change", changeMedia);
    restore();
  };
}
