import { officerOrganization } from "./organization.ts";
import type { Catalogs } from "./contracts.ts";
import { getVisitTiming, visitTiming, type DisplayStatus, type VisitTiming } from "./attendance.ts";
import { publicVisitHistory, type PublicVisitEvent, type StoredVisitEvent } from "./history.ts";
export {
  canVisitorCancel,
  getVisitTiming,
  visitTiming,
  type DisplayStatus,
  type VisitTiming,
} from "./attendance.ts";
export { type PublicVisitAction, type PublicVisitEvent } from "./history.ts";
export const statuses = [
  "pending",
  "approved",
  "rejected",
  "cancelled",
  "checked_in",
  "checked_out",
] as const;
export type Status = (typeof statuses)[number];
const receptionistStatuses: readonly Status[] = ["approved", "checked_in", "checked_out"];
export function bookingStatusesFor(session: Session): readonly DisplayStatus[] {
  return [...(session.role === "Receptionist" ? receptionistStatuses : statuses), "missed"];
}
export const labels: Record<DisplayStatus, string> = {
  pending: "Pending",
  approved: "Disetujui",
  rejected: "Ditolak",
  cancelled: "Dibatalkan",
  checked_in: "Checked in",
  checked_out: "Checked out",
  missed: "Terlewat",
};
export type Role = "PA" | "Protocol Officer" | "Receptionist" | "Super Admin";
export const roles: Role[] = ["PA", "Protocol Officer", "Receptionist", "Super Admin"];
export interface Session {
  id?: string;
  mustChangePassword?: boolean;
  passwordMinLength?: number;
  name: string;
  role: Role;
  permissions: string[];
  assignedOfficerIds: string[];
  availableRoles?: Role[];
  roleRevision?: number;
  roleSelectionRequired?: false;
}
export interface PendingSession extends Omit<Session, "role" | "roleSelectionRequired"> {
  role: "";
  roleSelectionRequired: true;
  availableRoles: Role[];
}
export interface Booking {
  /** Missing on older standard records. Restricted access is independent of VIP. */
  visitKind?: "standard" | "restricted";
  id: string;
  code: string;
  name: string;
  organization: string;
  email: string;
  phone: string;
  officerId: string;
  host: string;
  vip: boolean;
  date: string;
  time: string;
  duration: number;
  room: string;
  purpose: string;
  visitors: string[];
  identityType: string;
  status: Status;
  version: number;
  rescheduled?: boolean;
  history: StoredVisitEvent[];
  ndName?: string;
  visitorInstructions?: string;
  attachmentAvailable?: boolean;
  rejectionReason?: string;
  cancellationReason?: string;
  timing?: VisitTiming;
}
export interface PublicBooking extends Omit<Booking, "history"> {
  history: PublicVisitEvent[];
}
export interface StoredBooking extends Booking {
  attachmentToken?: string;
  visitorPhotoToken?: string;
  visitorPhotoName?: string;
  coordinationNote?: string;
  internalCancellationNote?: string;
  serviceId?: string;
  internalRejectionNote?: string;
  createdBy?: { name: string; role: Role };
}
type PrivateFields =
  | "code"
  | "name"
  | "organization"
  | "email"
  | "phone"
  | "officerId"
  | "host"
  | "vip"
  | "purpose"
  | "visitors"
  | "identityType"
  | "history";
