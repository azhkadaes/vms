import { Component, forwardRef, useEffect, type ReactNode, type ButtonHTMLAttributes } from "react";
import { useLocation } from "react-router";
import { ArrowRight } from "lucide-react";
import { preview } from "@vms/data/config";
import { useUiLanguage } from "./language";
import logo from "./assets/ikn-logo.png";
import logo48 from "./assets/ikn-logo-48.webp?no-inline";
import logo96 from "./assets/ikn-logo-96.webp";
import logo144 from "./assets/ikn-logo-144.webp";

export function Brand({
  dark = false,
  linked = true,
  href = "/",
}: {
  dark?: boolean;
  linked?: boolean;
  href?: string;
}) {
  const { t } = useUiLanguage();
  const content = (
    <>
      <picture>
        <source
          type="image/webp"
          srcSet={`${logo48} 48w, ${logo96} 96w, ${logo144} 144w`}
          sizes="48px"
        />
        <img src={logo} width="42" height="42" alt="" />
      </picture>
      <span>
        <strong>Visitor Management System</strong>
        <small>(VMS) · Otorita IKN</small>
      </span>
    </>
  );
  const className = `brand ${dark ? "on-dark" : ""}`;
  return linked ? (
    <a
      href={href}
      className={className}
      aria-label={t(
        "Visitor Management System (VMS) · Otorita IKN · Beranda",
        "Visitor Management System (VMS) · Otorita IKN · Home",
      )}
    >
      {content}
    </a>
  ) : (
    <div className={className}>{content}</div>
  );
}
type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  kind?: "primary" | "secondary" | "ghost" | "danger" | "danger-outline";
};
export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { children, kind = "primary", className = "", ...props },
  ref,
) {
  return (
    <button ref={ref} className={`button ${kind} ${className}`} {...props}>
      {children}
    </button>
  );
});
export function PreviewNotice() {
  const { t } = useUiLanguage();
  return preview ? (
    <div
      className="preview-notice"
      role="region"
      aria-label={t("Pratinjau lokal", "Local preview")}
    >
      <span className="live-dot" />
      {t(
        "Preview lokal · Data contoh, tanpa email atau unggahan nyata. Gunakan data fiktif.",
        "Local preview · Sample data, without real email or uploads. Use fictional data.",
      )}
    </div>
  ) : null;
}

export function Loading() {
  const { t } = useUiLanguage();
  return (
    <div className="skeleton" role="status" aria-label={t("Memuat data", "Loading data")}>
      <div />
      <div />
      <div />
    </div>
  );
}

export function RouteFocus() {
  const { pathname } = useLocation();
  useEffect(() => {
    document.querySelector<HTMLElement>("main")?.focus();
    window.scrollTo(0, 0);
  }, [pathname]);
  return null;
}
export class AppBoundary extends Component<{ children: ReactNode }, { failed: boolean }> {
  state = { failed: false };
  static getDerivedStateFromError() {
    return { failed: true };
  }
  render() {
    const english = document.documentElement.lang === "en";
    return this.state.failed ? (
      <main className="empty">
        <h1>{english ? "Page could not be loaded" : "Halaman belum dapat dimuat"}</h1>
        <p>{english ? "Reload to try again." : "Muat ulang untuk mencoba kembali."}</p>
        <Button onClick={() => location.reload()}>
          {english ? "Reload" : "Muat ulang"} <ArrowRight size={16} />
        </Button>
      </main>
    ) : (
      this.props.children
    );
  }
}
