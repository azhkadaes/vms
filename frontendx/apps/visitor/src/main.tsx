import {
  StrictMode,
  Suspense,
  lazy,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type MouseEvent,
  type RefObject,
} from "react";
import { createRoot } from "react-dom/client";
import {
  BrowserRouter,
  Link,
  NavLink,
  Route,
  Routes,
  useLocation,
  useNavigate,
} from "react-router";
import {
  ArrowUpRight,
  ArrowRight,
  CalendarDays,
  ShieldCheck,
  Check,
  MapPin,
  Clock3,
  ArrowDown,
  ArrowUp,
  ArrowLeft,
} from "lucide-react";
import { DataProvider } from "@vms/data/query";
import { AppBoundary, Brand, Loading, PreviewNotice, RouteFocus } from "@vms/ui/shell";
import hero from "../../../packages/ui/src/assets/nusantara.jpg";
import { heroAvifSrcSet, heroWebpSrcSet } from "./features/hero-image";
import { heroImageSizes } from "./hero-image-settings";
import { LanguageProvider, LanguageSwitch, useLanguage } from "./i18n";
import "@vms/ui/styles.css";
import "./styles.css";
const Booking = lazy(() => import("./pages/Booking"));
const Status = lazy(() => import("./pages/Status"));
type LandingView = "registration" | "status" | null;
type LandingLinks = { registration: string; status: string };
type SheetOrigin = { top: number; width: number; height: number; scroll: number };
type ChooseAction = (event: MouseEvent<HTMLAnchorElement>) => void;