export interface StaffBooking
  extends Omit<Booking, PrivateFields>, Partial<Pick<Booking, PrivateFields>> {
  roomId?: string | null;
  visitorPhotoAvailable?: boolean;
  visitorCount?: number;
  visitorPhotoName?: string;
  internalCancellationNote?: string;
  coordinationNote?: string;
  reference?: string;
  restrictedView?: "full" | "assigned-pa" | "reception" | "arrival";
  serviceId?: string;
  serviceIdLast4?: string;
  internalRejectionNote?: string;
  createdBy?: { name: string; role: Role };
}
type ScopeRecord = Pick<StaffBooking, "visitKind" | "status" | "officerId" | "vip">;
export const officers: Catalogs["officers"] = [
  { id: "o1", name: "Kepala Otorita IKN", vip: true, active: true },
  { id: "o2", name: "Direktorat Data dan Kecerdasan Buatan", vip: false, active: true },
  { id: "o3", name: "Direktorat Pelayanan Dasar", vip: false, active: true },
].map((officer) => ({ ...officer, ...officerOrganization(officer) }));
export function isRestricted(b: Pick<Booking, "visitKind">) {
  return b.visitKind === "restricted";
}
export function bookingName(b: Pick<StaffBooking, "name" | "visitKind" | "reference">) {
  return b.name || (isRestricted(b) ? b.reference || "Kunjungan terbatas" : "Nama belum diisi");
}
export function partyLabel(b: Pick<StaffBooking, "visitors" | "visitorCount">) {
  const count = b.visitorCount ?? b.visitors?.length;
  return count ? `${count} orang` : "Rombongan belum diisi";
}
export const rooms = ["Ruang Rapat Nusantara", "Ruang Rapat Meranti", "Ruang Rapat Mahakam"];
/** Preview room catalogue with the same `{ id, name }` identity the backend uses. */
export const roomOptions = rooms.map((name, index) => ({ id: `room-${index + 1}`, name }));
export function roomIdOf(name: string): string | null {
  return roomOptions.find((room) => room.name === name)?.id ?? null;
}
export function today() {
  return new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Makassar",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date());
}
export function dateLabel(d: string, language: "id" | "en" = "id") {
  return new Intl.DateTimeFormat(language === "en" ? "en-US" : "id-ID", {
    dateStyle: "long",
    timeZone: "Asia/Makassar",
  }).format(new Date(`${d}T12:00:00+08:00`));
}
export function scope<T extends ScopeRecord>(list: T[], s: Session): T[] {
  if (!s.permissions.includes("bookings")) return [];
  if (s.role === "Super Admin") return list.filter(isRestricted);
  if (s.role === "Receptionist") return list.filter((b) => receptionistStatuses.includes(b.status));
  if (s.role === "PA")
    return list.filter(
      (b) =>
        !!b.officerId && s.assignedOfficerIds.includes(b.officerId) && (isRestricted(b) || !b.vip),
    );
  return s.role === "Protocol Officer" ? list : [];
}
export function canReadInternalRejection(s: Session, b: ScopeRecord) {
  return (
    s.permissions.includes("reject") &&
    (s.role === "Protocol Officer"
      ? isRestricted(b) || b.vip
      : s.role === "PA" &&
        !isRestricted(b) &&
        !b.vip &&
        !!b.officerId &&
        s.assignedOfficerIds.includes(b.officerId))
  );
}
export function canReadInternalCancellation(s: Session, b: ScopeRecord) {
  return (
    scope([b], s).length > 0 &&
    (isRestricted(b)
      ? s.role === "Protocol Officer" || s.role === "Super Admin"
      : s.role === "PA" || s.role === "Protocol Officer")
  );
}
// Explicit public allowlist: staff-only or future fields cannot leak to visitors.
export function toVisitorBooking(b: Booking, now = Date.now()): PublicBooking {
  if (isRestricted(b)) throw new Error("Kunjungan tidak tersedia.");
  return {
    ...bookingProjection(b),
    history: publicVisitHistory(b.history),
    timing: getVisitTiming(b, now),
  };
}
function bookingProjection(b: Booking): Booking {
  return {
    id: b.id,
    code: b.code,
    name: b.name,
    organization: b.organization,
    email: b.email,
    phone: b.phone,
    officerId: b.officerId,
    host: b.host,
    vip: b.vip,
    date: b.date,
    time: b.time,
    duration: b.duration,
    room: b.room,
    purpose: b.purpose,
    visitors: b.visitors,
    identityType: b.identityType,
    status: b.status,
    version: b.version,
    rescheduled: b.rescheduled,
    history: b.history.map(({ at, message }) => ({ at, message })),
    ndName: b.ndName,
    visitorInstructions: b.visitorInstructions,
    rejectionReason: b.rejectionReason,
    cancellationReason: b.cancellationReason,
  };
}
export function canCreateAppointment(
  s: Session,
  officerId?: string,
  kind: "standard" | "restricted" = "standard",
  available = officers,
) {
  if (!s.permissions.includes("create") || !["PA", "Protocol Officer"].includes(s.role))
    return false;
  if (kind === "restricted")
    return (
      s.role === "Protocol Officer" &&
      (!officerId || available.some((o) => o.id === officerId && o.active))
    );
  if (officerId === undefined) return true;
  const officer = available.find((o) => o.id === officerId);
  return (
    !!officer?.active &&
    (s.role === "Protocol Officer"
      ? officer.vip
      : !officer.vip && s.assignedOfficerIds.includes(officer.id))
  );
}
export function toStaffBooking(
  b: StoredBooking,
  s: Session,
  { arrival = false, now = Date.now() }: { arrival?: boolean; now?: number } = {},
): StaffBooking {
  if (!scope([b], s).length) throw new Error("Kunjungan tidak tersedia.");
  if (isRestricted(b)) {
    const common: StaffBooking = {
      id: b.id,
      reference: `TRB-${b.id.startsWith("demo-") ? b.id : b.id.slice(0, 8)}`,
      visitKind: "restricted",
      date: b.date,
      time: b.time,
      duration: b.duration,
      room: b.room,
      status: b.status,
      version: b.version,
      rescheduled: b.rescheduled,
      timing: getVisitTiming(b, now),
    };
    if (s.role === "Receptionist" && !arrival) return { ...common, restrictedView: "reception" };
    const identity = {
      ...(b.name ? { name: b.name } : {}),
      ...(b.officerId ? { officerId: b.officerId, host: b.host } : {}),
    };
    if (s.role === "Receptionist" || s.role === "PA")
      return {
        ...common,
        ...identity,
        restrictedView: s.role === "PA" ? "assigned-pa" : "arrival",
        ...(s.role === "PA" && b.organization ? { organization: b.organization } : {}),
        ...(b.serviceId ? { serviceIdLast4: b.serviceId.slice(-4) } : {}),
      };
    return {
      ...common,
      ...identity,
      restrictedView: "full",
      roomId: roomIdOf(b.room),
      visitorInstructions: b.visitorInstructions,
      coordinationNote: b.coordinationNote,
      code: b.code,
      ...(b.organization ? { organization: b.organization } : {}),
      ...(b.serviceId ? { serviceId: b.serviceId } : {}),
      history: b.history.map(({ at, message }) => ({ at, message })),
      createdBy: b.createdBy,
      rejectionReason: b.rejectionReason,
      cancellationReason: b.cancellationReason,
      ...(canReadInternalCancellation(s, b) && b.internalCancellationNote
        ? { internalCancellationNote: b.internalCancellationNote }
        : {}),
      ...(canReadInternalRejection(s, b) && b.internalRejectionNote
        ? { internalRejectionNote: b.internalRejectionNote }
        : {}),
    };
  }
  if (s.role === "Receptionist" && !arrival)
    return {
      id: b.id,
      visitKind: b.visitKind,
      name: b.name,
      organization: b.organization,
      host: b.host,
      vip: b.vip,
      visitorCount: b.visitors.length,
      date: b.date,
      time: b.time,
      duration: b.duration,
      room: b.room,
      status: b.status,
      version: b.version,
      rescheduled: b.rescheduled,
      timing: getVisitTiming(b, now),
    };
  const visible: StaffBooking = {
    ...bookingProjection(b),
    roomId: roomIdOf(b.room),
    ...(b.attachmentAvailable ? { attachmentAvailable: true } : {}),
    ...(b.visitorPhotoToken
      ? { visitorPhotoAvailable: true, visitorPhotoName: b.visitorPhotoName }
      : {}),
    visitKind: b.visitKind,
    createdBy: b.createdBy,
    timing: getVisitTiming(b, now),
    ...(["PA", "Protocol Officer"].includes(s.role)
      ? { coordinationNote: b.coordinationNote }
      : {}),
    ...(canReadInternalCancellation(s, b)
      ? { internalCancellationNote: b.internalCancellationNote }
      : {}),
  };
  return canReadInternalRejection(s, b)
    ? { ...visible, internalRejectionNote: b.internalRejectionNote }
    : visible;
}
const missedRescheduleWindowMs = 7 * 24 * 60 * 60 * 1000;

