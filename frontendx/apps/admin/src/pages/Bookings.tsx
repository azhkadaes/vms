import {
  ApiError,
  bookingName,
  bookingStatusesFor,
  canCreateAppointment,
  dateLabel,
  isRestricted,
  labels,
  partyLabel,
  preview,
  selectBookings,
  today,
  useBookingDetail,
  useCatalogs,
  useBookings,
  useSession,
  type SortKey,
  type StaffBooking,
} from "@vms/data";
import {
  Select,
  Button,
  Empty,
  ErrorNotice,
  Loading,
  Modal,
  QueryState,
  SearchField,
  StatusBadge,
  VisitKind,
  TableRegion,
} from "@vms/ui";
import { CheckboxDropdown } from "@vms/ui/checkbox-dropdown";
import { visitKindLabels } from "@vms/data/visit-log";
import { toast } from "@vms/ui/toast";
import {
  ArrowDownUp,
  ArrowUpRight,
  Check,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  Download,
  Plus,
} from "lucide-react";
import { lazy, Suspense, useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router";
import { useDebouncedValue } from "../hooks/useDebouncedValue";
const CreateAppointment = lazy(() => import("../features/CreateAppointment"));
const RestrictedAppointment = lazy(() => import("../features/RestrictedAppointment"));
const Reports = lazy(() => import("./Reports"));
const BookingDetail = lazy(() => import("../features/BookingDetail"));
const visitKindOptions = ["standard", "vip", "restricted"] as const;
type ListKind = (typeof visitKindOptions)[number];
function kindOf(b: { visitKind?: string; vip?: boolean }): ListKind {
  return isRestricted(b as Parameters<typeof isRestricted>[0])
    ? "restricted"
    : b.vip
      ? "vip"
      : "standard";
}
function officerPAs(pas: StaffBooking["officerPAs"]) {
  if (!pas?.length) return "Belum ada PA";
  return pas.map((pa) => (
    <div key={pa.name + pa.phones.join()}>
      {pa.name}
      <small>{pa.phones.join(", ") || "Nomor HP belum diisi"}</small>
    </div>
  ));
}
export default function Bookings({ todayOnly = false }: { todayOnly?: boolean }) {
  const { session } = useSession();
  const [params, setParams] = useSearchParams();
  const [notice, setNotice] = useState("");
  const [exportOpen, setExportOpen] = useState(false);
  const detailTrigger = useRef<HTMLElement | null>(null);
  const createTrigger = useRef<HTMLElement | null>(null);
  const mayCreate =
    !!session &&
    canCreateAppointment(
      session,
      undefined,
      params.get("kind") === "restricted" && params.get("new") === "1" ? "restricted" : "standard",
    );
  const reception = session?.role === "Receptionist";
  const showKindFilter = session?.role === "Protocol Officer";
  const statusOptions = session ? bookingStatusesFor(session) : [];
  const creating = params.get("new") === "1";
  const kind = params.get("kind");
  function openCreate(kind: "standard" | "restricted") {
    setParams((p) => {
      p.delete("selected");
      p.set("new", "1");
      p.set("kind", kind);
      return p;
    });
  }
  const search = params.get("q") || "";
  const settledSearch = useDebouncedValue(search, preview ? 0 : 250);
  // `status` and `kind` hold comma-separated choices; none chosen means every row.
  const requestedStatus = params.get("status") || "";
  const statuses = statusOptions.filter((s) => requestedStatus.split(",").includes(s));
  const status = statuses.join(",");
  const requestedKind = params.get("list-kind") || "";
  const kinds = showKindFilter
    ? visitKindOptions.filter((k) => requestedKind.split(",").includes(k))
    : [];
  const officerOptions = (useCatalogs().data?.officers || []).map(({ id, name }) => ({
    value: id,
    label: name,
  }));
  const officers = reception ? (params.get("officer") || "").split(",").filter(Boolean) : [];
  const officerNames = officerOptions.filter((o) => officers.includes(o.value)).map((o) => o.label);
  const requestedSort = params.get("sort");
  const sort: SortKey = ["name", "organization", "host", "time", "status"].includes(
    requestedSort || "",
  )
    ? (requestedSort as SortKey)
    : "priority";
  useEffect(() => {
    if (!session || (requestedStatus === status && requestedKind === kinds.join(","))) return;
    setParams(
      (p) => {
        if (status) p.set("status", status);
        else p.delete("status");
        if (kinds.length) p.set("list-kind", kinds.join(","));
        else p.delete("list-kind");
        p.delete("page");
        return p;
      },
      { replace: true },
    );
  }, [session, requestedStatus, status, requestedKind, kinds, setParams]);
  const desc = sort !== "priority" && params.get("dir") === "desc";
  const page = Math.min(100000, Math.max(1, Math.floor(Number(params.get("page")) || 1)));
  const q = useBookings({
    filter: {
      view: "list",
      q: settledSearch,
      status: status || "all",
      ...(kinds.length ? { kind: kinds.join(",") } : {}),
      ...(officers.length ? { officer: officers.join(",") } : {}),
      sort: ["priority", "name", "organization", "host", "time", "status"].includes(sort)
        ? sort
        : "priority",
      dir: desc ? "desc" : "asc",
      ...(todayOnly ? { date: today() } : {}),
      page,
      pageSize: 7,
    },
  });
  const selectedId = reception ? null : params.get("selected");
  const detail = useBookingDetail(selectedId);
  useEffect(() => {
    if (!reception || !params.has("selected")) return;
    setParams(
      (current) => {
        current.delete("selected");
        return current;
      },
      { replace: true },
    );
  }, [reception, params, setParams]);
  const list = (q.data || []).filter(
    (b) =>
      (!todayOnly || b.date === today()) &&
      (!kinds.length || kinds.includes(kindOf(b))) &&
      (!officers.length ||
        (b.restrictedView !== "reception" && officerNames.includes(b.host || ""))),
  );
  const rows = !preview
    ? q.data || []
    : selectBookings(
        list,
        search,
        statuses,
        ["priority", "name", "organization", "host", "time", "status"].includes(sort)
          ? sort
          : "priority",
        desc,
      );
  const total = preview ? rows.length : q.total || 0;
  const pages = Math.max(1, Math.ceil(total / 7));
  const activePage = Math.min(page, pages);
  const accessLost = q.error instanceof ApiError && [401, 403].includes(q.error.status);
  const selected =
    !selectedId || accessLost
      ? undefined
      : preview
        ? q.data?.find((b) => b.id === selectedId)
        : detail.error instanceof ApiError && [401, 403, 404].includes(detail.error.status)
          ? undefined
          : detail.data;
  function update(key: string, value: string) {
    setParams((p) => {
      value ? p.set(key, value) : p.delete(key);
      if (key === "sort") p.delete("dir");
      if (key !== "page") p.delete("page");
      return p;
    });
  }
  function col(key: SortKey, title: string) {
    return (
      <Button
        type="button"
        kind="sort-label"
        onClick={() => {
          setParams((p) => {
            p.set("sort", key);
            p.set("dir", sort === key && !desc ? "desc" : "asc");
            p.delete("page");
            return p;
          });
        }}
      >
        {title}
        <ArrowDownUp size={12} />
      </Button>
    );
  }
  function sortDirection(key: SortKey) {
    return sort === key ? (desc ? "descending" : "ascending") : undefined;
  }
  function resetFilters() {
    setParams((p) => {
      for (const key of ["q", "status", "list-kind", "officer", "sort", "dir", "page"])
        p.delete(key);
      return p;
    });
  }
  const pageRows = preview ? rows.slice((activePage - 1) * 7, activePage * 7) : rows;
  return (
    <>
      <div className="page-intro booking-page-intro">
        <div>
          <h1>{todayOnly ? "Kunjungan hari ini" : "Kunjungan"}</h1>
          <p>
            {todayOnly
              ? dateLabel(today())
              : session?.role === "Receptionist"
                ? "Kunjungan disetujui, pengunjung yang sedang berkunjung, dan kunjungan selesai."
                : "Tinjau permohonan, atur jadwal, dan siapkan kedatangan."}
          </p>
        </div>
        <div className="booking-intro-actions">
          <span className="date-chip">
            {q.data === undefined
              ? "Memuat jumlah…"
              : `${preview ? list.length : q.totalScoped} booking dalam akses Anda`}
          </span>
          {!todayOnly &&
            session?.permissions.includes("reports") &&
            session.permissions.includes("reportExport") && (
              <Button kind="secondary" onClick={() => setExportOpen(true)}>
                <Download size={16} /> Ekspor laporan kunjungan
              </Button>
            )}
          {mayCreate && session?.role === "Protocol Officer" ? (
            <details
              className="appointment-type-menu"
              onKeyDown={(event) => {
                if (event.key === "Escape") {
                  event.currentTarget.open = false;
                  event.currentTarget.querySelector("summary")?.focus();
                }
              }}
              onBlur={(event) => {
                if (!event.currentTarget.contains(event.relatedTarget as Node | null))
                  event.currentTarget.open = false;
              }}
            >
              <summary className="link-button primary">
                <Plus size={17} />
                Buat janji temu
                <ChevronDown size={16} />
              </summary>
              <div className="appointment-type-options">
                <Button
                  type="button"
                  kind="menu-option"
                  aria-label="Kunjungan VIP"
                  onClick={(event) => {
                    createTrigger.current =
                      event.currentTarget.closest("details")?.querySelector("summary") || null;
                    event.currentTarget.closest("details")?.removeAttribute("open");
                    openCreate("standard");
                  }}
                >
                  <strong>Kunjungan VIP</strong>
                  <small>Janji temu untuk Kepala Otorita</small>
                </Button>
                <Button
                  type="button"
                  kind="menu-option"
                  aria-label="Kunjungan Terbatas"
                  onClick={(event) => {
                    createTrigger.current =
                      event.currentTarget.closest("details")?.querySelector("summary") || null;
                    event.currentTarget.closest("details")?.removeAttribute("open");
                    openCreate("restricted");
                  }}
                >
                  <strong>Kunjungan Terbatas</strong>
                  <small>Dibuat dan dikelola Protokol</small>
                </Button>
              </div>
            </details>
          ) : mayCreate ? (
            <Button
              onClick={(event) => {
                createTrigger.current = event.currentTarget;
                setParams((p) => {
                  p.delete("selected");
                  p.set("new", "1");
                  return p;
                });
              }}
            >
              <Plus size={17} />
              Buat janji temu
            </Button>
          ) : null}
        </div>
      </div>
      {notice && (
        <div className="hint" role="status">
          <Check size={18} />
          {notice}
        </div>
      )}
      {creating && session && !mayCreate && (
        <ErrorNotice error="Peran ini tidak dapat membuat janji temu." />
      )}
      {selectedId && session && !q.isPending && !q.error && !selected && (
        <ErrorNotice error="Kunjungan tidak tersedia dalam akses Anda." />
      )}
      <section className="panel section-gap booking-panel">
        <div className="booking-toolbar">
          <SearchField
            aria-label="Cari kunjungan"
            placeholder="Cari nama, instansi, atau tujuan…"
            value={search}
            onChange={(e) => update("q", e.target.value)}
          />
          <CheckboxDropdown
            className="booking-filter"
            label="Status"
            allLabel="Semua status"
            options={statusOptions.map((value) => ({
              value,
              label: labels[value as keyof typeof labels],
            }))}
            selected={statuses}
            onChange={(values) => update("status", values.join(","))}
          />
          {reception && (
            <CheckboxDropdown
              className="booking-filter"
              label="Tujuan kunjungan"
              allLabel="Semua tujuan"
              options={officerOptions}
              selected={officers}
              onChange={(values) => update("officer", values.join(","))}
            />
          )}
          {showKindFilter && (
            <CheckboxDropdown
              className="booking-filter"
              label="Tipe kunjungan"
              allLabel="Semua tipe"
              options={visitKindOptions.map((value) => ({
                value,
                label: visitKindLabels[value],
              }))}
              selected={kinds}
              onChange={(values) => update("list-kind", values.join(","))}
            />
          )}
        </div>
        <QueryState
          pending={q.isPending}
          error={q.error}
          retry={q.refetch}
          hasData={q.data !== undefined}
        >
          {rows.length ? (
            <>
              <div className="booking-mobile-list">
                {pageRows.map((b) => (
                  <article className="booking-mobile-item" key={b.id}>
                    <div className="booking-mobile-heading">
                      <div>
                        <h2>{bookingName(b)}</h2>
                        <VisitKind booking={b} />
                      </div>
                      <StatusBadge
                        status={b.status}
                        timing={b.timing}
                        rescheduled={b.rescheduled}
                      />
                    </div>
                    <p>
                      {b.organization || (isRestricted(b) ? b.reference : "Instansi belum diisi")}{" "}
                      {!isRestricted(b) && `· ${partyLabel(b)}`}
                    </p>
                    <dl>
                      <div>
                        <dt>Jadwal · WITA</dt>
                        <dd>
                          {dateLabel(b.date)} · {b.time}
                        </dd>
                      </div>
                      {b.restrictedView !== "reception" && (
                        <div>
                          <dt>Tujuan</dt>
                          <dd>{b.host || "Belum ditentukan"}</dd>
                        </div>
                      )}
                      {reception && b.restrictedView !== "reception" && (
                        <div>
                          <dt>PA pejabat</dt>
                          <dd>{officerPAs(b.officerPAs)}</dd>
                        </div>
                      )}
                    </dl>
                    {!reception && (
                      <Button
                        kind="secondary"
                        aria-label={`Lihat detail ${bookingName(b)}`}
                        onClick={(event) => {
                          detailTrigger.current = event.currentTarget;
                          update("selected", b.id);
                        }}
                      >
                        Lihat detail <ArrowUpRight size={16} />
                      </Button>
                    )}
                  </article>
                ))}
              </div>
              <TableRegion className="booking-desktop-table" label="Daftar kunjungan">
                <table>
                  <thead>
                    <tr>
                      <th aria-sort={sortDirection("name")}>{col("name", "Pengunjung")}</th>
                      <th aria-sort={sortDirection("host")}>{col("host", "Tujuan kunjungan")}</th>
                      {reception && <th>PA pejabat</th>}
                      <th aria-sort={sortDirection("time")}>{col("time", "Jadwal · WITA")}</th>
                      <th aria-sort={sortDirection("status")}>{col("status", "Status")}</th>
                      {!reception && (
                        <th>
                          <span className="sr-only">Detail</span>
                        </th>
                      )}
                    </tr>
                  </thead>
                  <tbody>
                    {pageRows.map((b) => (
                      <tr key={b.id}>
                        <td>
                          <div className="person-cell">
                            <span className="avatar">
                              {bookingName(b)
                                .split(" ")
                                .map((n) => n[0])
                                .slice(0, 2)
                                .join("")}
                            </span>
                            <div>
                              {reception ? (
                                <strong>
                                  {bookingName(b)} <VisitKind booking={b} />
                                </strong>
                              ) : (
                                <Button
                                  type="button"
                                  kind="row-link"
                                  onClick={(event) => {
                                    detailTrigger.current = event.currentTarget;
                                    update("selected", b.id);
                                  }}
                                >
                                  {bookingName(b)}
                                  <VisitKind booking={b} />
                                </Button>
                              )}
                              <small>
                                {isRestricted(b)
                                  ? b.reference
                                  : `${b.organization || "Instansi belum diisi"} · ${partyLabel(b)}`}
                              </small>
                            </div>
                          </div>
                        </td>
                        <td>
                          {b.restrictedView === "reception"
                            ? "Identitas melalui kode/QR"
                            : b.host || "Belum ditentukan"}
                          <small>{b.room || "Ruangan belum ditentukan"}</small>
                        </td>
                        {reception && (
                          <td>
                            {b.restrictedView === "reception"
                              ? "Identitas melalui kode/QR"
                              : officerPAs(b.officerPAs)}
                          </td>
                        )}
                        <td className="tabular">
                          {b.time}
                          <small>{dateLabel(b.date)}</small>
                        </td>
                        <td>
                          <StatusBadge
                            status={b.status}
                            timing={b.timing}
                            rescheduled={b.rescheduled}
                          />
                        </td>
                        {!reception && (
                          <td>
                            <Button
                              kind="ghost"
                              size="icon"
                              aria-label={`Lihat detail ${bookingName(b)}`}
                              onClick={(event) => {
                                detailTrigger.current = event.currentTarget;
                                update("selected", b.id);
                              }}
                            >
                              <ArrowUpRight size={18} />
                            </Button>
                          </td>
                        )}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </TableRegion>
              <div className="pagination">
                <span>
                  {(activePage - 1) * 7 + 1}–{Math.min(activePage * 7, total)} dari {total} booking
                </span>
                <div>
                  <Button
                    kind="ghost"
                    size="icon"
                    disabled={activePage === 1}
                    aria-label="Halaman sebelumnya"
                    onClick={() => update("page", String(activePage - 1))}
                  >
                    <ChevronLeft size={16} />
                  </Button>
                  <span className="page-number">
                    {activePage} / {pages}
                  </span>
                  <Button
                    kind="ghost"
                    size="icon"
                    disabled={activePage === pages}
                    aria-label="Halaman berikutnya"
                    onClick={() => update("page", String(activePage + 1))}
                  >
                    <ChevronRight size={16} />
                  </Button>
                </div>
              </div>
            </>
          ) : (
            <>
              <Empty
                title="Tidak ada kunjungan yang sesuai"
                text="Coba ubah kata pencarian atau filter status."
              />
              <div className="reset-filter">
                <Button kind="secondary" onClick={resetFilters}>
                  Reset filter
                </Button>
              </div>
            </>
          )}
        </QueryState>
      </section>
      {creating && mayCreate && (
        <Suspense
          fallback={
            <Modal
              open
              title="Buat janji temu"
              description="Menyiapkan formulir janji temu."
              returnFocusTo={createTrigger.current}
              onClose={() => update("new", "")}
            >
              <Loading />
            </Modal>
          }
        >
          {session?.role === "Protocol Officer" &&
          !["standard", "restricted"].includes(kind || "") ? (
            <Modal
              open
              title="Pilih jenis kunjungan"
              description="Tentukan jenis janji temu yang akan dibuat."
              onClose={() => update("new", "")}
              returnFocusTo={createTrigger.current}
            >
              <div className="appointment-type-options inline">
                <Button type="button" kind="menu-option" onClick={() => openCreate("standard")}>
                  <strong>Kunjungan VIP</strong>
                  <small>Janji temu untuk Kepala Otorita</small>
                </Button>
                <Button type="button" kind="menu-option" onClick={() => openCreate("restricted")}>
                  <strong>Kunjungan Terbatas</strong>
                  <small>Dibuat dan dikelola Protokol</small>
                </Button>
              </div>
            </Modal>
          ) : kind === "restricted" ? (
            <RestrictedAppointment
              returnFocusTo={createTrigger.current}
              onClose={() => update("new", "")}
              onSaved={(created) => {
                setNotice(
                  "Kunjungan terbatas berhasil dibuat dan langsung disetujui. Rincian tersedia sesuai hak akses petugas.",
                );
                toast("Kunjungan terbatas berhasil dibuat.");
                setParams({ selected: created.id });
              }}
            />
          ) : (
            <CreateAppointment
              key={session!.role}
              returnFocusTo={createTrigger.current}
              onClose={() => update("new", "")}
              onCreated={(created) => {
                toast("Janji temu berhasil dibuat dan langsung disetujui.");
                setNotice("Janji temu berhasil dibuat dan langsung disetujui.");
                setParams({ selected: created.id });
              }}
            />
          )}
        </Suspense>
      )}
      {!creating && !preview && selectedId && !selected && (
        <Modal
          open
          title="Rincian kunjungan"
          description="Memuat rincian sesuai kewenangan Anda."
          onClose={() => update("selected", "")}
          returnFocusTo={detailTrigger.current}
        >
          {detail.isPending ? (
            <Loading />
          ) : (
            <>
              <ErrorNotice error={detail.error} />
              <Button onClick={() => void detail.refetch()}>Coba lagi</Button>
            </>
          )}
        </Modal>
      )}
      {exportOpen &&
        session?.permissions.includes("reports") &&
        session.permissions.includes("reportExport") && (
          <Modal
            open
            className="report-export-modal"
            title="Ekspor laporan kunjungan"
            description="Pilih periode dan status laporan sesuai lingkup akses Anda."
            onClose={() => setExportOpen(false)}
          >
            <Suspense fallback={<Loading />}>
              <Reports key={session.role} page="logs" />
            </Suspense>
          </Modal>
        )}
      {!creating && selected && session && (
        <BookingDetail
          returnFocusTo={detailTrigger.current}
          booking={selected}
          onClose={() => update("selected", "")}
          onSuccess={setNotice}
        />
      )}
    </>
  );
}