function useMobileLayout() {
  const [mobile, setMobile] = useState(() => window.matchMedia("(max-width: 760px)").matches);
  useEffect(() => {
    const media = window.matchMedia("(max-width: 760px)");
    const update = () => setMobile(media.matches);
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, []);
  return mobile;
}
function App() {
  const { t } = useLanguage();
  const location = useLocation();
  const isLanding = location.pathname === "/";
  const mobile = useMobileLayout();
  const requestedView = new URLSearchParams(location.search).get("view");
  const selectedView: LandingView =
    requestedView === "registration" || requestedView === "status" ? requestedView : null;
  const choiceClosed = isLanding && mobile && !selectedView;
  const taskLocked = isLanding && mobile && selectedView !== null;
  const sheetOrigin = useRef<SheetOrigin | null>(null);
  const rememberSheetPosition: ChooseAction = (event) => {
    if (
      !mobile ||
      !isLanding ||
      selectedView ||
      event.button !== 0 ||
      event.metaKey ||
      event.ctrlKey ||
      event.shiftKey ||
      event.altKey
    )
      return;
    const sheet = document.getElementById("visitor-sheet");
    if (sheet)
      sheetOrigin.current = {
        top: sheet.getBoundingClientRect().top,
        width: window.innerWidth,
        height: window.innerHeight,
        scroll: window.scrollY,
      };
  };
  const links: LandingLinks = {
    registration: mobile || selectedView ? "/?view=registration#registration" : "#registration",
    status: (mobile || selectedView) && isLanding ? "/?view=status#booking-status" : "/status",
  };
  const headerThreshold = useRef<HTMLSpanElement>(null);
  const [headerVisible, setHeaderVisible] = useState(false);
  useLayoutEffect(() => {
    if (!taskLocked) return;
    const body = document.body;
    const scroll = window.scrollY;
    const previous = {
      position: body.style.position,
      top: body.style.top,
      left: body.style.left,
      right: body.style.right,
      width: body.style.width,
      overflow: body.style.overflow,
    };
    Object.assign(body.style, {
      position: "fixed",
      top: `${-scroll}px`,
      left: "0",
      right: "0",
      width: "100%",
      overflow: "hidden",
    });
    return () => {
      Object.assign(body.style, previous);
      window.scrollTo({ top: scroll, behavior: "instant" });
    };
  }, [taskLocked]);
  useEffect(() => {
    const threshold = headerThreshold.current;
    if (!isLanding || !threshold) return;
    const observer = new IntersectionObserver(([entry]) => {
      setHeaderVisible(entry.boundingClientRect.bottom <= 0);
    });
    observer.observe(threshold);
    return () => observer.disconnect();
  }, [isLanding]);
  return (
    <div
      className={`visitor-app${isLanding ? " visitor-app--landing" : ""}${taskLocked ? " visitor-app--task" : ""}`}
    >
      {isLanding && (
        <span ref={headerThreshold} className="header-reveal-threshold" aria-hidden="true" />
      )}
      <a className="skip-link" href="#content">
        {t("Lewati ke konten", "Skip to content")}
      </a>
      <header
        className="visitor-header glass"
        data-revealed={taskLocked || headerVisible}
        inert={taskLocked}
        aria-hidden={taskLocked || undefined}
      >
        <div className="visitor-header__inner">
          <Brand linked={!taskLocked} />
          {!mobile && (
            <nav aria-label={t("Navigasi pengunjung", "Visitor navigation")}>
              {location.pathname !== "/status" && (
                <NavLink to="/status">{t("Cek status", "Check status")}</NavLink>
              )}
              {!location.pathname.startsWith("/booking") && (
                <Link
                  className="link-button primary"
                  aria-label={t("Daftar kunjungan", "Request a visit")}
                  to={isLanding ? links.registration : "/booking"}
                  onClick={isLanding ? rememberSheetPosition : undefined}
                >
                  <span className="visitor-nav-label--full">
                    {t("Daftar kunjungan", "Request a visit")}
                  </span>
                  <span className="visitor-nav-label--compact">{t("Daftar", "Request")}</span>
                </Link>
              )}
            </nav>
          )}
          <LanguageSwitch className="language-switch--header" compact={!isLanding} />
        </div>
      </header>
      <PreviewNotice />
      <main id="content" tabIndex={-1}>
        <Suspense fallback={<Loading />}>
          <Routes>
            <Route
              element={
                <VisitPage
                  mobile={mobile}
                  selectedView={selectedView}
                  links={links}
                  sheetOrigin={sheetOrigin}
                  onChoose={rememberSheetPosition}
                />
              }
            >
              <Route index element={null} />
              <Route path="booking/*" element={null} />
            </Route>
            <Route path="status" element={<Status />} />
            <Route
              path="*"
              element={
                <div className="empty">
                  <h1>{t("Halaman tidak ditemukan", "Page not found")}</h1>
                  <Link className="text-link" to="/">
                    {t("Kembali ke beranda", "Back to home")}
                  </Link>
                </div>
              }
            />
          </Routes>
        </Suspense>
      </main>
      <footer className="visitor-footer" hidden={choiceClosed} inert={taskLocked}>
        <div>
          <Brand />
          <p>
            {t("Terhubung untuk melayani.", "Connected to serve.")}
            <br />
            {t("Selamat datang di Ibu Kota Nusantara.", "Welcome to Nusantara Capital City.")}
          </p>
        </div>
        <div>
          <strong>Otorita Ibu Kota Nusantara</strong>
          <p>
            {t("Kawasan Inti Pusat Pemerintahan", "Central Government District")}
            <br />
            Nusantara, Kalimantan Timur
          </p>
          <a href="/admin/" className="text-link">
            {t("Akses pegawai", "Staff access")} <ArrowUpRight size={14} />
          </a>
        </div>
        <small>© {new Date().getFullYear()} Otorita Ibu Kota Nusantara</small>
      </footer>
    </div>
  );
}
// One route layout keeps the same form instance when moving from home to a step.
function VisitPage({
  mobile,
  selectedView,
  links,
  sheetOrigin,
  onChoose,
}: {
  mobile: boolean;
  selectedView: LandingView;
  links: LandingLinks;
  sheetOrigin: RefObject<SheetOrigin | null>;
  onChoose: ChooseAction;
}) {
  const location = useLocation();
  const navigate = useNavigate();
  const isLanding = location.pathname === "/";
  const sheet = useRef<HTMLDivElement>(null);
  const sheetAnimation = useRef<Animation | null>(null);
  const returnStarted = useRef(false);
  const wasExpanded = useRef(false);
  const [returningKey, setReturningKey] = useState<string | null>(null);
  const returning = mobile && isLanding && selectedView !== null && returningKey === location.key;
  const choiceClosed = isLanding && mobile && !selectedView;
  const registrationVisible =
    !isLanding || selectedView === "registration" || (!mobile && selectedView !== "status");
  const [registrationMounted, setRegistrationMounted] = useState(registrationVisible);
  useEffect(() => {
    if (registrationVisible) setRegistrationMounted(true);
  }, [registrationVisible]);
  const statusVisible = isLanding && selectedView === "status";
  function originTop() {
    const saved = sheetOrigin.current;
    if (saved && saved.width === window.innerWidth && saved.height === window.innerHeight)
      return saved.top;
    const home = sheet.current?.parentElement;
    const photo = home?.querySelector<HTMLElement>(".public-hero");
    return (
      (home?.getBoundingClientRect().top ?? 0) +
      (photo?.offsetHeight ?? window.innerHeight * 0.82) -
      84
    );
  }
  useLayoutEffect(() => {
    setReturningKey(null);
    returnStarted.current = false;
    const motion = window.matchMedia("(prefers-reduced-motion: reduce)");
    const expanded = mobile && isLanding && selectedView !== null;
    const entering = expanded && !wasExpanded.current;
    wasExpanded.current = expanded;
    if (expanded && sheet.current) {
      sheet.current.scrollTop = 0;
      if (entering && !motion.matches && typeof sheet.current.animate === "function") {
        const card = sheet.current;
        const from = originTop() - card.getBoundingClientRect().top;
        const animation = card.animate(
          [
            { transform: `translate3d(0, ${from}px, 0)`, borderRadius: "28px" },
            { transform: "translate3d(0, 0, 0)", borderRadius: "0px" },
          ],
          { duration: 480, easing: "cubic-bezier(0.16, 1, 0.3, 1)", fill: "both" },
        );
        sheetAnimation.current = animation;
        void animation.finished.then(
          () => {
            if (sheetAnimation.current !== animation || returnStarted.current) return;
            animation.cancel();
            sheetAnimation.current = null;
          },
          () => {},
        );
      }
    }
    const finishImmediately = () => {
      if (!motion.matches || !sheetAnimation.current) return;
      sheetAnimation.current.cancel();
      sheetAnimation.current = null;
      if (returnStarted.current) navigate("/", { state: { visitorReturn: true } });
    };
    motion.addEventListener("change", finishImmediately);
    return () => {
      motion.removeEventListener("change", finishImmediately);
      sheetAnimation.current?.cancel();
      sheetAnimation.current = null;
    };
  }, [location.key, mobile, isLanding, selectedView, navigate]);
  function returnToLanding(event: MouseEvent<HTMLAnchorElement>) {
    if (
      !mobile ||
      !isLanding ||
      !selectedView ||
      event.button !== 0 ||
      event.metaKey ||
      event.ctrlKey ||
      event.shiftKey ||
      event.altKey
    )
      return;
    event.preventDefault();
    if (returnStarted.current) return;
    returnStarted.current = true;
    const card = sheet.current;
    if (
      !card ||
      typeof card.animate !== "function" ||
      window.matchMedia("(prefers-reduced-motion: reduce)").matches
    ) {
      navigate("/", { state: { visitorReturn: true } });
      return;
    }
    const currentStyle = getComputedStyle(card);
    const currentTransform = currentStyle.transform;
    const currentRadius = currentStyle.borderRadius;
    const top = parseFloat(currentStyle.top);
    sheetAnimation.current?.cancel();
    card.scrollTop = 0;
    setReturningKey(location.key);
    const animation = card.animate(
      [
        { transform: currentTransform, borderRadius: currentRadius },
        { transform: `translate3d(0, ${originTop() - top}px, 0)`, borderRadius: "28px" },
      ],
      { duration: 420, easing: "cubic-bezier(0.4, 0, 0.2, 1)", fill: "forwards" },
    );
    sheetAnimation.current = animation;
    void animation.finished.then(
      () => {
        if (sheetAnimation.current === animation) navigate("/", { state: { visitorReturn: true } });
      },
      () => {
        /* Route changes and resizing cancel the pending return. */
      },
    );
  }
  useLayoutEffect(() => {
    if (!isLanding) return;
    if (
      !selectedView &&
      !location.hash &&
      (location.state as { visitorReturn?: boolean } | null)?.visitorReturn
    ) {
      const frame = requestAnimationFrame(() => {
        document.getElementById("content")?.focus({ preventScroll: true });
        window.scrollTo({
          top: 0,
          behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches
            ? "instant"
            : "smooth",
        });
      });
      return () => cancelAnimationFrame(frame);
    }
    const targetId =
      location.hash === "#visit-options"
        ? "visit-options"
        : selectedView === "registration"
          ? "registration"
          : selectedView === "status"
            ? "booking-status"
            : !mobile && location.hash === "#registration"
              ? "registration"
              : null;
    if (!targetId) return;
    if (mobile && selectedView && targetId !== "visit-options") {
      const target = document.getElementById(targetId);
      if (target && !target.hidden) {
        target.scrollTop = 0;
        target.focus({ preventScroll: true });
      }
      return;
    }
    const frame = requestAnimationFrame(() => {
      const target = document.getElementById(targetId);
      if (!target || target.hidden) return;
      target.focus({ preventScroll: true });
      target.scrollIntoView({
        block: "start",
        behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches
          ? "instant"
          : "smooth",
      });
    });
    return () => cancelAnimationFrame(frame);
  }, [isLanding, mobile, selectedView, location.key, location.hash]);
  return (
    <div
      className={
        isLanding ? `visitor-home${choiceClosed ? " visitor-home--choice-closed" : ""}` : undefined
      }
      style={
        isLanding && mobile && selectedView
          ? {
              minHeight: `calc(var(--landing-photo-height) + ${sheetOrigin.current?.scroll ?? 0}px)`,
            }
          : undefined
      }
    >
      {isLanding && (
        <LandingHero
          links={links}
          inactive={mobile && selectedView !== null}
          onChoose={onChoose}
          showIntro={!mobile}
        />
      )}
      <div
        id="visitor-sheet"
        ref={sheet}
        className={
          isLanding && mobile
            ? `visitor-sheet${selectedView ? " visitor-sheet--selected" : ""}${returning ? " visitor-sheet--returning" : ""}`
            : undefined
        }
      >
        {isLanding && mobile && (
          <LandingIntro links={links} onChoose={onChoose} inactive={selectedView !== null} />
        )}
        <div
          id="registration"
          tabIndex={-1}
          className={`landing-task${isLanding && selectedView === "registration" ? " landing-task--selected" : ""}`}
          hidden={!registrationVisible || returning}
        >
          {isLanding && selectedView === "registration" && (
            <TaskNavigation mobile={mobile} returning={returning} onBack={returnToLanding} />
          )}
          <Suspense fallback={<Loading />}>
            {(registrationVisible || registrationMounted) && (
              <Booking
                embedded={isLanding}
                headingLevel={isLanding && !mobile ? 2 : 1}
                active={registrationVisible && !returning}
              />
            )}
          </Suspense>
          {isLanding && mobile && selectedView === "registration" && (
            <VisitInformation registrationHref={links.registration} />
          )}
        </div>
        {statusVisible && (
          <div
            id="booking-status"
            className="landing-task landing-task--selected landing-task--status"
            tabIndex={-1}
            hidden={returning}
          >
            <TaskNavigation mobile={mobile} returning={returning} onBack={returnToLanding} />
            <Suspense fallback={<Loading />}>
              <Status embedded headingLevel={mobile ? 1 : 2} />
            </Suspense>
          </div>
        )}
      </div>
      {isLanding && !mobile && <VisitInformation registrationHref={links.registration} />}
    </div>
  );
}
function TaskNavigation({
  mobile,
  returning,
  onBack,
}: {
  mobile: boolean;
  returning: boolean;
  onBack: ChooseAction;
}) {
  const { t } = useLanguage();
  return (
    <div className="landing-task-toolbar">
      <Link
        className="text-link landing-task-back"
        to={mobile ? "/" : "/#visit-options"}
        onClick={onBack}
        aria-disabled={returning || undefined}
      >
        <ArrowLeft size={16} />
        {t("Kembali", "Back")}
      </Link>
      <Brand linked={false} />
      <LanguageSwitch className="language-switch--task" />
    </div>
  );
}
function LandingHero({
  links,
  inactive,
  onChoose,
  showIntro,
}: {
  links: LandingLinks;
  inactive: boolean;
  onChoose: ChooseAction;
  showIntro: boolean;
}) {
  const { t } = useLanguage();
  const scene = useRef<HTMLElement>(null);
  useEffect(() => {
    const media = window.matchMedia(
      "(max-width: 760px) and (min-height: 501px) and (prefers-reduced-motion: no-preference)",
    );
    let disposed = false;
    let requested = false;
    let cleanup: (() => void) | undefined;
    const loadMotion = () => {
      if (inactive || !media.matches || requested) return;
      requested = true;
      void import("./features/landingMotion")
        .then(({ installLandingMotion }) => {
          window.removeEventListener("scroll", loadMotion);
          if (!disposed && scene.current)
            cleanup = installLandingMotion(scene.current.parentElement ?? scene.current);
        })
        .catch((error: unknown) => {
          // The native sticky image and form remain usable if motion cannot load.
          if (import.meta.env.DEV) console.warn("Visitor landing motion could not load.", error);
          requested = false;
        });
    };
    window.addEventListener("scroll", loadMotion, { passive: true });
    media.addEventListener("change", loadMotion);
    return () => {
      disposed = true;
      window.removeEventListener("scroll", loadMotion);
      media.removeEventListener("change", loadMotion);
      cleanup?.();
    };
  }, [inactive]);
  return (
    <section ref={scene} className="public-hero" aria-labelledby="welcome-heading" inert={inactive}>
      <div className="hero-visual">
        <picture>
          <source type="image/avif" srcSet={heroAvifSrcSet} sizes={heroImageSizes} />
          <source type="image/webp" srcSet={heroWebpSrcSet} sizes={heroImageSizes} />
          <img
            className="hero-image"
            src={hero}
            width="1920"
            height="1080"
            fetchPriority="high"
            alt={t(
              "Gedung di kawasan Ibu Kota Nusantara dikelilingi lanskap hijau",
              "Building in Nusantara Capital City surrounded by green landscape",
            )}
          />
        </picture>
        <div className="hero-toolbar">
          <Brand />
          <div className="hero-toolbar-actions">
            <LanguageSwitch className="language-switch--hero" />
          </div>
        </div>
        <div className="hero-photo-caption">
          <MapPin size={16} aria-hidden="true" />
          <span>Nusantara, Kalimantan Timur</span>
        </div>
      </div>
      {showIntro && <LandingIntro links={links} onChoose={onChoose} />}
      <div className="hero-note glass">
        <span className="note-icon">
          <MapPin size={21} />
        </span>
        <div>
          <strong>{t("Menuju Nusantara", "Your visit to Nusantara")}</strong>
          <small>
            {t("Siapkan pertemuan Anda, mulai dari sini.", "Plan your meeting, starting here.")}
          </small>
        </div>
      </div>
    </section>
  );
}
function LandingIntro({
  links,
  onChoose,
  inactive = false,
}: {
  links: LandingLinks;
  onChoose: ChooseAction;
  inactive?: boolean;
}) {
  const { t } = useLanguage();
  return (
    <div className="hero-content" inert={inactive}>
      <a className="hero-card-prompt" href="#visit-options">
        <span className="hero-sheet-handle" aria-hidden="true" />
        <span>
          <ArrowUp size={16} aria-hidden="true" />
          {t("Geser ke atas untuk mulai", "Swipe up to begin")}
        </span>
      </a>
      <h1 id="welcome-heading">
        {t("Setiap pertemuan,", "Every meeting,")} <br />
        {t("awal sebuah", "the start of")} <br />
        <em>{t("kemajuan.", "progress.")}</em>
      </h1>
      <p>
        {t(
          "Rencanakan kunjungan Anda ke Otorita Ibu Kota Nusantara. Lebih mudah, tertata, dan nyaman.",
          "Plan your visit to the Nusantara Capital Authority. Simple, organized, and comfortable.",
        )}
      </p>
      <div
        className="hero-actions"
        id="visit-options"
        tabIndex={-1}
        aria-label={t("Pilihan layanan kunjungan", "Visit services")}
      >
        <Link to={links.registration} className="link-button primary" onClick={onChoose}>
          {t("Daftar kunjungan", "Request a visit")} <ArrowDown size={18} />
        </Link>
        <Link to={links.status} className="hero-status-link" onClick={onChoose}>
          {t("Cek status kunjungan", "Check visit status")} <ArrowRight size={16} />
        </Link>
      </div>
    </div>
  );
}
function VisitInformation({ registrationHref }: { registrationHref: string }) {
  const { t } = useLanguage();
  return (
    <>
      <section className="public-steps">
        <div className="public-steps-intro">
          <h2>
            {t("Kunjungan yang baik.", "A better visit.")} <br />
            {t("Dari langkah pertama.", "From the first step.")}
          </h2>
          <p>
            {t(
              "Tiga langkah untuk pertemuan yang lebih terencana.",
              "Three steps to a well-planned meeting.",
            )}
          </p>
        </div>
        <ol>
          <li>
            <span>01</span>
            <div>
              <h3>{t("Rencanakan", "Plan")}</h3>
              <p>
                {t(
                  "Pilih tujuan, waktu, dan lengkapi data pengunjung.",
                  "Choose who to meet and when, then provide visitor details.",
                )}
              </p>
            </div>
          </li>
          <li>
            <span>02</span>
            <div>
              <h3>{t("Terima konfirmasi", "Get confirmation")}</h3>
              <p>
                {t(
                  "Terima email konfirmasi dan pantau status dengan email serta kode kunjungan.",
                  "Get a confirmation email and track your visit with your email and visit code.",
                )}
              </p>
            </div>
          </li>
          <li>
            <span>03</span>
            <div>
              <h3>{t("Datang dengan tenang", "Arrive prepared")}</h3>
              <p>
                {t(
                  "Bawa identitas asli dan kode kunjungan atau QR kunjungan.",
                  "Bring your original ID and visit code or QR code.",
                )}
              </p>
            </div>
          </li>
        </ol>
      </section>
      <section className="visit-prep">
        <div>
          <CalendarDays size={24} />
          <h2>
            {t("Sedikit persiapan.", "A little preparation.")}
            <br />
            {t("Pengalaman yang lebih baik.", "A better experience.")}
          </h2>
        </div>
        <div>
          <p>
            <Check size={17} />
            {t("Siapkan nama setiap anggota rombongan.", "Have each visitor's name ready.")}
          </p>
          <p>
            <ShieldCheck size={17} />
            {t(
              "Bawa KTP, paspor, atau SIM asli saat kedatangan.",
              "Bring your original ID card, passport, or driver's license.",
            )}
          </p>
          <p>
            <Clock3 size={17} />
            {t(
              "Seluruh jadwal menggunakan Waktu Indonesia Tengah.",
              "All schedules use Central Indonesia Time (WITA).",
            )}
          </p>
        </div>
        <Link to={registrationHref} className="text-link">
          {t("Mulai sekarang", "Get started")} <ArrowRight size={17} />
        </Link>
      </section>
    </>
  );
}
createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <LanguageProvider>
      <AppBoundary>
        <DataProvider>
          <BrowserRouter>
            <RouteFocus />
            <App />
          </BrowserRouter>
        </DataProvider>
      </AppBoundary>
    </LanguageProvider>
  </StrictMode>,
);