export function can(
  s: Session,
  b: StaffBooking,
  action: "approve" | "reject" | "reschedule" | "checkin" | "checkout" | "cancel" | "update",
  now?: number,
) {
  const permission = action === "update" ? "create" : action;
  if (!s.permissions.includes(permission) || !scope([b], s).length) return false;
  if (action === "update")
    return (
      s.role === "Protocol Officer" && isRestricted(b) && ["pending", "approved"].includes(b.status)
    );
  const timing = now === undefined ? visitTiming(b) : getVisitTiming(b, now);
  if (action === "checkin") return s.role === "Receptionist" && timing.canCheckIn;
  if (action === "checkout") return s.role === "Receptionist" && timing.canCheckOut;
  const owns =
    s.role === "Protocol Officer"
      ? isRestricted(b) || b.vip
      : s.role === "PA" &&
        !isRestricted(b) &&
        !b.vip &&
        !!b.officerId &&
        s.assignedOfficerIds.includes(b.officerId);
  if (action === "cancel") return owns && ["pending", "approved"].includes(b.status);
  return (
    owns &&
    (action === "reschedule"
      ? b.status === "approved" &&
        (timing.displayStatus !== "missed" ||
          timing.serverTime <= timing.endsAt + missedRescheduleWindowMs)
      : b.status === "pending")
  );
}
export function sessionFor(role: Role): Session {
  return {
    name:
      role === "PA"
        ? "Aulia Rahma"
        : role === "Receptionist"
          ? "Dimas Pratama"
          : role === "Protocol Officer"
            ? "Nadia Putri"
            : "Admin Nusantara",
    role,
    assignedOfficerIds: role === "PA" ? ["o2"] : [],
    permissions:
      role === "PA" || role === "Protocol Officer"
        ? [
            "bookings",
            "create",
            "approve",
            "reject",
            "reschedule",
            "cancel",
            "reports",
            "reportExport",
          ]
        : role === "Receptionist"
          ? ["bookings", "checkin", "checkout"]
          : ["bookings", "people", "roles", "integrations", "reports", "reportExport", "audit"],
  };
}
export type SortKey = "priority" | "name" | "organization" | "host" | "time" | "status";
export function selectBookings<T extends StaffBooking>(
  list: T[],
  q: string,
  status: string,
  sort: SortKey,
  desc = false,
) {
  const needle = q.trim().toLocaleLowerCase("id");
  const rank = (b: StaffBooking) =>
    b.name?.toLowerCase().startsWith(needle)
      ? 0
      : b.name?.toLowerCase().includes(needle)
        ? 1
        : b.visitors?.some((n) => n.toLowerCase().includes(needle))
          ? 2
          : b.organization?.toLowerCase().includes(needle)
            ? 3
            : 4;
  const filtered = list.filter(
    (b) =>
      (status === "all" ||
        (status === "missed"
          ? visitTiming(b).displayStatus === "missed"
          : b.status === status && visitTiming(b).displayStatus !== "missed")) &&
      (!needle ||
        [b.name, ...(b.visitors || []), b.organization, b.host, b.code, b.reference].some((v) =>
          v?.toLowerCase().includes(needle),
        )),
  );
  const priority = (b: StaffBooking) => {
    if (b.status === "pending") return 0;
    if (visitTiming(b).displayStatus === "approved" || b.status === "checked_in") return 1;
    return 2;
  };
  return filtered.sort((a, b) => {
    if (needle && rank(a) !== rank(b)) return rank(a) - rank(b);
    let v = 0;
    if (sort === "priority") {
      const group = priority(a) - priority(b);
      if (group) return group;
      const time = (a.date + a.time).localeCompare(b.date + b.time);
      return (priority(a) === 2 ? -time : time) || a.id.localeCompare(b.id);
    } else if (sort === "time") v = (a.date + a.time).localeCompare(b.date + b.time);
    else v = String(a[sort]).localeCompare(String(b[sort]), "id");
    return (
      (v || (a.date + a.time).localeCompare(b.date + b.time) || a.id.localeCompare(b.id)) *
      (desc ? -1 : 1)
    );
  });
}
export function generalReportRecords<T extends Pick<Booking, "visitKind">>(records: T[]): T[] {
  return records.filter((record) => !isRestricted(record));
}
export const notaDinasType = "application/pdf";
export const visitorPhotoTypes: readonly string[] = ["image/jpeg", "image/png"];

