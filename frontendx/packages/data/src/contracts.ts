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
type VisitNotes = { visitorInstructions?: string; coordinationNote?: string };
/** `null` clears the room; an omitted `roomId` keeps the current one. */
type RoomChange = { roomId?: string | null };
/** A staff instruction that changes a visit (see CONTEXT.md, "Visit command"). */
export type VisitCommand =
  | ({ type: "approve"; duration?: number } & RoomChange & VisitNotes)
  | ({
      type: "reschedule";
      date: string;
      time: string;
      duration?: number;
      reason?: string;
    } & RoomChange &
      VisitNotes)
  | { type: "reject"; rejectionReason: string; internalRejectionNote?: string }
  | { type: "cancel"; reason: string; internalCancellationNote?: string }
  | { type: "attend"; direction: "in" | "out"; credential: string }
  | ({
      type: "update";
      visitKind?: "restricted";
      officerId: string;
      date?: string;
      time?: string;
      name?: string;
      organization?: string;
      serviceId?: string;
    } & RoomChange &
      VisitNotes);
/** The policy name a command is authorised under (`can(session, visit, name)`). */
export function commandAction(command: VisitCommand): BookingAction {
  if (command.type === "attend") return command.direction === "in" ? "checkin" : "checkout";
  return command.type;
}
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
  perform(
    visit: Pick<StaffBooking, "id" | "version">,
    command: VisitCommand,
    role: Role,
  ): Promise<StaffBooking>;
  resolve(credential: string, role: Role): Promise<StaffBooking>;
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
