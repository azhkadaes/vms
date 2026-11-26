import * as Dialog from "@radix-ui/react-dialog";
import { StrictMode, Suspense, lazy } from "react";
import { createRoot } from "react-dom/client";
import { Toaster } from "@vms/ui/toast";
import {
  BrowserRouter,
  NavLink,
  Navigate,
  Outlet,
  Route,
  Routes,
  useLocation,
  useNavigate,
} from "react-router";
import ProfilePreview from "./features/ProfilePreview";
import {
  LayoutDashboard,
  CalendarDays,
  BookOpen,
  LogIn,
  LogOut,
  Users,
  Shield,
  Menu,
  X,
  ChevronRight,
  ExternalLink,
} from "lucide-react";
import {
  DataProvider,
  SessionProvider,
  preview,
  roles,
  useSession,
  dateLabel,
  today,
  ApiError,
  signIn,
} from "@vms/data";
import {
  AppBoundary,
  Brand,
  Button,
  ErrorNotice,
  RequiredMark,
  RequiredFieldsHint,
  Loading,
  PreviewNotice,
  RouteFocus,
} from "@vms/ui";
import { useEffect, useState } from "react";
import "@vms/ui/styles.css";
import "./styles.css";
import BookingRefresh from "./components/BookingRefresh";
import { RoleSelection, RoleSwitcher } from "./features/RoleControls";
const visitorHomeHref =
  preview && location.port === "3002" ? `${location.protocol}//${location.hostname}:3000/` : "/";