export function validateND(file: { type: string; size: number; name?: string } | null) {
  if (!file) return null;
  if (file.name !== undefined && (!file.name.trim() || file.name.trim().length > 255))
    return "Nama dokumen maksimal 255 karakter.";
  if (file.type !== notaDinasType) return "Gunakan berkas PDF.";
  if (file.size > 5 * 1024 * 1024 || file.size === 0)
    return "Ukuran dokumen maksimal 5 MB dan tidak boleh kosong.";
  return null;
}
export function validateVisitorPhoto(file: { type: string; size: number; name?: string } | null) {
  if (!file) return null;
  if (file.name !== undefined && (!file.name.trim() || file.name.trim().length > 255))
    return "Nama foto maksimal 255 karakter.";
  if (!visitorPhotoTypes.includes(file.type)) return "Gunakan foto JPEG atau PNG.";
  if (file.size > 5 * 1024 * 1024 || file.size === 0)
    return "Ukuran foto maksimal 5 MB dan tidak boleh kosong.";
  return null;
}
const attachmentSignatures: Readonly<Record<string, readonly number[]>> = {
  [notaDinasType]: [0x25, 0x50, 0x44, 0x46, 0x2d],
  "image/jpeg": [0xff, 0xd8, 0xff],
  "image/png": [0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a],
};

export function matchesAttachmentSignature(type: string, bytes: Uint8Array): boolean {
  const signature = attachmentSignatures[type];
  return Array.isArray(signature) && signature.every((byte, index) => bytes[index] === byte);
}
export function validateReschedule(
  date: string,
  time: string,
  duration: number,
  room: string,
  reason: string,
  restricted = false,
  availableRooms: string[] | null = rooms,
  now = Date.now(),
) {
  const stamp = new Date(`${date}T${time}:00+08:00`).getTime();
  if (!date || !time || !Number.isFinite(stamp) || stamp <= now)
    return "Pilih jadwal baru di masa depan (WITA).";
  if (!/^(?:08|09|1[0-5]):[0-5]\d$|^16:00$/.test(time))
    return "Pilih waktu mulai antara 08.00–16.00 WITA.";
  if (!Number.isInteger(duration) || duration < 15 || duration > 240)
    return "Durasi harus 15–240 menit.";
  if (availableRooms && (!restricted || room) && !availableRooms.includes(room))
    return "Pilih ruangan.";
  if ((!restricted || reason) && (typeof reason !== "string" || reason.trim().length < 5))
    return "Tuliskan alasan minimal 5 karakter.";
  return null;
}
