import { apiBase, preview } from "./config.ts";
import type { Repository } from "./contracts.ts";
import { officers, roomOptions, rooms } from "./domain.ts";
import { ApiError } from "./errors.ts";
import { createHttpClient } from "./http.ts";
import { reportFilterParams } from "./report-filter.ts";

export function createRepository({
  development = false,
  base = "/v1",
  fetcher = fetch,
}: { development?: boolean; base?: string; fetcher?: typeof fetch } = {}): Repository {
  const request = createHttpClient(development ? "/__preview" : base, fetcher);
  // The preview server takes production paths (ADR-0001); it only needs the impersonated role.
  const staff = (path: string, role: string) =>
    development
      ? `/staff${path}${path.includes("?") ? "&" : "?"}role=${encodeURIComponent(role)}`
      : `/staff${path}`;
  const publicPath = (path: string) => `/public${path}`;
  async function parse<T>(
    value: unknown,
    schema: { safeParse: (value: unknown) => { success: true; data: T } | { success: false } },
  ): Promise<T> {
    const parsed = schema.safeParse(value);
    if (!parsed.success) throw new ApiError(502, "Respons layanan tidak valid. Coba lagi.");
    return parsed.data;
  }
  return {
    visitLog: async (filter, page, pageSize, role, signal) => {
      const { visitLogResponseSchema } = await import("./visit-log.ts");
      const params = reportFilterParams({
        ...filter,
        page: String(page),
        pageSize: String(pageSize),
      });
      const result = await parse(
        await request(staff(`/reports/visits?${params}`, role), {
          signal,
          authenticated: true,
        }),
        visitLogResponseSchema,
      );
      if (JSON.stringify(result.filter) !== JSON.stringify(filter))
        throw new ApiError(502, "Filter log tidak sesuai. Coba lagi.");
      return result;
    },
    exportVisitLog: async (filter, revision, role) => {
      const { visitLogSnapshotSchema } = await import("./visit-log.ts");
      const result = await parse(
        await request(
          staff(`/reports/visits/export?${reportFilterParams({ ...filter, revision })}`, role),
          {
            authenticated: true,
          },
        ),
        visitLogSnapshotSchema,
      );
      if (
        result.revision !== revision ||
        JSON.stringify(result.filter) !== JSON.stringify(filter) ||
        result.items.length !== result.summary.total
      )
        throw new ApiError(409, "Log telah berubah. Perbarui sebelum mengekspor.");
      return result;
    },
    catalogs: async (role, signal) => {
      if (development) return { officers, rooms, roomOptions };
      const { catalogsResponse } = await import("./wire.ts");
      return parse(
        await request(role ? "/staff/catalogs" : "/public/catalogs", {
          signal,
          authenticated: !!role,
        }),
        catalogsResponse,
      );
    },
    list: async (role, signal) => {
      const { staffBookingResponse } = await import("./wire.ts");
      const result = await request(staff("/bookings", role), { signal, authenticated: true });
      if (!Array.isArray(result))
        throw new ApiError(502, "Respons layanan tidak valid. Coba lagi.");
      return Promise.all(result.map((value) => parse(value, staffBookingResponse)));
    },
    create: async (body, key) => {
      const { publicBookingResponse } = await import("./wire.ts");
      return parse(
        await request(publicPath("/bookings"), { body: { ...body, idempotencyKey: key } }),
        publicBookingResponse,
      );
    },
    resolveSubmission: async (idempotencyKey, email) => {
      const { resolutionResponse } = await import("./wire.ts");
      return parse(
        await request(publicPath("/submissions/resolve"), { body: { idempotencyKey, email } }),
        resolutionResponse,
      );
    },
    createStaff: async (input, key, role) => {
      const { staffBookingResponse } = await import("./wire.ts");
      return parse(
        await request(staff("/bookings", role), {
          body: { ...input, idempotencyKey: key },
          authenticated: true,
        }),
        staffBookingResponse,
      );
    },
    perform: async (visit, command, role) => {
      const { staffBookingResponse } = await import("./wire.ts");
      const { type, ...fields } = command;
      const [operation, body] =
        command.type === "attend"
          ? [
              "attendance",
              {
                action: command.direction === "in" ? "check_in" : "check_out",
                credential: command.credential,
              },
            ]
          : [type, fields];
      return parse(
        await request(staff(`/bookings/${encodeURIComponent(visit.id)}/${operation}`, role), {
          body: { ...body, version: visit.version },
          authenticated: true,
        }),
        staffBookingResponse,
      );
    },
    resolve: async (credential, role) => {
      const { staffBookingResponse } = await import("./wire.ts");
      return parse(
        await request(staff("/resolve", role), { body: { credential }, authenticated: true }),
        staffBookingResponse,
      );
    },
    lookup: async (code, email) => {
      const { publicBookingResponse } = await import("./wire.ts");
      return parse(
        await request(publicPath("/lookup"), { body: { code: code.trim(), email: email.trim() } }),
        publicBookingResponse,
      );
    },
    cancel: async (id, input) => {
      const { publicBookingResponse } = await import("./wire.ts");
      return parse(
        await request(publicPath(`/bookings/${encodeURIComponent(id)}/cancel`), { body: input }),
        publicBookingResponse,
      );
    },
    admission: async (booking, signal) => {
      if (development)
        return {
          payload: `VMS-DEMO:${booking.code}`,
          expiresAt: new Date(Date.now() + 60_000).toISOString(),
          serverTime: Date.now(),
        };
      const { admissionResponse } = await import("./wire.ts");
      return parse(
        await request(publicPath(`/bookings/${encodeURIComponent(booking.id)}/admission`), {
          body: { code: booking.code, email: booking.email, version: booking.version },
          signal,
        }),
        admissionResponse,
      );
    },
    summary: async (filter, role, signal) => {
      const { summaryResponseSchema } = await import("./reports.ts");
      const result = await parse(
        await request(staff(`/reports/summary?${reportFilterParams(filter)}`, role), {
          signal,
          authenticated: true,
        }),
        summaryResponseSchema,
      );
      if (
        result.filter.from !== filter.from ||
        result.filter.to !== filter.to ||
        JSON.stringify(result.filter.status) !== JSON.stringify(filter.status)
      )
        throw new ApiError(502, "Filter ringkasan tidak sesuai. Coba lagi.");
      return result;
    },
    exportSummary: async (filter, revision, role) => {
      const { summaryExportSchema } = await import("./reports.ts");
      return parse(
        await request(
          staff(`/reports/export?${reportFilterParams({ ...filter, revision })}`, role),
          { authenticated: true },
        ),
        summaryExportSchema,
      );
    },
  };
}
export const repository = createRepository({ development: preview, base: apiBase });
