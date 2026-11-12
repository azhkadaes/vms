import { ApiError, dateLabel, repository, useSession, useVisitSummary } from "@vms/data";
import {
  canExportSummary,
  canReadSummary,
  defaultSummaryFilter,
  reportLabels,
  reportScopeLabels,
  reportStatuses,
  summaryFilterSchema,
} from "@vms/data/reports";
import { Button, Empty, ErrorNotice, Loading, RequiredFieldsHint, RequiredMark } from "@vms/ui";
import { Download, RefreshCw } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router";

export default function Reports({ page }: { page: "analytics" | "logs" }) {
  const { session, role } = useSession();
  const [params, setParams] = useSearchParams();
  const defaults = defaultSummaryFilter();
  const values = {
    from: params.get("from") ?? defaults.from,
    to: params.get("to") ?? defaults.to,
    status: params.get("status") ?? "all",
  };
  const parsed = summaryFilterSchema.safeParse(values);
  const q = useVisitSummary(parsed.success ? parsed.data : null);
  const [dirty, setDirty] = useState(false);
  const [errors, setErrors] = useState<Record<string, string[] | undefined>>({});
  const [exporting, setExporting] = useState(false);
  const [exportError, setExportError] = useState<unknown>();
  const [notice, setNotice] = useState("");
  const lock = useRef(false);
  const mounted = useRef(true);
  const revision = useRef(q.data?.revision);
  revision.current = q.data?.revision;
  const form = useRef<HTMLFormElement>(null);
  const applied = params.toString();
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  useEffect(() => {
    setDirty(false);
    setErrors({});
    setExportError(undefined);
    setNotice("");
  }, [applied]);
  const denied = [q.error, exportError].some(
    (error) => error instanceof ApiError && [401, 403].includes(error.status),
  );
  const paused = q.fetchStatus === "paused";
  const issues = { ...(parsed.success ? {} : parsed.error.flatten().fieldErrors), ...errors };
  const data = q.data;
  const exportable =
    !!session &&
    canExportSummary(session) &&
    !!data?.canExport &&
    !dirty &&
    !q.isFetching &&
    !q.error &&
    !paused &&
    !denied &&
    !exporting &&
    parsed.success;

  async function refresh() {
    setExportError(undefined);
    setNotice("");
    await q.refetch();
  }
  async function download() {
    if (!data || !exportable || lock.current) return;
    lock.current = true;
    setExporting(true);
    setExportError(undefined);
    setNotice("");
    try {
      if (!role) throw new ApiError(401, "Sesi berakhir. Silakan masuk kembali.");
      const result = await repository.exportSummary(data.filter, data.revision, role);
      if (!mounted.current) return;
      if (result.revision !== revision.current)
        throw new ApiError(409, "Ringkasan telah berubah. Perbarui laporan sebelum mengekspor.");
      const url = URL.createObjectURL(new Blob([result.csv], { type: "text/csv;charset=utf-8" }));
      const link = document.createElement("a");
      link.href = url;
      link.download = result.filename;
      link.click();
      URL.revokeObjectURL(url);
      setNotice("CSV ringkasan berhasil dibuat sesuai angka yang ditampilkan.");
    } catch (error) {
      if (!mounted.current) return;
      setExportError(error);
      if (error instanceof ApiError && error.status === 409) {
        const fresh = await q.refetch();
        if (!mounted.current) return;
        if (!fresh.error) {
          setExportError(undefined);
          setNotice("Ringkasan diperbarui. Periksa angka terbaru, lalu ekspor kembali.");
        }
      }
    } finally {
      lock.current = false;
      if (mounted.current) setExporting(false);
    }
  }
  if (!session || !canReadSummary(session))
    return (
      <Empty
        title="Akses laporan tidak tersedia"
        text="Akun Anda memerlukan izin laporan untuk melihat ringkasan."
      />
    );

  return (
    <>
      <div className="page-intro report-intro">
        <div>
          <h1>{page === "analytics" ? "Analitik kunjungan" : "Laporan kunjungan"}</h1>
          <p>
            Ringkasan tanggal dan status sesuai akses Anda. Kunjungan Terbatas tidak disertakan.
          </p>
        </div>
        <div className="report-actions">
          <Button
            type="button"
            kind="secondary"
            disabled={dirty || !parsed.success || q.isFetching || exporting}
            onClick={() => void refresh()}
          >
            <RefreshCw size={16} />
            {q.isFetching ? "Memperbarui…" : "Perbarui laporan"}
          </Button>
          <Button type="button" disabled={!exportable} onClick={() => void download()}>
            <Download size={16} />
            {exporting ? "Membuat CSV…" : "Unduh ringkasan CSV"}
          </Button>
        </div>
      </div>
      <form
        ref={form}
        key={applied}
        className="panel report-filters"
        noValidate
        onChange={() => setDirty(true)}
        onSubmit={(event) => {
          event.preventDefault();
          const fields = new FormData(event.currentTarget);
          const result = summaryFilterSchema.safeParse({
            from: fields.get("from"),
            to: fields.get("to"),
            status: fields.get("status"),
          });
          if (!result.success) {
            const fieldErrors = result.error.flatten().fieldErrors;
            setErrors(fieldErrors);
            const first = ["from", "to", "status"].find(
              (key) => fieldErrors[key as keyof typeof fieldErrors]?.length,
            );
            if (first) form.current?.querySelector<HTMLElement>(`[name="${first}"]`)?.focus();
            return;
          }
          setErrors({});
          setDirty(false);
          setParams(result.data);
        }}
      >
        <RequiredFieldsHint />
        <fieldset disabled={exporting}>
          <legend className="sr-only">Filter laporan</legend>
          <div className="report-filter-grid">
            {(["from", "to"] as const).map((name) => (
              <div className="field" key={name}>
                <label htmlFor={`report-${name}`}>
                  {name === "from" ? "Tanggal awal" : "Tanggal akhir"} <RequiredMark />
                </label>
                <input
                  type="date"
                  id={`report-${name}`}
                  name={name}
                  defaultValue={values[name]}
                  required
                  aria-invalid={!!issues[name]?.length}
                  aria-describedby={issues[name]?.length ? `report-${name}-error` : undefined}
                />
                {issues[name]?.[0] && (
                  <span role="alert" className="field-error" id={`report-${name}-error`}>
                    {issues[name]![0]}
                  </span>
                )}
              </div>
            ))}
            <div className="field">
              <label htmlFor="report-status">
                Status <RequiredMark />
              </label>
              <select
                id="report-status"
                name="status"
                defaultValue={values.status}
                required
                aria-invalid={!!issues.status?.length}
                aria-describedby={issues.status?.length ? "report-status-error" : undefined}
              >
                <option value="all">Semua status</option>
                {reportStatuses.map((status) => (
                  <option key={status} value={status}>
                    {reportLabels[status]}
                  </option>
                ))}
              </select>
              {issues.status?.[0] && (
                <span role="alert" className="field-error" id="report-status-error">
                  Pilih status laporan yang tersedia.
                </span>
              )}
            </div>
            <Button type="submit">Terapkan filter</Button>
          </div>
        </fieldset>
        <p className="muted">
          Rentang memakai tanggal kunjungan dalam WITA, maksimal 366 hari pada preview.
        </p>
        {dirty && (
          <p role="status" className="hint">
            Terapkan perubahan filter sebelum memperbarui atau mengekspor ringkasan.
          </p>
        )}
      </form>
      {notice && (
        <p role="status" className="hint section-gap">
          {notice}
        </p>
      )}
      <ErrorNotice error={denied ? null : exportError} />
      {!parsed.success ? (
        <Empty
          title="Filter laporan belum valid"
          text="Perbaiki tanggal atau status pada filter untuk memuat ringkasan."
        />
      ) : denied ? (
        <div className="panel report-feedback">
          <ErrorNotice error={q.error || exportError} />
          <Button type="button" kind="secondary" onClick={() => void refresh()}>
            Coba lagi
          </Button>
        </div>
      ) : (
        <section
          className="panel report-counts section-gap"
          aria-label="Ringkasan kunjungan"
          aria-busy={q.isFetching}
        >
          {q.error && (
            <div className="report-feedback">
              <ErrorNotice error={q.error} />
              {data && (
                <p className="muted">
                  Angka terakhir tetap ditampilkan. Perbarui laporan sebelum mengekspor.
                </p>
              )}
              <Button type="button" kind="secondary" onClick={() => void refresh()}>
                Coba lagi
              </Button>
            </div>
          )}
          {paused && (
            <p role="status" className="hint">
              Anda sedang offline. Sambungkan kembali untuk memuat laporan terbaru.
            </p>
          )}
          {q.isPending && !paused ? (
            <Loading />
          ) : !data ? (
            !q.error && (
              <Empty
                title="Data ringkasan belum tersedia"
                text="Muat ringkasan setelah koneksi tersedia. Angka belum dapat ditampilkan."
              />
            )
          ) : (
            <>
              <div className="report-summary-heading">
                <div>
                  <h2>
                    {dateLabel(data.filter.from)} – {dateLabel(data.filter.to)}
                  </h2>
                  <p className="muted">{reportScopeLabels[data.scope]}</p>
                  <small className="muted">
                    Diperbarui{" "}
                    <time dateTime={data.generatedAt}>
                      {new Date(data.generatedAt).toLocaleString("id-ID", {
                        timeZone: "Asia/Makassar",
                      })}
                    </time>{" "}
                    WITA
                  </small>
                </div>
                <div className="report-total">
                  <strong data-testid="report-total">{data.total}</strong>
                  <span>kunjungan sesuai filter</span>
                </div>
              </div>
              {data.total === 0 && (
                <p role="status" className="hint">
                  Tidak ada kunjungan sesuai filter dalam akses Anda.
                </p>
              )}
              <table className="report-summary-table">
                <caption className="sr-only">
                  Jumlah kunjungan per status untuk periode yang ditampilkan
                </caption>
                <thead>
                  <tr>
                    <th scope="col">Status</th>
                    <th scope="col">Jumlah</th>
                  </tr>
                </thead>
                <tbody>
                  {reportStatuses
                    .filter(
                      (status) => data.filter.status === "all" || data.filter.status === status,
                    )
                    .map((status) => (
                      <tr key={status}>
                        <th scope="row">
                          {reportLabels[status]}
                          <meter
                            aria-label={reportLabels[status]}
                            min={0}
                            max={Math.max(1, data.total)}
                            value={data.counts[status]}
                          />
                        </th>
                        <td>{data.counts[status]}</td>
                      </tr>
                    ))}
                </tbody>
              </table>
              {!data.canExport && (
                <p className="muted">
                  Akun Anda dapat membaca ringkasan; izin ekspor diperlukan untuk mengunduhnya.
                </p>
              )}
            </>
          )}
        </section>
      )}
    </>
  );
}