import {
  SidebarProvider,
  SidebarRoot,
  SidebarRail,
  SidebarHeader,
  SidebarContent,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
  SidebarFooter,
} from "./components/lightswind/sidebar";
const Dashboard = lazy(() => import("./pages/Dashboard"));
const Bookings = lazy(() => import("./pages/Bookings"));
const VisitLog = lazy(() => import("./pages/VisitLog"));
const Reception = lazy(() => import("./pages/Reception"));
const Settings = lazy(() => import("./pages/Settings"));
const PasswordChange = lazy(() => import("./features/PasswordChange"));
const items = [
  { title: "Dashboard", to: "/dashboard", icon: LayoutDashboard },
  {
    title: "Kunjungan hari ini",
    to: "/today",
    icon: CalendarDays,
    permission: "bookings",
  },
  { title: "Kunjungan", to: "/booking", icon: BookOpen, permission: "bookings" },
  { title: "Check in", to: "/check-in", icon: LogIn, permission: "checkin" },
  {
    title: "Check out",
    to: "/check-out",
    icon: LogOut,
    permission: "checkout",
  },
  { title: "Pegawai", to: "/employees", icon: Users, permission: "people" },
  { title: "Peran & akses", to: "/roles", icon: Shield, permission: "roles" },
];
function Shell() {
  const { session, pendingSession, loading, error, signOut, refreshSession } = useSession();
  const [menu, setMenu] = useState(false);
  const [sidebarExpanded, setSidebarExpanded] = useState(() => {
    try {
      return localStorage.getItem("vms-backoffice-sidebar-expanded") !== "false";
    } catch {
      return true;
    }
  });
  const [signoutError, setSignoutError] = useState<unknown>();
  const loc = useLocation();
  const pageTitle = items.find((i) => i.to === loc.pathname)?.title || "Backoffice";
  useEffect(() => {
    document.title = `${pageTitle} · Visitor Management System (VMS)`;
  }, [pageTitle]);
  if (loading) return <Loading />;
  if (pendingSession) return <RoleSelection />;
  if (!session) {
    if (error && !(error instanceof ApiError && error.status === 401))
      return (
        <main className="login-page">
          <section className="panel login-panel">
            <h1>
              {error instanceof ApiError && error.status === 403
                ? "Akses tidak diizinkan."
                : "Layanan sesi belum tersedia."}
            </h1>
            <ErrorNotice error={error} />
            <Button onClick={() => void refreshSession()}>Coba lagi</Button>
          </section>
        </main>
      );
    return <Navigate to="/login" replace />;
  }
  if (session.mustChangePassword)
    return (
      <Suspense fallback={<Loading />}>
        <PasswordChange />
      </Suspense>
    );
  const visible = items.filter(
    (i) =>
      (!i.permission || session.permissions.includes(i.permission)) &&
      !(session.role === "Super Admin" && i.to === "/today"),
  );
  const title = pageTitle;
  const changeSidebarExpanded = (expanded: boolean) => {
    setSidebarExpanded(expanded);
    try {
      localStorage.setItem("vms-backoffice-sidebar-expanded", String(expanded));
    } catch {
      /* Sidebar remains usable when preferences cannot be saved. */
    }
  };
  return (
    <SidebarProvider expanded={sidebarExpanded} onExpandedChange={changeSidebarExpanded}>
      <a href="#content" className="skip-link">
        Lewati ke konten
      </a>
      <div className="admin-shell" data-sidebar-expanded={sidebarExpanded}>
        <Sidebar open={menu} onClose={() => setMenu(false)}>
          <SidebarHeader>
            <Brand href={visitorHomeHref} />
          </SidebarHeader>
          <SidebarContent>
            <SidebarGroup>
              <SidebarGroupLabel>Ruang kerja</SidebarGroupLabel>
              <SidebarGroupContent>
                <SidebarMenu aria-label="Navigasi backoffice">
                  {visible.map((item, i) => {
                    const active =
                      loc.pathname === item.to || loc.pathname.startsWith(`${item.to}/`);
                    return (
                      <SidebarMenuItem key={item.to}>
                        <SidebarMenuButton
                          isActive={active}
                          className={
                            i === visible.findIndex((x) => x.to === "/employees")
                              ? "nav-separator"
                              : ""
                          }
                        >
                          <NavLink to={item.to} onClick={() => setMenu(false)} title={item.title}>
                            <item.icon size={18} aria-hidden="true" />
                            <span className="lw-sidebar-link-label">{item.title}</span>
                          </NavLink>
                        </SidebarMenuButton>
                      </SidebarMenuItem>
                    );
                  })}
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>
          </SidebarContent>
          <SidebarFooter>
            <div className="sidebar-signature">
              <Shield size={19} />
              <span>
                Terhubung untuk melayani.
                <small>Otorita Ibu Kota Nusantara</small>
              </span>
            </div>
            <RoleSwitcher />
            <a href={visitorHomeHref} target="_blank" rel="noopener noreferrer">
              Portal pengunjung <ExternalLink size={14} />
            </a>
            <Button
              kind="ghost"
              className="sidebar-signout"
              onClick={() => void signOut().catch(setSignoutError)}
            >
              Keluar <LogOut size={16} />
            </Button>
          </SidebarFooter>
        </Sidebar>
        <div className="admin-body">
          <header className="admin-topbar glass">
            <div className="toolbar">
              <Button
                kind="ghost"
                size="icon"
                className="menu-button"
                aria-label="Buka navigasi"
                aria-expanded={menu}
                onClick={() => setMenu(!menu)}
              >
                <Menu size={20} />
              </Button>
              <span className="mobile-hide">Backoffice</span>
              <ChevronRight size={14} className="mobile-hide" />
              <strong>{title}</strong>
            </div>
            <div className="toolbar">
              <span className="top-date mobile-hide">{dateLabel(today())} · WITA</span>
              <ProfilePreview key={JSON.stringify([session.id, session.role])} session={session} />
            </div>
          </header>
          <PreviewNotice />
          <main id="content" tabIndex={-1} className="admin-main">
            <ErrorNotice error={signoutError || error} />
            {error && (
              <Button kind="secondary" onClick={() => void refreshSession()}>
                Periksa sesi kembali
              </Button>
            )}
            {session.permissions.includes("bookings") &&
              !(session.role === "Super Admin" && loc.pathname === "/dashboard") &&
              ["/dashboard", "/today", "/booking", "/check-in", "/check-out"].includes(
                loc.pathname,
              ) && (
                <BookingRefresh mode={session.role === "Receptionist" ? "reception" : "staff"} />
              )}
            <Suspense fallback={<Loading />}>
              <Outlet key={JSON.stringify([session.id, session.role])} />
            </Suspense>
          </main>
          <footer className="admin-footer">
            <span>© {new Date().getFullYear()} Otorita Ibu Kota Nusantara</span>
            <span>Visitor Management System (VMS)</span>
          </footer>
        </div>
      </div>
    </SidebarProvider>
  );
}
function Guard({ permission, children }: { permission: string; children: React.ReactNode }) {
  const { session } = useSession();
  return session?.permissions.includes(permission) ? (
    children
  ) : (
    <div className="empty">
      <Shield size={32} />
      <h1>Akses terbatas</h1>
      <p>Peran Anda tidak memiliki akses ke halaman ini.</p>
      <NavLink to="/dashboard" className="link-button primary">
        Kembali ke dashboard
      </NavLink>
    </div>
  );
}
function VisitRoute({
  children,
  todayOnly = false,
}: {
  children: React.ReactNode;
  todayOnly?: boolean;
}) {
  const { session } = useSession();
  const loc = useLocation();
  if (session?.role !== "Super Admin") return children;
  if (!todayOnly)
    return (
      <Guard permission="reports">
        <VisitLog />
      </Guard>
    );
  const params = new URLSearchParams(loc.search);
  if (params.get("sort") === "time" || params.get("sort") === "priority") params.delete("sort");
  if (params.get("vip") === "1") params.set("kind", "vip");
  if (todayOnly) {
    params.set("from", today());
    params.set("to", today());
  }
  for (const key of ["new", "selected", "vip", "page"]) params.delete(key);
  return <Navigate to={`/booking${params.size ? `?${params}` : ""}`} replace />;
}
function Login() {
  const { session, pendingSession, loading } = useSession();
  const [error, setError] = useState<unknown>();
  const [pending, setPending] = useState(false);
  if (!preview && loading) return <Loading />;
  if (!preview && (session || pendingSession)) return <Navigate to="/dashboard" replace />;
  return (
    <main className="login-page">
      <Brand href={visitorHomeHref} />
      <div className="panel login-panel">
        <h1>Selamat datang kembali.</h1>
        <p>Masuk ke ruang kerja Visitor Management System (VMS).</p>
        <form
          onSubmit={async (e) => {
            e.preventDefault();
            if (pending) return;
            setPending(true);
            setError(null);
            const data = new FormData(e.currentTarget);
            try {
              await signIn({
                email: String(data.get("email") || ""),
                password: String(data.get("password") || ""),
              });
              location.assign("/admin/dashboard");
            } catch (err) {
              setError(err);
            } finally {
              setPending(false);
            }
          }}
        >
          <ErrorNotice error={error} />
          {!preview && (
            <>
              <RequiredFieldsHint />
              <div className="field">
                <label htmlFor="email">
                  Email pegawai <RequiredMark />
                </label>
                <input required id="email" name="email" type="email" autoComplete="username" />
              </div>
              <div className="field">
                <label htmlFor="password">
                  Kata sandi <RequiredMark />
                </label>
                <input
                  required
                  id="password"
                  name="password"
                  type="password"
                  autoComplete="current-password"
                />
              </div>
            </>
          )}
          <Button disabled={pending} type="submit">
            {pending ? "Memproses…" : preview ? "Buka preview lokal" : "Masuk ke backoffice"}
            <ChevronRight size={17} />
          </Button>
        </form>
        <PreviewNotice />
      </div>
    </main>
  );
}
createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <AppBoundary>
      <DataProvider>
        <BrowserRouter basename="/admin">
          <SessionProvider>
            <Toaster />
            <RouteFocus />
            <Routes>
              <Route path="login" element={<Login />} />
              <Route element={<Shell />}>
                <Route index element={<Navigate to="/dashboard" replace />} />
                <Route path="dashboard" element={<Dashboard />} />
                <Route
                  path="booking"
                  element={
                    <Guard permission="bookings">
                      <VisitRoute>
                        <Bookings />
                      </VisitRoute>
                    </Guard>
                  }
                />
                <Route
                  path="today"
                  element={
                    <Guard permission="bookings">
                      <VisitRoute todayOnly>
                        <Bookings todayOnly />
                      </VisitRoute>
                    </Guard>
                  }
                />
                <Route
                  path="check-in"
                  element={
                    <Guard permission="checkin">
                      <Reception mode="checkin" />
                    </Guard>
                  }
                />
                <Route
                  path="check-out"
                  element={
                    <Guard permission="checkout">
                      <Reception mode="checkout" />
                    </Guard>
                  }
                />
                <Route path="integration" element={<Navigate to="/dashboard" replace />} />
                <Route
                  path="analytics"
                  element={<Navigate to={`/dashboard${location.search}`} replace />}
                />
                <Route
                  path="logs"
                  element={<Navigate to={`/booking${location.search}`} replace />}
                />
                {["employees", "roles"].map((page) => (
                  <Route
                    key={page}
                    path={page}
                    element={
                      <Guard
                        permission={
                          {
                            employees: "people",
                            roles: "roles",
                          }[page]!
                        }
                      >
                        <Settings page={page} />
                      </Guard>
                    }
                  />
                ))}
                <Route
                  path="*"
                  element={
                    <div className="empty">
                      <h1>Halaman tidak ditemukan</h1>
                      <NavLink className="text-link" to="/dashboard">
                        Kembali ke dashboard
                      </NavLink>
                    </div>
                  }
                />
              </Route>
            </Routes>
          </SessionProvider>
        </BrowserRouter>
      </DataProvider>
    </AppBoundary>
  </StrictMode>,
);

