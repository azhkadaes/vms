import { z } from "zod";
import {
  generalReportRecords,
  getVisitTiming,
  labels,
  scope,
  statuses,
  today,
  type Booking,
  type DisplayStatus,
  type Session,
} from "./domain.ts";

export const reportStatuses = [...statuses, "missed"] as const;
const statusSchema = z.enum(["all", ...reportStatuses]);
const dateSchema = z
  .string()
  .refine(
    (value) =>
      /^\d{4}-\d{2}-\d{2}$/.test(value) &&
      Number.isFinite(Date.parse(`${value}T12:00:00Z`)) &&
      new Date(`${value}T12:00:00Z`).toISOString().slice(0, 10) === value,
    "Pilih tanggal yang valid.",
  );
export const summaryFilterSchema = z
  .object({ from: dateSchema, to: dateSchema, status: statusSchema })
  .strict()
  .refine((value) => value.from <= value.to, {
    path: ["to"],
    message: "Tanggal akhir harus sama atau sesudah tanggal awal.",
  })
  .refine((value) => Date.parse(value.to) - Date.parse(value.from) <= 365 * 86400_000, {
    path: ["to"],
    message: "Rentang laporan maksimal 366 hari pada preview.",
  });
export type SummaryFilter = z.infer<typeof summaryFilterSchema>;
export const reportLabels: Record<DisplayStatus, string> = {
  ...labels,
};
export function defaultSummaryFilter(): SummaryFilter {
  const current = today();
  const [year, month] = current.split("-").map(Number);
  const lastDay = new Date(Date.UTC(year, month, 0)).getUTCDate();
  return {
    from: `${current.slice(0, 7)}-01`,
    to: `${current.slice(0, 7)}-${lastDay}`,
    status: "all",
  };
}
export function canReadSummary(session: Session) {
  return session.permissions.includes("reports");
}
export function canExportSummary(session: Session) {
  return canReadSummary(session) && session.permissions.includes("reportExport");
}
const countsSchema = z
  .object(
    Object.fromEntries(
      reportStatuses.map((status) => [status, z.number().int().nonnegative()]),
    ) as Record<DisplayStatus, z.ZodNumber>,
  )
  .strict();
export const summaryResponseSchema = z
  .object({
    filter: summaryFilterSchema,
    scope: z.enum(["organization", "assigned", "authorized", "attendance"]),
    generatedAt: z.string().datetime(),
    canExport: z.boolean(),
    revision: z.string().regex(/^[a-f0-9]{64}$/),
    counts: countsSchema,
    total: z.number().int().nonnegative(),
  })
  .strict()
  .refine(
    (value) =>
      reportStatuses.reduce((total, status) => total + value.counts[status], 0) === value.total,
    "Jumlah ringkasan tidak sesuai.",
  );
export type VisitSummary = z.infer<typeof summaryResponseSchema>;
export const reportScopeLabels: Record<VisitSummary["scope"], string> = {
  organization: "Ringkasan organisasi",
  assigned: "Pejabat dalam penugasan Anda",
  attendance: "Lingkup penerimaan",
  authorized: "Lingkup kunjungan yang diizinkan",
};
export const summaryExportSchema = z
  .object({
    filename: z.string().regex(/^kunjungan-ringkasan-[\d-]+\.csv$/),
    csv: z.string(),
    revision: z.string().regex(/^[a-f0-9]{64}$/),
  })
  .strict();
export type SummaryExport = z.infer<typeof summaryExportSchema>;
type SummaryRecord = Pick<
  Booking,
  "visitKind" | "date" | "time" | "duration" | "status" | "officerId" | "vip"
>;

export function aggregateVisits(
  records: SummaryRecord[],
  session: Session,
  filter: SummaryFilter,
  now: number,
): Omit<VisitSummary, "revision"> {
  if (!canReadSummary(session)) throw new Error("Laporan tidak tersedia.");
  const ordinary = generalReportRecords(records);
  const permitted =
    session.role === "Super Admin"
      ? ordinary
      : scope(ordinary, { ...session, permissions: ["bookings"] });
  const counts = Object.fromEntries(reportStatuses.map((status) => [status, 0])) as Record<
    DisplayStatus,
    number
  >;
  for (const record of permitted) {
    if (record.date < filter.from || record.date > filter.to) continue;
    const status = getVisitTiming(record, now).displayStatus;
    if (filter.status === "all" || filter.status === status) counts[status]++;
  }
  return {
    filter,
    counts,
    total: Object.values(counts).reduce((sum, count) => sum + count, 0),
    generatedAt: new Date(now).toISOString(),
    canExport: canExportSummary(session),
    scope:
      session.role === "Super Admin"
        ? "organization"
        : session.role === "PA"
          ? "assigned"
          : session.role === "Receptionist"
            ? "attendance"
            : "authorized",
  };
}
export function summaryCsv(summary: VisitSummary) {
  const rows = [
    ["Tanggal awal", "Tanggal akhir", "Cakupan", "Status", "Jumlah"],
    ...reportStatuses
      .filter((status) => summary.filter.status === "all" || summary.filter.status === status)
      .map((status) => [
        summary.filter.from,
        summary.filter.to,
        reportScopeLabels[summary.scope],
        reportLabels[status],
        String(summary.counts[status]),
      ]),
  ];
  return (
    "\uFEFF" +
    rows.map((row) => row.map((cell) => `"${cell.replaceAll('"', '""')}"`).join(",")).join("\r\n")
  );
}
