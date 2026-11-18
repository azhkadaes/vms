import { Popover, PopoverContent, PopoverTrigger } from "@vms/ui/popover";
import { ApiError, dateLabel, useVisitSummary } from "@vms/data";
import { matchesFilter, readFilter } from "@vms/data/report-filter";
import {
  defaultSummaryFilter,
  reportLabels,
  reportScopeLabels,
  reportStatuses,
  summaryFilterSchema,
} from "@vms/data/reports";
import { Button, Empty, ErrorNotice, Loading } from "@vms/ui";
import { ChevronDown, RefreshCw, SlidersHorizontal } from "lucide-react";
import { useState } from "react";
import ReportMultiSelect from "./ReportMultiSelect";

export default function DashboardAnalytics() {
  const [filter, setFilter] = useState(defaultSummaryFilter);
  const [open, setOpen] = useState(false);
  const [errors, setErrors] = useState<Record<string, string[] | undefined>>({});
  const q = useVisitSummary(filter);
  const denied = q.error instanceof ApiError && [401, 403].includes(q.error.status);
  const data = denied ? undefined : q.data;
  const period = `${dateLabel(filter.from)} – ${dateLabel(filter.to)}`;
  return (
    <section
      className="panel dashboard-analytics"
      aria-labelledby="analytics-title"
      aria-busy={q.isFetching}
    >
      <header className="analytics-heading">
        <div>
          <h2 id="analytics-title">Analitik kunjungan</h2>
          <p>{period}</p>
        </div>
        <div className="analytics-tools">
          <Popover open={open} onOpenChange={setOpen}>
            <PopoverTrigger asChild>
              <Button kind="ghost" aria-label="Filter statistik">
                <SlidersHorizontal size={15} /> Filter <ChevronDown size={13} />
              </Button>
            </PopoverTrigger>

            <PopoverContent
              className="analytics-filter-popover"
              align="end"
              sideOffset={8}
              collisionPadding={12}
              aria-label="Filter statistik kunjungan"
            >
              <form
                key={JSON.stringify(filter)}
                noValidate
                onSubmit={(event) => {
                  event.preventDefault();
                  const fields = new FormData(event.currentTarget);
                  const result = summaryFilterSchema.safeParse({
                    from: fields.get("from"),
                    to: fields.get("to"),
                    status: readFilter(fields, "status", "all"),
                  });
                  if (!result.success) {
                    setErrors(result.error.flatten().fieldErrors);
                    const name = result.error.issues[0]?.path[0];
                    if (name)
                      event.currentTarget
                        .querySelector<HTMLElement>(`#analytics-${String(name)}`)
                        ?.focus();
                    return;
                  }
                  setFilter(result.data);
                  setErrors({});
                  setOpen(false);
                }}
              >
                <h3>Filter statistik</h3>
                <div className="analytics-date-fields">
                  {(["from", "to"] as const).map((name) => (
                    <div className="field" key={name}>
                      <label htmlFor={`analytics-${String(name)}`}>
                        {name === "from" ? "Dari tanggal" : "Sampai tanggal"}
                      </label>
                      <input
                        id={`analytics-${String(name)}`}
                        name={name}
                        type="date"
                        defaultValue={filter[name]}
                        required
                        aria-invalid={!!errors[name]}
                        aria-describedby={
                          errors[name] ? `analytics-${String(name)}-error` : undefined
                        }
                      />
                      {errors[name]?.[0] && (
                        <span
                          className="field-error"
                          role="alert"
                          id={`analytics-${String(name)}-error`}
                        >
                          {errors[name]![0]}
                        </span>
                      )}
                    </div>
                  ))}
                </div>
                <ReportMultiSelect
                  id="analytics-status"
                  name="status"
                  label="Status"
                  placeholder="Semua status"
                  options={reportStatuses.map((value) => ({ value, label: reportLabels[value] }))}
                  value={filter.status}
                  error={errors.status?.[0]}
                />
                <div className="analytics-filter-actions">
                  <Button
                    kind="ghost"
                    type="button"
                    onClick={() => {
                      setFilter(defaultSummaryFilter());
                      setErrors({});
                      setOpen(false);
                    }}
                  >
                    Bulan ini
                  </Button>
                  <Button type="submit">Terapkan</Button>
                </div>
              </form>
            </PopoverContent>
          </Popover>
          <Button
            kind="ghost"
            aria-label="Perbarui statistik"
            title="Perbarui statistik"
            disabled={q.isFetching}
            onClick={() => void q.refetch()}
          >
            <RefreshCw size={15} className={q.isFetching ? "spin" : undefined} />
          </Button>
        </div>
      </header>
      <ErrorNotice error={q.error} />
      {q.fetchStatus === "paused" && (
        <p className="hint" role="status">
          Anda sedang offline. Statistik diperbarui setelah terhubung kembali.
        </p>
      )}
      {q.isPending && q.fetchStatus !== "paused" ? (
        <Loading />
      ) : data ? (
        <>
          <div className="analytics-cards">
            <article className="analytics-card analytics-card-total">
              <span>Total kunjungan</span>
              <strong>{data.total.toLocaleString("id-ID")}</strong>
              <small>Sesuai filter</small>
            </article>
            {reportStatuses
              .filter((status) => matchesFilter(status, data.filter.status, "all"))
              .map((status) => (
                <article className="analytics-card" key={status}>
                  <span>{reportLabels[status]}</span>
                  <strong>{data.counts[status].toLocaleString("id-ID")}</strong>
                  <meter
                    aria-label={reportLabels[status]}
                    min={0}
                    max={Math.max(1, data.total)}
                    value={data.counts[status]}
                  />
                </article>
              ))}
          </div>
          <footer className="analytics-footnote">
            {reportScopeLabels[data.scope]} · Tidak termasuk kunjungan terbatas
            {q.error && " · Menampilkan data terakhir"}
          </footer>
        </>
      ) : (
        !q.error && (
          <Empty
            title="Statistik belum tersedia"
            text="Statistik akan tampil setelah data berhasil dimuat."
          />
        )
      )}
    </section>
  );
}