function Sidebar({
  open,
  onClose,
  children,
}: {
  open: boolean;
  onClose: () => void;
  children: React.ReactNode;
}) {
  const [mobile, setMobile] = useState(() => matchMedia("(max-width: 760px)").matches);
  useEffect(() => {
    const media = matchMedia("(max-width: 760px)");
    const changed = () => setMobile(media.matches);
    media.addEventListener("change", changed);
    return () => media.removeEventListener("change", changed);
  }, []);
  if (!mobile)
    return (
      <SidebarRoot id="backoffice-sidebar">
        {children}
        <SidebarRail />
      </SidebarRoot>
    );
  return (
    <Dialog.Root open={open} onOpenChange={(value) => !value && onClose()}>
      <Dialog.Portal>
        <Dialog.Overlay className="nav-scrim" />
        <Dialog.Content
          id="backoffice-sidebar"
          className="sidebar lw-sidebar sidebar-open"
          onOpenAutoFocus={(event) => {
            event.preventDefault();
            requestAnimationFrame(() =>
              document.querySelector<HTMLAnchorElement>(".sidebar nav a.active")?.focus(),
            );
          }}
          onCloseAutoFocus={(event) => {
            event.preventDefault();
            document.querySelector<HTMLButtonElement>(".menu-button")?.focus();
          }}
        >
          <Dialog.Title className="sr-only">Navigasi backoffice</Dialog.Title>
          <Dialog.Description className="sr-only">
            Pilih halaman ruang kerja Anda.
          </Dialog.Description>
          <Dialog.Close asChild>
            <Button className="sidebar-close" kind="ghost" size="icon" aria-label="Tutup navigasi">
              <X size={18} />
            </Button>
          </Dialog.Close>
          {children}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
