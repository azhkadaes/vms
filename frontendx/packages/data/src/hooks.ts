import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { preview } from "./config.ts";
import type { BookingAction } from "./contracts.ts";
import { can, officers, rooms, scope, type StaffBooking } from "./domain.ts";
import { ApiError } from "./errors.ts";
import {
  readStaffBooking,
  readStaffBookings,
  type ReadFilter,
  type StaffReadModel,
} from "./read-model.ts";
import type { SummaryFilter } from "./reports.ts";
import { canReadVisitLog, type VisitLogFilter } from "./visit-log.ts";
import { repository } from "./repositories.ts";
import { sessionKey } from "./session-contract.ts";
import { useSession } from "./session.tsx";
import type { StaffAppointmentInput } from "./validation.ts";

export function useCatalogs() {
  const { session } = useSession();
  return useQuery({
    queryKey: ["catalogs", sessionKey(session)],
    queryFn: ({ signal }) => repository.catalogs(session?.role, signal),
    staleTime: 60000,
    initialData: preview ? { officers, rooms } : undefined,
  });
}
export function useBookings({
  refresh,
  filter,
}: { refresh?: "staff" | "reception"; filter?: ReadFilter } = {}) {
  const { session } = useSession();
  const query = useQuery({
    queryKey: ["bookings", sessionKey(session), ...(preview ? [] : [filter])],
    queryFn: async ({ signal }): Promise<StaffReadModel> => {
      if (!preview) return readStaffBookings(filter!, signal);
      const items = await repository.list(session!.role, signal);
      return {
        items,
        total: items.length,
        totalScoped: items.length,
        page: 1,
        pageSize: items.length || 1,
      };
    },
    enabled:
      !!session?.permissions.includes("bookings") &&
      (preview || !!filter) &&
      !(filter?.view === "dashboard" && session?.role === "Super Admin"),
    refetchInterval: refresh ? (refresh === "reception" ? 15000 : 30000) : false,
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: !preview || refresh ? "always" : false,
    refetchOnReconnect: "always",
  });
  return {
    ...query,
    data: query.data && session ? scope(query.data.items, session) : undefined,
    total: query.data?.total,
    totalScoped: query.data?.totalScoped,
    metrics: query.data?.metrics,
  };
}
export function useBookingDetail(id: string | null) {
  const { session } = useSession();
  return useQuery({
    queryKey: ["bookings", sessionKey(session), "detail", id],
    queryFn: async ({ signal }) => {
      const booking = await readStaffBooking(id!, signal);
      if (!session || !scope([booking], session).length)
        throw new ApiError(404, "Kunjungan tidak tersedia.");
      return booking;
    },
    enabled: !preview && !!id && !!session && session.role !== "Receptionist",
    refetchOnWindowFocus: "always",
  });
}
export function useVisitSummary(filter: SummaryFilter | null) {
  const { session } = useSession();
  return useQuery({
    queryKey: ["visit-summary", sessionKey(session), filter],
    queryFn: ({ signal }) => repository.summary(filter!, session!.role, signal),
    enabled: !!filter && !!session?.permissions.includes("reports"),
    refetchOnWindowFocus: "always",
    retry: (failures, error) =>
      failures < 1 && !(error instanceof ApiError && [401, 403, 422].includes(error.status)),
  });
}
export function useVisitLog(filter: VisitLogFilter | null, page: number, pageSize: number) {
  const { session } = useSession();
  return useQuery({
    queryKey: ["visit-log", sessionKey(session), filter, page, pageSize],
    queryFn: ({ signal }) => repository.visitLog(filter!, page, pageSize, session!.role, signal),
    enabled: !!filter && !!session && canReadVisitLog(session),
    refetchOnWindowFocus: "always",
    retry: (failures, error) =>
      failures < 1 && !(error instanceof ApiError && [401, 403, 422].includes(error.status)),
  });
}
export function useCreateAppointment() {
  const { session } = useSession();
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({
      input,
      idempotencyKey,
    }: {
      input: StaffAppointmentInput;
      idempotencyKey: string;
    }) => {
      if (!session) throw new ApiError(401, "Sesi berakhir. Silakan masuk kembali.");
      return repository.createStaff(input, idempotencyKey, session.role);
    },
    onSuccess: () => client.invalidateQueries({ queryKey: ["bookings"] }),
    onError: (error) => {
      if (error instanceof ApiError && error.status === 409)
        void client.invalidateQueries({ queryKey: ["bookings"] });
    },
  });
}
export function useBookingAction() {
  const { session } = useSession();
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({
      booking,
      action,
      values = {},
    }: {
      booking: StaffBooking;
      action: BookingAction;
      values?: Record<string, unknown>;
    }) => {
      if (!session) throw new ApiError(401, "Sesi berakhir. Silakan masuk kembali.");
      if (!can(session, booking, action))
        throw new ApiError(403, "Tindakan tidak diizinkan untuk kunjungan ini.");
      return repository.action(
        booking.id,
        action,
        { ...values, version: booking.version },
        session.role,
      );
    },
    onSuccess: () => client.invalidateQueries({ queryKey: ["bookings"] }),
    onError: (error) => {
      if (error instanceof ApiError && error.status === 409)
        void client.invalidateQueries({ queryKey: ["bookings"] });
    },
  });
}
