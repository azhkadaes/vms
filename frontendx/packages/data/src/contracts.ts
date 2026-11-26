import type { PublicBooking, Role, StaffBooking } from "./domain.ts";
import type { SummaryExport, SummaryFilter, VisitSummary } from "./reports.ts";
import type { VisitLogFilter, VisitLogResponse, VisitLogSnapshot } from "./visit-log.ts";
import type { SubmissionResolution } from "./submission.ts";
import type { PublicBookingInput, StaffAppointmentInput } from "./validation.ts";

export interface Catalogs {
  officers: {
    id: string;
    name: string;
    vip: boolean;
    active: boolean;
    jobTitle?: string;
    directorate?: string;
    deputy?: string;
  }[];
  rooms: string[];
  roomOptions?: { id: string; name: string }[];
  privacyNotice?: { version: string; id: string; en: string };
}
export interface Admission {
  payload: string;
  imagePng?: string;
  expiresAt: string;
  serverTime: number;
}
export type BookingAction =
  "approve" | "reject" | "reschedule" | "checkin" | "checkout" | "update" | "cancel";
export interface Repository {
  catalogs(role?: Role, signal?: AbortSignal): Promise<Catalogs>;
  list(role: Role, signal?: AbortSignal): Promise<StaffBooking[]>;
  create(input: PublicBookingInput, idempotencyKey: string): Promise<PublicBooking>;
  resolveSubmission(key: string, email: string): Promise<SubmissionResolution>;
  summary(filter: SummaryFilter, role: Role, signal?: AbortSignal): Promise<VisitSummary>;
  exportSummary(filter: SummaryFilter, revision: string, role: Role): Promise<SummaryExport>;
  visitLog(
    filter: VisitLogFilter,
    page: number,
    pageSize: number,
    role: Role,
    signal?: AbortSignal,
  ): Promise<VisitLogResponse>;
  exportVisitLog(filter: VisitLogFilter, revision: string, role: Role): Promise<VisitLogSnapshot>;
  createStaff(
    input: StaffAppointmentInput,
    idempotencyKey: string,
    role: Role,
  ): Promise<StaffBooking>;
  action(
    id: string,
    action: BookingAction,
    input: Record<string, unknown>,
    role: Role,
  ): Promise<StaffBooking>;
  resolve(credential: string, role: Role, visitId?: string): Promise<StaffBooking>;
  lookup(code: string, email: string): Promise<PublicBooking>;
  cancel(
    id: string,
    input: { code: string; email: string; version: number },
  ): Promise<PublicBooking>;
  admission(
    booking: Pick<PublicBooking, "id" | "code" | "email" | "version">,
    signal?: AbortSignal,
  ): Promise<Admission>;
}
