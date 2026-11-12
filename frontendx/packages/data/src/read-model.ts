import { apiBase } from "./config.ts";
import type { StaffBooking } from "./domain.ts";
import { ApiError } from "./errors.ts";
import { createHttpClient } from "./http.ts";

export type ReadFilter = {
  view: "list" | "dashboard" | "reception";
  q?: string;
  status?: string;
  sort?: string;
  dir?: string;
  date?: string;
  vip?: string;
  page?: number;
  pageSize?: number;
  mode?: "checkin" | "checkout";
};
export type StaffReadModel = {
  items: StaffBooking[];
  total: number;
  totalScoped: number;
  page: number;
  pageSize: number;
  metrics?: { today: number; pending: number; arrivals: number; inside: number; completed: number };
};
const request = createHttpClient(apiBase);
export async function readStaffBookings(
  filter: ReadFilter,
  signal?: AbortSignal,
): Promise<StaffReadModel> {
  const { staffReadModelResponse } = await import("./wire.ts");
  const params = new URLSearchParams(
    Object.entries(filter)
      .filter(([, value]) => value !== undefined)
      .map(([key, value]) => [key, String(value)]),
  );
  const parsed = staffReadModelResponse.safeParse(
    await request(`/staff/bookings?${params}`, { signal, authenticated: true }),
  );
  if (
    !parsed.success ||
    (filter.view === "dashboard" && !parsed.data.metrics) ||
    (filter.pageSize && parsed.data.items.length > filter.pageSize)
  )
    throw new ApiError(502, "Hasil daftar tidak valid. Coba lagi.");
  return parsed.data;
}
export async function readStaffBooking(id: string, signal?: AbortSignal) {
  const { staffBookingResponse } = await import("./wire.ts");
  const parsed = staffBookingResponse.safeParse(
    await request(`/staff/bookings/${encodeURIComponent(id)}`, { signal, authenticated: true }),
  );
  if (!parsed.success) throw new ApiError(502, "Rincian kunjungan tidak valid.");
  return parsed.data;
}
