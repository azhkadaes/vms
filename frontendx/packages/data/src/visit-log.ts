import { z } from "zod";
import { getVisitTiming, type Session, type StoredBooking } from "./domain.ts";
import { reportStatuses, reportStatusFilter } from "./reports.ts";
import { matchesFilter, multipleFilter } from "./report-filter.ts";

export const visitLogSorts = [
  "date",
  "name",
  "organization",
  "host",
  "status",
  "kind",
  "visitors",
] as const;
export const visitKindLabels = { standard: "Reguler", vip: "VIP", restricted: "Terbatas" };
export const visitLogFilterSchema = z
  .object({
    from: z.union([z.literal(""), z.iso.date()]).default(""),
    to: z.union([z.literal(""), z.iso.date()]).default(""),
    q: z.string().trim().max(200).default(""),
    status: reportStatusFilter.default("all"),
    kind: multipleFilter(
      z.enum(["all", "standard", "vip", "restricted"]),
      z.enum(["standard", "vip", "restricted"]),
    ).default("all"),
    officer: multipleFilter(z.string().max(200), z.string().min(1).max(200)).default(""),
    room: multipleFilter(z.string().max(200), z.string().min(1).max(200)).default(""),
    sort: z.enum(visitLogSorts).default("date"),
    dir: z.enum(["asc", "desc"]).default("desc"),
  })
  .strict()
  .refine((f) => !f.from || !f.to || f.from <= f.to, {
    path: ["to"],
    message: "Tanggal akhir harus sama atau sesudah tanggal awal.",
  });
export type VisitLogFilter = z.infer<typeof visitLogFilterSchema>;
export const defaultVisitLogFilter = (): VisitLogFilter => visitLogFilterSchema.parse({});
export const visitLogRowSchema = z
  .object({
    id: z.string().min(1),
    reference: z.string(),
    name: z.string(),
    organization: z.string(),
    officerId: z.string(),
    host: z.string(),
    date: z.iso.date(),
    time: z.string(),
    duration: z.number().int().nonnegative(),
    room: z.string(),
    purpose: z.string(),
    kind: z.enum(["standard", "vip", "restricted"]),
    visitors: z.number().int().nonnegative().nullable(),
    status: z.enum(reportStatuses),
    checkedInAt: z.string().datetime().nullable(),
    checkedOutAt: z.string().datetime().nullable(),
  })
  .strict();
export type VisitLogRow = z.infer<typeof visitLogRowSchema>;
export const visitLogSummarySchema = z
  .object({
    total: z.number().int().nonnegative(),
    visitors: z.number().int().nonnegative(),
    inside: z.number().int().nonnegative(),
    completed: z.number().int().nonnegative(),
    pending: z.number().int().nonnegative(),
  })
  .strict();
export const visitLogSnapshotSchema = z
  .object({
    filter: visitLogFilterSchema,
    items: z.array(visitLogRowSchema),
    summary: visitLogSummarySchema,
    generatedAt: z.string().datetime(),
    revision: z.string().regex(/^[a-f0-9]{64}$/),
  })
  .strict();
export type VisitLogSnapshot = z.infer<typeof visitLogSnapshotSchema>;
export const visitLogResponseSchema = visitLogSnapshotSchema
  .extend({
    page: z.number().int().positive(),
    pageSize: z.number().int().positive(),
    total: z.number().int().nonnegative(),
    canExport: z.boolean(),
    officers: z.array(z.object({ id: z.string(), name: z.string() }).strict()),
    rooms: z.array(z.string()),
  })
  .superRefine((data, ctx) => {
    if (
      data.total !== data.summary.total ||
      data.items.length > data.pageSize ||
      data.items.length > data.total
    )
      ctx.addIssue({ code: "custom", message: "Jumlah log tidak sesuai." });
  });
export type VisitLogResponse = z.infer<typeof visitLogResponseSchema>;
export function canReadVisitLog(session: Session) {
  return session.role === "Super Admin" && session.permissions.includes("reports");
}

// Deliberately separate from operational booking scope and from aggregate-only reports.
// This projection never includes service IDs, restricted attendance credentials, contact data or private notes.
export function projectVisitLog(records: StoredBooking[], now: number): VisitLogRow[] {
  return records.map((b) => ({
    id: b.id,
    reference:
      b.visitKind === "restricted"
        ? `TRB-${b.id.startsWith("demo-") ? b.id : b.id.slice(0, 8)}`
        : b.code,
    name: b.name || "Kunjungan terbatas",
    organization: b.organization || "",
    officerId: b.officerId || "",
    host: b.host || "",
    date: b.date,
    time: b.time,
    duration: b.duration,
    room: b.room,
    purpose: b.visitKind === "restricted" ? "" : b.purpose,
    kind: b.visitKind === "restricted" ? "restricted" : b.vip ? "vip" : "standard",
    visitors: b.visitKind === "restricted" ? null : b.visitors.length,
    status: getVisitTiming(b, now).displayStatus,
    checkedInAt: b.history.find((e) => e.action === "checked_in")?.at ?? null,
    checkedOutAt: b.history.find((e) => e.action === "checked_out")?.at ?? null,
  }));
}
export function selectVisitLog(records: VisitLogRow[], filter: VisitLogFilter) {
  const needle = filter.q.toLocaleLowerCase("id");
  return records
    .filter(
      (b) =>
        (!filter.from || b.date >= filter.from) &&
        (!filter.to || b.date <= filter.to) &&
        matchesFilter(b.status, filter.status, "all") &&
        matchesFilter(b.kind, filter.kind, "all") &&
        matchesFilter(b.officerId, filter.officer, "") &&
        matchesFilter(b.room, filter.room, "") &&
        (!needle ||
          [b.reference, b.name, b.organization, b.host, b.room, b.purpose].some((v) =>
            v.toLocaleLowerCase("id").includes(needle),
          )),
    )
    .sort((a, b) => {
      const key = filter.sort;
      const value =
        key === "date"
          ? (a.date + a.time).localeCompare(b.date + b.time)
          : key === "visitors"
            ? (a.visitors ?? -1) - (b.visitors ?? -1)
            : String(a[key]).localeCompare(String(b[key]), "id", { numeric: true });
      return (value || a.id.localeCompare(b.id)) * (filter.dir === "desc" ? -1 : 1);
    });
}
export function summarizeVisitLog(items: VisitLogRow[]) {
  return {
    total: items.length,
    visitors: items.reduce((sum, b) => sum + (b.visitors ?? 0), 0),
    inside: items.filter((b) => b.status === "checked_in").length,
    completed: items.filter((b) => b.status === "checked_out").length,
    pending: items.filter((b) => b.status === "pending").length,
  };
}
