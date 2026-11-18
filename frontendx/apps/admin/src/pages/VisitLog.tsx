import {
  ApiError,
  dateLabel,
  labels,
  repository,
  sessionKey,
  today,
  useSession,
  useVisitLog,
} from "@vms/data";
import {
  canReadVisitLog,
  defaultVisitLogFilter,
  visitKindLabels,
  visitLogFilterSchema,
  visitLogSorts,
  type VisitLogFilter,
  type VisitLogRow,
} from "@vms/data/visit-log";
import { reportStatuses } from "@vms/data/reports";
import { readFilter, reportFilterParams, selectedFilters } from "@vms/data/report-filter";
import { Select, Button, Empty, ErrorNotice, Loading, Modal, StatusBadge } from "@vms/ui";
import {
  ArrowDown,
  ArrowDownUp,
  ArrowUp,
  ArrowUpRight,
  CalendarDays,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  Download,
  FileSpreadsheet,
  FileText,
  ListFilter,
  RefreshCw,
  Search,
} from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router";
import ReportMultiSelect from "../features/ReportMultiSelect";

const filterKeys = ["from", "to", "q", "status", "kind", "officer", "room", "sort", "dir"] as const;
const number = (value: number) => value.toLocaleString("id-ID");
function timestamp(value: string | null) {
  return value
    ? new Intl.DateTimeFormat("id-ID", {
        timeZone: "Asia/Makassar",
        dateStyle: "short",
        timeStyle: "short",
      }).format(new Date(value))
    : "—";
}
export default function VisitLog() {
  const { session } = useSession();
  const [params, setParams] = useSearchParams();
  const defaults = defaultVisitLogFilter();
  const values = Object.fromEntries(
    filterKeys.map((key) => [key, readFilter(params, key, defaults[key])]),
  );
  const parsed = visitLogFilterSchema.safeParse(values);
  const filter = parsed.success ? parsed.data : defaults;
  const applied = JSON.stringify(values);
  const page = Math.min(100000, Math.max(1, Math.floor(Number(params.get("page")) || 1)));
  const pageSize = [10, 25, 50].includes(Number(params.get("size")))
    ? Number(params.get("size"))
    : 10;
  const q = useVisitLog(parsed.success ? filter : null, page, pageSize);
  const [dirty, setDirty] = useState(false);
  const [resetVersion, setResetVersion] = useState(0);
  const [issues, setIssues] = useState<Record<string, string[] | undefined>>({});
  const [exportOpen, setExportOpen] = useState(false);
  const [format, setFormat] = useState<"csv" | "pdf">("csv");
  const [exporting, setExporting] = useState(false);
  const [exportError, setExportError] = useState<unknown>();
  const [notice, setNotice] = useState("");
  const form = useRef<HTMLFormElement>(null);
  const trigger = useRef<HTMLElement | null>(null);
  const lock = useRef(false);
  const mounted = useRef(true);
  const context = `${sessionKey(session)}:${applied}`;
  const latest = useRef(context);
  latest.current = context;
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  useEffect(() => {
    setDirty(false);
    setIssues({});
    setNotice("");
    setExportOpen(false);
    setExportError(undefined);
  }, [applied]);
  const denied = [q.error, exportError].some(
    (error) => error instanceof ApiError && [401, 403].includes(error.status),
  );
  const data = denied ? undefined : q.data;
  const selected = data?.items.find((b) => b.id === params.get("selected"));
  const errors = { ...(!parsed.success ? parsed.error.flatten().fieldErrors : {}), ...issues };
  const paused = q.fetchStatus === "paused";
  const canExport =
    !!data?.canExport &&
    !!session?.permissions.includes("reportExport") &&
    !dirty &&
    !q.isFetching &&
    !q.error &&
    !paused &&
    !exporting;
  const pages = Math.max(1, Math.ceil((data?.total || 0) / pageSize));
  const officerOptions = (data?.officers || []).map(({ id, name }) => ({ value: id, label: name }));
  for (const value of selectedFilters(filter.officer, ""))
    if (!officerOptions.some((option) => option.value === value))
      officerOptions.push({ value, label: value });
  const roomOptions = [
    ...new Set([...(data?.rooms || []), ...selectedFilters(filter.room, "")]),
  ].map((value) => ({ value, label: value }));
  const activeFilters = [
    filter.q,
    filter.from || filter.to,
    filter.status !== "all",
    filter.kind !== "all",
    filter.officer,
    filter.room,
  ].filter(Boolean).length;

  function markDirty() {
    setDirty(true);
    setIssues({});
  }
  function apply(next: VisitLogFilter) {
    setParams((p) => {
      for (const key of filterKeys) p.delete(key);
      const changed = Object.fromEntries(
        filterKeys.filter((key) => next[key] !== defaults[key]).map((key) => [key, next[key]]),
      );
      for (const [key, value] of reportFilterParams(changed)) p.append(key, value);
      p.delete("page");
      p.delete("selected");
      return p;
    });
    setDirty(false);
    setIssues({});
  }
  function update(key: string, value: string) {
    setParams((p) => {
      value ? p.set(key, value) : p.delete(key);
      if (key !== "page" && key !== "selected") p.delete("page");
      return p;
    });
  }
  async function refresh() {
    setExportError(undefined);
    setNotice("");
    await q.refetch();
  }
  async function download() {
    if (!data || !session || !canExport || lock.current) return;
    lock.current = true;
    setExporting(true);
    setExportError(undefined);
    setNotice("");
    const started = context;
    try {
      const snapshot = await repository.exportVisitLog(data.filter, data.revision, session.role);
      const exporter = await import("../features/visitLogExport");
      if (!mounted.current || latest.current !== started) return;
      await exporter.downloadVisitLog(
        snapshot,
        format,
        () => mounted.current && latest.current === started,
      );
      if (!mounted.current || latest.current !== started) return;
      setExportOpen(false);
      setNotice(
        `${format.toUpperCase()} berhasil diunduh: ${number(snapshot.items.length)} kunjungan sesuai filter.`,
      );
    } catch (error) {
      if (!mounted.current || latest.current !== started) return;
      setExportError(error);
      if (error instanceof ApiError && error.status === 409) {
        setExportOpen(false);
        const fresh = await q.refetch();
        if (mounted.current && latest.current === started && !fresh.error) {
          setExportError(undefined);
          setNotice("Log diperbarui. Periksa hasil terbaru, lalu ekspor kembali.");
        }
      }
    } finally {
      lock.current = false;
      if (mounted.current) setExporting(false);
    }
  }
  function col(key: VisitLogFilter["sort"], title: string) {
    const Icon = filter.sort === key ? (filter.dir === "desc" ? ArrowDown : ArrowUp) : ArrowDownUp;
    return (
      <th
        aria-sort={
          filter.sort === key ? (filter.dir === "desc" ? "descending" : "ascending") : undefined
        }
      >
        <button
          onClick={() =>
            apply({
              ...filter,
              sort: key,
              dir: filter.sort === key && filter.dir === "asc" ? "desc" : "asc",
            })
          }
        >
          {title}
          <Icon size={13} aria-hidden="true" />
        </button>
      </th>
    );
  }
  function openDetail(b: VisitLogRow, element: HTMLElement) {
    trigger.current = element;
    update("selected", b.id);
  }

  if (!session || !canReadVisitLog(session))
    return (
      <Empty
        title="Akses log tidak tersedia"
        text="Halaman ini tersedia untuk Super Admin dengan izin laporan."
      />
    );
  return (
    <>
      <div className="page-intro visit-log-intro">
        <div>
          <h1>Kunjungan</h1>
          <p>Pantau seluruh kunjungan dan unduh laporan dalam satu tempat.</p>
        </div>
        <div className="visit-log-actions">
          <Button
            kind="ghost"
            disabled={q.isFetching || exporting || !parsed.success}
            onClick={() => void refresh()}
            aria-label="Perbarui log"
          >
            <RefreshCw size={16} className={q.isFetching ? "spin" : ""} />
            <span>Perbarui</span>
          </Button>
          <Button disabled={!canExport} onClick={() => setExportOpen(true)}>
            <Download size={16} />
            Ekspor laporan kunjungan
            <ChevronDown size={15} />
          </Button>
        </div>
      </div>
      <div className="visit-log-meta">
        <span>
          <CalendarDays size={14} />
          {filter.from || filter.to
            ? `${filter.from ? dateLabel(filter.from) : "Awal pencatatan"} – ${filter.to ? dateLabel(filter.to) : "Terbaru"}`
            : "Semua waktu"}
        </span>
        <span>
          {data ? `Diperbarui ${timestamp(data.generatedAt)} WITA` : "Data belum tersedia"}
        </span>
      </div>
      {notice && (
        <p className="notice" role="status">
          {notice}
        </p>
      )}
      <ErrorNotice error={exportError} />
      {q.error && data && (
        <div className="notice" role="status">
          <ErrorNotice error={q.error} />
          Data terakhir ditampilkan. Perbarui log sebelum mengekspor.
        </div>
      )}
      {paused && (
        <p className="notice" role="status">
          Anda sedang offline. Hubungkan kembali untuk memuat log terbaru.
        </p>
      )}
      <section className="panel visit-log-panel" aria-labelledby="visit-log-table-title">
        <div className="visit-log-heading">
          <div>
            <h2 id="visit-log-table-title">Daftar kunjungan</h2>
            <p>Seluruh jadwal, status, dan riwayat kehadiran.</p>
          </div>
          {activeFilters > 0 && (
            <span className="date-chip">
              <ListFilter size={14} />
              {activeFilters} filter aktif
            </span>
          )}
        </div>
        <form
          key={`${applied}:${resetVersion}`}
          ref={form}
          className="visit-log-filters"
          onChange={markDirty}
          noValidate
          onSubmit={(e) => {
            e.preventDefault();
            const fields = new FormData(e.currentTarget);
            const result = visitLogFilterSchema.safeParse({
              ...Object.fromEntries(
                filterKeys.map((key) => [
                  key,
                  readFilter(
                    fields,
                    key,
                    key === "sort" || key === "dir" ? filter[key] : defaults[key],
                  ),
                ]),
              ),
            });
            if (!result.success) {
              const fieldErrors = result.error.flatten().fieldErrors;
              setIssues(fieldErrors);
              const key = Object.keys(fieldErrors)[0];
              const invalid =
                form.current?.querySelector<HTMLElement>(`#visit-log-${key}`) ||
                form.current?.querySelector<HTMLElement>(`[name="${key}"]`);
              invalid?.focus();
              return;
            }
            apply(result.data);
          }}
        >
          <div className="visit-log-search-row">
            <div className="field search-field">
              <label className="sr-only" htmlFor="visit-log-search">
                Cari kunjungan
              </label>
              <Search size={17} aria-hidden="true" />
              <input
                id="visit-log-search"
                name="q"
                type="search"
                maxLength={200}
                placeholder="Cari nama, instansi, tujuan, atau referensi…"
                defaultValue={filter.q}
              />
            </div>
            <div className="visit-log-periods" aria-label="Periode cepat">
              {["Semua waktu", "Bulan ini", "Hari ini"].map((label, i) => {
                const current = today();
                const from = i === 0 ? "" : i === 1 ? `${current.slice(0, 7)}-01` : current;
                const to =
                  i === 0
                    ? ""
                    : i === 1
                      ? new Date(
                          Date.UTC(Number(current.slice(0, 4)), Number(current.slice(5, 7)), 0),
                        )
                          .toISOString()
                          .slice(0, 10)
                      : current;
                return (
                  <button
                    type="button"
                    key={label}
                    aria-pressed={filter.from === from && filter.to === to}
                    onClick={() => apply({ ...filter, from, to })}
                  >
                    {label}
                  </button>
                );
              })}
            </div>
          </div>
          <div className="visit-log-filter-grid form-density-compact">
            {(
              [
                ["from", "Dari tanggal"],
                ["to", "Sampai tanggal"],
              ] as const
            ).map(([key, label]) => (
              <div className="field" key={key}>
                <label htmlFor={`visit-log-${key}`}>{label}</label>
                <input
                  id={`visit-log-${key}`}
                  name={key}
                  type="date"
                  defaultValue={filter[key]}
                  aria-invalid={!!errors[key]}
                  aria-describedby={errors[key] ? `visit-log-${key}-error` : undefined}
                />
                {errors[key] && (
                  <span className="field-error" id={`visit-log-${key}-error`}>
                    {errors[key]?.[0]}
                  </span>
                )}
              </div>
            ))}
            {([
              [
                "status", "Status", "Semua status",
                reportStatuses.map((value) => ({ value, label: labels[value] })),
              ],
              [
                "kind", "Jenis kunjungan", "Semua jenis",
                Object.entries(visitKindLabels).map(([value, label]) => ({ value, label })),
              ],
              ["officer", "Tujuan kunjungan", "Semua tujuan", officerOptions],
              ["room", "Ruangan", "Semua ruangan", roomOptions],
            ] as const).map(([name, label, placeholder, options]) => (
              <ReportMultiSelect
                key={name}
                id={`visit-log-${name}`}
                name={name}
                label={label}
                placeholder={placeholder}
                options={options}
                value={filter[name]}
                empty={name === "officer" || name === "room" ? "" : "all"}
                error={errors[name]?.[0]}
                disabled={exporting}
                onChange={markDirty}
              />
            ))}
          </div>
          <div className="visit-log-filter-footer">
            <span>
              {dirty
                ? "Terapkan untuk memperbarui hasil dan laporan."
                : "Ringkasan dan ekspor mengikuti filter yang diterapkan."}
            </span>
            <div>
              <Button
                kind="ghost"
                type="button"
                onClick={() => {
                  apply(defaults);
                  setResetVersion((version) => version + 1);
                }}
              >
                Reset filter
              </Button>
              <Button kind="secondary" type="submit">
                <ListFilter size={15} />
                Terapkan filter
              </Button>
            </div>
          </div>
        </form>
        <div className="visit-log-results">
          <span role="status">
            {data
              ? `${number(data.total)} kunjungan ditemukan`
              : q.isPending && parsed.success
                ? "Memuat kunjungan…"
                : "Log belum tersedia"}
          </span>
          <div className="visit-log-sort">
            <label htmlFor="visit-log-sort">Urutkan</label>
            <Select
              id="visit-log-sort"
              value={`${filter.sort}:${filter.dir}`}
              onChange={(e) => {
                const [sort, dir] = e.target.value.split(":");
                apply({
                  ...filter,
                  sort: sort as VisitLogFilter["sort"],
                  dir: dir as "asc" | "desc",
                });
              }}
            >
              {visitLogSorts.flatMap((key) =>
                ["asc", "desc"].map((dir) => (
                  <option value={`${key}:${dir}`} key={`${key}:${dir}`}>
                    {
                      {
                        date: "Jadwal",
                        name: "Pengunjung",
                        organization: "Instansi",
                        host: "Tujuan",
                        status: "Status",
                        kind: "Jenis",
                        visitors: "Jumlah orang",
                      }[key]
                    }{" "}
                    ·{" "}
                    {key === "date"
                      ? dir === "desc"
                        ? "Terbaru"
                        : "Terlama"
                      : dir === "asc"
                        ? "Naik"
                        : "Turun"}
                  </option>
                )),
              )}
            </Select>
          </div>
        </div>
        {!parsed.success ? (
          <div className="empty" role="alert">
            <p>Filter pada tautan tidak valid. Reset filter untuk memuat log.</p>
            <Button kind="secondary" onClick={() => apply(defaults)}>
              Reset filter log
            </Button>
          </div>
        ) : denied || (!data && q.error) ? (
          <div className="empty">
            <ErrorNotice error={q.error || exportError} />
            <Button kind="secondary" onClick={() => void refresh()}>
              Coba lagi
            </Button>
          </div>
        ) : !data ? (
          paused ? (
            <Empty title="Log belum tersedia" text="Hubungkan kembali untuk memuat data." />
          ) : (
            <Loading />
          )
        ) : data.items.length ? (
          <>
            <div
              className="table-wrap visit-log-table"
              tabIndex={0}
              role="region"
              aria-label="Tabel log kunjungan"
            >
              <table>
                <caption className="sr-only">
                  Log kunjungan sesuai filter. Klik judul kolom untuk mengurutkan.
                </caption>
                <thead>
                  <tr>
                    {col("name", "Pengunjung")}
                    {col("kind", "Jenis")}
                    {col("date", "Jadwal · WITA")}
                    {col("host", "Tujuan kunjungan")}
                    <th>Keperluan</th>
                    {col("visitors", "Orang")}
                    {col("status", "Status")}
                    <th>Kehadiran · WITA</th>
                    <th>
                      <span className="sr-only">Detail</span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {data.items.map((b) => (
                    <tr key={b.id}>
                      <td>
                        <button
                          className="row-link"
                          onClick={(e) => openDetail(b, e.currentTarget)}
                        >
                          {b.name}
                        </button>
                        <small>{b.organization || "—"}</small>
                        <span className="visit-log-reference">{b.reference}</span>
                      </td>
                      <td>
                        <span className={`visit-log-kind visit-log-kind-${b.kind}`}>
                          {visitKindLabels[b.kind]}
                        </span>
                      </td>
                      <td className="tabular">
                        <strong>{dateLabel(b.date)}</strong>
                        <small>
                          {b.time} · {b.duration} menit
                        </small>
                      </td>
                      <td>
                        {b.host || "Belum ditentukan"}
                        <small>{b.room || "Ruangan belum ditentukan"}</small>
                      </td>
                      <td className="visit-log-purpose">{b.purpose || "—"}</td>
                      <td className="tabular">
                        {b.visitors === null ? (
                          <span title="Jumlah pengunjung terbatas tidak dicatat">—</span>
                        ) : (
                          b.visitors
                        )}
                      </td>
                      <td>
                        <StatusBadge
                          status={b.status === "missed" ? "approved" : b.status}
                          timing={{
                            displayStatus: b.status,
                            serverTime: 0,
                            opensAt: 0,
                            endsAt: 0,
                            canCancel: false,
                            canCheckIn: false,
                            canCheckOut: false,
                          }}
                        />
                      </td>
                      <td className="visit-log-attendance">
                        <span>
                          Masuk <strong>{timestamp(b.checkedInAt)}</strong>
                        </span>
                        <span>
                          Keluar <strong>{timestamp(b.checkedOutAt)}</strong>
                        </span>
                      </td>
                      <td>
                        <Button
                          kind="ghost"
                          aria-label={`Lihat detail ${b.name}`}
                          onClick={(e) => openDetail(b, e.currentTarget)}
                        >
                          <ArrowUpRight size={17} />
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <div className="pagination visit-log-pagination">
              <div>
                <label htmlFor="visit-log-size">Baris per halaman</label>
                <Select
                  id="visit-log-size"
                  value={pageSize}
                  onChange={(e) => update("size", e.target.value)}
                >
                  {[10, 25, 50].map((size) => (
                    <option key={size}>{size}</option>
                  ))}
                </Select>
              </div>
              <span>
                {(data.page - 1) * pageSize + 1}–{Math.min(data.page * pageSize, data.total)} dari{" "}
                {number(data.total)}
              </span>
              <div>
                <Button
                  kind="ghost"
                  disabled={data.page === 1}
                  aria-label="Halaman sebelumnya"
                  onClick={() => update("page", String(data.page - 1))}
                >
                  <ChevronLeft size={16} />
                </Button>
                <span className="page-number">
                  {data.page} / {pages}
                </span>
                <Button
                  kind="ghost"
                  disabled={data.page >= pages}
                  aria-label="Halaman berikutnya"
                  onClick={() => update("page", String(data.page + 1))}
                >
                  <ChevronRight size={16} />
                </Button>
              </div>
            </div>
          </>
        ) : (
          <Empty
            title="Tidak ada kunjungan sesuai filter"
            text="Coba kata pencarian lain atau perluas rentang tanggal."
          />
        )}
        {data && !data.canExport && (
          <p className="visit-log-footnote">
            Izin ekspor laporan diperlukan untuk mengunduh CSV atau PDF.
          </p>
        )}
      </section>
      {selected && (
        <Modal
          open
          title="Detail log kunjungan"
          description={`${selected.reference} · ${visitKindLabels[selected.kind]}`}
          returnFocusTo={trigger.current}
          onClose={() => update("selected", "")}
        >
          <dl className="visit-log-detail">
            {Object.entries({
              Pengunjung: selected.name,
              Instansi: selected.organization || "—",
              Jenis: visitKindLabels[selected.kind],
              Jadwal: `${dateLabel(selected.date)}, ${selected.time} WITA`,
              Durasi: `${selected.duration} menit`,
              Tujuan: selected.host || "—",
              Ruangan: selected.room || "—",
              Keperluan: selected.purpose || "—",
              "Jumlah pengunjung":
                selected.visitors === null ? "Tidak dicatat" : `${selected.visitors} orang`,
              Status: labels[selected.status],
              "Check in": timestamp(selected.checkedInAt),
              "Check out": timestamp(selected.checkedOutAt),
            }).map(([key, value]) => (
              <div key={key}>
                <dt>{key}</dt>
                <dd>{value}</dd>
              </div>
            ))}
          </dl>
        </Modal>
      )}
      {exportOpen && (
        <Modal
          open
          title="Ekspor laporan"
          description="Unduh seluruh hasil sesuai filter dan urutan yang diterapkan."
          dismissible={!exporting}
          onClose={() => setExportOpen(false)}
        >
          <ErrorNotice error={exportError} />
          <p className="notice">
            {number(data?.total || 0)} kunjungan akan diekspor, mencakup seluruh halaman.
          </p>
          <fieldset className="visit-log-export-options" disabled={exporting}>
            <legend>Format laporan</legend>
            {(
              [
                [
                  "csv",
                  FileSpreadsheet,
                  "CSV",
                  "Data lengkap untuk diolah di Excel atau spreadsheet.",
                ],
                [
                  "pdf",
                  FileText,
                  "PDF",
                  "Laporan siap cetak dengan ringkasan dan tabel kunjungan.",
                ],
              ] as const
            ).map(([value, Icon, label, description]) => (
              <label key={value} className={format === value ? "selected" : ""}>
                <input
                  type="radio"
                  name="export-format"
                  value={value}
                  checked={format === value}
                  onChange={() => setFormat(value)}
                />
                <Icon size={22} />
                <span>
                  <strong>{label}</strong>
                  <small>{description}</small>
                </span>
              </label>
            ))}
          </fieldset>
          <div className="form-actions">
            <Button kind="ghost" disabled={exporting} onClick={() => setExportOpen(false)}>
              Batal
            </Button>
            <Button disabled={!canExport} onClick={() => void download()}>
              <Download size={16} />
              {exporting ? "Menyiapkan laporan…" : `Ekspor ${format.toUpperCase()}`}
            </Button>
          </div>
        </Modal>
      )}
    </>
  );
}
