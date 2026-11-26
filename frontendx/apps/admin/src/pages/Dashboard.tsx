import {
  ApiError,
  bookingName,
  canCreateAppointment,
  dateLabel,
  selectBookings,
  today,
  useBookings,
  useSession,
  visitTiming,
} from "@vms/data";
import { Empty, QueryState, StatusBadge, VisitKind, TableRegion } from "@vms/ui";
import {
  ArrowRight,
  ArrowUpRight,
  CalendarDays,
  Check,
  Clock3,
  LogIn,
  ShieldCheck,
  Users,
} from "lucide-react";
import { Link } from "react-router";
import { lazy, Suspense } from "react";
import { Loading } from "@vms/ui";
const DashboardAnalytics = lazy(() => import("../features/DashboardAnalytics"));
export default function Dashboard() {
  const { session } = useSession();
  const q = useBookings({ filter: { view: "dashboard", date: today(), page: 1, pageSize: 50 } });
  const available =
    q.data !== undefined && !(q.error instanceof ApiError && [401, 403].includes(q.error.status));
  const list = available ? q.data || [] : [];
  const current = list.filter((b) => b.date === today());
  const pending = list.filter((b) => b.status === "pending");
  const configOnly = session?.role === "Super Admin";
  const reception = session?.role === "Receptionist";
  const arrivals = selectBookings(
    current.filter((b) => visitTiming(b).displayStatus === "approved"),
    "",
    "all",
    "time",
  );
  const inside = selectBookings(
    current.filter((b) => b.status === "checked_in"),
    "",
    "all",
    "time",
  );
  const agenda = reception ? arrivals : selectBookings(current, "", "all", "time");
  const attention = reception ? inside : selectBookings(pending, "", "all", "priority");
  return (
    <div className="dashboard-page">
      <div className="page-intro">
        <div>
          <h1>Selamat datang, {session?.name.split(" ")[0]}.</h1>
          <p>
            {configOnly
              ? "Kelola akses dan tata kelola layanan kunjungan."
              : reception
                ? "Siapkan kedatangan dan kepulangan pengunjung hari ini."
                : "Semua yang perlu Anda siapkan untuk kunjungan hari ini."}
          </p>
        </div>
        <div className="date-chip">
          <CalendarDays size={17} />
          {dateLabel(today())}
        </div>
      </div>
      <div className="dashboard-welcome">
        <div>
          <h2>
            {configOnly
              ? "Layanan yang tertata, akses yang terjaga."
              : reception
                ? "Siap menyambut pengunjung?"
                : "Pertemuan yang baik dimulai dari persiapan."}
          </h2>
          <p>
            {configOnly
              ? "Pantau kunjungan, unduh laporan, dan kelola akses pegawai."
              : reception
                ? available
                  ? `${q.metrics?.arrivals ?? arrivals.length} kunjungan disetujui menunggu kedatangan · ${q.metrics?.inside ?? inside.length} pengunjung sedang berada di lokasi.`
                  : "Ringkasan kedatangan belum tersedia."
                : !available
                  ? "Ringkasan kunjungan belum tersedia."
                  : `${q.metrics?.pending ?? pending.length} permohonan menunggu keputusan. Mari siapkan pengalaman berkunjung yang lebih baik.`}
          </p>
          <div className="welcome-actions">
            {session && canCreateAppointment(session) && (
              <Link className="link-button primary" to="/booking?new=1">
                Buat janji temu <ArrowRight size={16} />
              </Link>
            )}
            <Link
              className={reception ? "link-button primary" : ""}
              to={configOnly ? "/booking" : reception ? "/check-in" : "/booking?status=pending"}
            >
              {configOnly ? "Buka kunjungan" : reception ? "Mulai check in" : "Tinjau permohonan"}{" "}
              <ArrowRight size={16} />
            </Link>
            {reception && (
              <Link className="link-button secondary" to="/check-out">
                Check out pengunjung <ArrowRight size={16} />
              </Link>
            )}
          </div>
        </div>
        <div className="welcome-mark">
          <ShieldCheck size={58} strokeWidth={1} />
        </div>
      </div>
      {configOnly ? (
        <div className="admin-quick-links">
          {[
            {
              name: "Pegawai",
              to: "/employees",
              icon: Users,
              text: "Akun dan penugasan pegawai",
            },
            {
              name: "Peran & akses",
              to: "/roles",
              icon: ShieldCheck,
              text: "Izin berdasarkan tanggung jawab",
            },
          ].map((x) => (
            <Link className="panel quick-setting" key={x.to} to={x.to}>
              <x.icon size={25} />
              <h2>{x.name}</h2>
              <p>{x.text}</p>
              <ArrowUpRight size={20} />
            </Link>
          ))}
        </div>
      ) : (
        <QueryState
          pending={q.isPending}
          error={q.error}
          retry={q.refetch}
          hasData={q.data !== undefined}
        >
          <div className="metrics">
            {[
              {
                label: reception ? "Menunggu kedatangan" : "Kunjungan hari ini",
                value: reception
                  ? (q.metrics?.arrivals ?? arrivals.length)
                  : (q.metrics?.today ?? current.length),
                icon: CalendarDays,
                note: reception ? "Kunjungan disetujui" : "Jumlah janji temu",
                to: reception ? "/today?status=approved" : "/today",
              },
              {
                label: reception ? "Sedang berkunjung" : "Menunggu keputusan",
                value: reception
                  ? (q.metrics?.inside ?? inside.length)
                  : (q.metrics?.pending ?? pending.length),
                icon: Clock3,
                note: reception ? "Kunjungan sudah check in" : "Perlu ditinjau",
                to: reception ? "/check-out" : "/booking?status=pending",
              },
              {
                label: reception ? "Kunjungan hari ini" : "Sedang berkunjung",
                value: reception
                  ? (q.metrics?.today ?? current.length)
                  : (q.metrics?.inside ?? inside.length),
                icon: LogIn,
                note: reception ? "Disetujui, Checked in, Checked out" : "Kunjungan sudah check in",
                to: reception ? "/today" : "/booking?status=checked_in",
              },
              {
                label: "Kunjungan selesai",
                value:
                  q.metrics?.completed ?? current.filter((b) => b.status === "checked_out").length,
                icon: Check,
                note: "Kunjungan sudah check out",
                to: "/booking?status=checked_out",
              },
            ].map((m) => (
              <Link to={m.to} key={m.label} className="metric">
                <div>
                  <span>{m.label}</span>
                  <m.icon size={18} aria-hidden="true" />
                </div>
                <strong>{m.value.toString().padStart(2, "0")}</strong>
                <small>
                  {m.note}
                  <ArrowUpRight size={16} aria-hidden="true" />
                </small>
              </Link>
            ))}
          </div>
          <div className="dashboard-columns">
            <section className="panel">
              <div className="panel-heading">
                <div>
                  <h2>{reception ? "Menunggu kedatangan" : "Agenda hari ini"}</h2>
                  <p>
                    {reception
                      ? "Kunjungan hari ini yang sudah disetujui."
                      : "Jadwal kunjungan dalam lingkup akses Anda."}
                  </p>
                </div>
                <Link
                  className="text-link"
                  to={reception ? "/today?status=approved" : "/today"}
                  aria-label="Lihat semua agenda"
                >
                  <ArrowUpRight size={20} />
                </Link>
              </div>
              {!agenda.length ? (
                <Empty
                  title={reception ? "Belum ada kedatangan berikutnya" : undefined}
                  text={
                    reception
                      ? "Kunjungan yang sudah disetujui untuk hari ini akan tampil di sini."
                      : undefined
                  }
                />
              ) : (
                <>
                  <TableRegion className="dashboard-agenda-table" label="Agenda kunjungan">
                    <table>
                      <thead>
                        <tr>
                          <th>Waktu · WITA</th>
                          <th>Pengunjung</th>
                          <th>Status</th>
                        </tr>
                      </thead>
                      <tbody>
                        {agenda.slice(0, 5).map((b) => (
                          <tr key={b.id}>
                            <td className="tabular">
                              {b.time}
                              <small>{b.duration} menit</small>
                            </td>
                            <td>
                              {reception ? (
                                <strong>{bookingName(b)} <VisitKind booking={b} /></strong>
                              ) : (
                                <Link to={`/booking?selected=${b.id}`} className="row-link">
                                  {bookingName(b)}
                                  <VisitKind booking={b} />
                                </Link>
                              )}
                              <small>{b.organization}</small>
                            </td>
                            <td>
                              <StatusBadge
                                status={b.status}
                                timing={b.timing}
                                rescheduled={b.rescheduled}
                              />
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </TableRegion>
                  <ul className="dashboard-agenda-list">
                    {agenda.slice(0, 5).map((b) => {
                      const content = (
                        <>
                          <span className="agenda-time tabular">
                            <strong>{b.time}</strong>
                            <small>{b.duration} menit</small>
                          </span>
                          <span className="agenda-details">
                            <strong>{bookingName(b)} <VisitKind booking={b} /></strong>
                            <small>{b.organization}</small>
                            <StatusBadge
                              status={b.status}
                              timing={b.timing}
                              rescheduled={b.rescheduled}
                            />
                          </span>
                        </>
                      );
                      return (
                        <li key={b.id}>
                          {reception ? (
                            <div className="agenda-entry">{content}</div>
                          ) : (
                            <Link to={`/booking?selected=${b.id}`} className="agenda-entry">
                              {content}
                              <ArrowUpRight size={16} aria-hidden="true" />
                            </Link>
                          )}
                        </li>
                      );
                    })}
                  </ul>
                </>
              )}
            </section>
            <section className="panel">
              <div className="panel-heading">
                <div>
                  <h2>{reception ? "Sedang berkunjung" : "Perlu perhatian"}</h2>
                  <p>
                    {reception
                      ? "Siap diproses saat pengunjung pulang."
                      : "Permohonan yang belum diputuskan."}
                  </p>
                </div>
                <span className="count">
                  {reception
                    ? (q.metrics?.inside ?? attention.length)
                    : (q.metrics?.pending ?? attention.length)}
                </span>
              </div>
              {attention.length ? (
                attention.slice(0, 4).map((b) => {
                  const content = (
                    <>
                      <span className="avatar">{bookingName(b)[0]}</span>
                      <span>
                        <strong>
                          {bookingName(b)}
                          <VisitKind booking={b} />
                        </strong>
                        <small>
                          {dateLabel(b.date)} · {b.time} WITA · {b.organization}
                        </small>
                      </span>
                    </>
                  );
                  return reception ? (
                    <div key={b.id} className="attention-row">{content}</div>
                  ) : (
                    <Link key={b.id} to={`/booking?selected=${b.id}`} className="attention-row">
                      {content}
                      <ArrowUpRight size={16} aria-hidden="true" />
                    </Link>
                  );
                })
              ) : (
                <Empty
                  title={reception ? "Belum ada pengunjung di dalam" : "Semua sudah ditinjau"}
                  text={
                    reception
                      ? "Daftar diperbarui setelah check in dicatat."
                      : "Permohonan baru akan tampil di sini."
                  }
                />
              )}
              <Link
                to={reception ? "/check-out" : "/booking?status=pending"}
                className="attention-footer"
              >
                {reception ? "Mulai check out" : "Lihat permohonan"} <ArrowRight size={16} />
              </Link>
            </section>
          </div>
        </QueryState>
      )}
      {session?.permissions.includes("reports") && (
        <div className="section-gap">
          <Suspense fallback={<Loading />}>
            <DashboardAnalytics key={session.role} />
          </Suspense>
        </div>
      )}
    </div>
  );
}
