import { z } from "zod";
import { officers, today } from "./domain.ts";
export const emailSchema = z.string().trim().email("Periksa nama email dan domain Anda.");
export const rejectionSchema = z.object({
  rejectionReason: z
    .string()
    .trim()
    .min(5, "Tuliskan pesan untuk pengaju minimal 5 karakter.")
    .max(1000, "Pesan untuk pengaju maksimal 1000 karakter."),
  internalRejectionNote: z
    .string()
    .trim()
    .max(2000, "Catatan internal maksimal 2000 karakter.")
    .optional()
    .default(""),
});
export type RejectionInput = z.input<typeof rejectionSchema>;
export const staffCancellationSchema = z.object({
  reason: z
    .string()
    .trim()
    .min(5, "Tuliskan alasan pembatalan minimal 5 karakter.")
    .max(1000, "Alasan pembatalan maksimal 1000 karakter."),
  internalCancellationNote: z
    .string()
    .trim()
    .max(2000, "Catatan internal maksimal 2000 karakter.")
    .optional()
    .default(""),
});
const calendarDate = z
  .string()
  .refine(
    (d) =>
      /^\d{4}-\d{2}-\d{2}$/.test(d) &&
      Number.isFinite(Date.parse(`${d}T12:00:00+08:00`)) &&
      new Date(`${d}T12:00:00+08:00`).toISOString().slice(0, 10) === d,
    "Pilih tanggal yang valid, hari ini atau sesudahnya.",
  );
const officerId = z.string().trim().min(1, "Pilih tujuan kunjungan.");
type Officer = { id: string; active: boolean };
const activeOfficer = (available: Officer[]) => (data: { officerId: string }) =>
  available.some((officer) => officer.id === data.officerId && officer.active);
const officerError = { path: ["officerId"], message: "Pilih pejabat aktif yang tersedia." };
const privacyAck = z
  .boolean()
  .refine((value) => value, "Baca dan konfirmasi pemberitahuan privasi.");
const visitFields = {
  officerId,
  date: calendarDate.refine(
    (d) => d >= today(),
    "Pilih tanggal yang valid, hari ini atau sesudahnya.",
  ),
  time: z
    .string()
    .regex(/^(?:08|09|1[0-5]):[0-5]\d$|^16:00$/, "Pilih waktu mulai antara 08.00–16.00 WITA."),
  purpose: z.string().trim().min(5, "Jelaskan keperluan minimal 5 karakter.").max(1000),
};
function futureStart(d: { date: string; time: string }) {
  return new Date(`${d.date}T${d.time}:00+08:00`).getTime() > Date.now();
}
const scheduleError = {
  path: ["time"],
  message: "Waktu kunjungan harus di masa depan (WITA).",
};
const visitBase = z.object(visitFields).refine(futureStart, scheduleError);
export const visitSchemaFor = (available: Officer[]) =>
  visitBase.refine(activeOfficer(available), officerError);
export const visitSchema = visitSchemaFor(officers);
const bookingFields = {
  ...visitFields,
  name: z.string().trim().min(2, "Masukkan nama lengkap.").max(120),
  organization: z.string().trim().min(2, "Masukkan instansi atau asal."),
  email: emailSchema,
  phone: z.string().regex(/^[0-9]{8,20}$/, "Gunakan 8–20 digit angka untuk nomor telepon."),
  identityType: z.enum(["KTP", "Paspor", "SIM"]),
  visitors: z
    .array(
      z.object({
        name: z.string().trim().min(2, "Lengkapi nama pengunjung.").max(120),
      }),
    )
    .max(19),
};
const bookingBase = z
  .object({
    ...bookingFields,
    ack: privacyAck,
  })
  .refine(futureStart, scheduleError);
export const bookingSchemaFor = (available: Officer[]) =>
  bookingBase.refine(activeOfficer(available), officerError);
export const bookingSchema = bookingSchemaFor(officers);
export type BookingDraft = z.input<typeof bookingSchema>;
const namedRoster = z
  .array(z.string().trim().min(2, "Lengkapi nama pengunjung.").max(120))
  .min(1, "Lengkapi rombongan pengunjung.")
  .max(20, "Rombongan maksimal 20 orang.");
const documentName = z
  .string()
  .trim()
  .min(1, "Pilih nama dokumen yang valid.")
  .max(255, "Nama dokumen maksimal 255 karakter.")
  .optional();
export const publicBookingInputSchema = z
  .object({
    ...bookingFields,
    visitKind: z.literal("standard").default("standard"),
    officerId: z.string().trim().min(1, "Pilih tujuan kunjungan."),
    date: calendarDate,
    email: emailSchema.transform((email) => email.toLowerCase()),
    visitors: namedRoster,
    ndName: documentName,
    attachmentToken: z.string().min(16).max(4096).optional(),
    privacyVersion: z.string().min(1).max(120).optional(),
    ack: privacyAck,
  })
  .refine((data) => data.visitors[0] === data.name, {
    path: ["visitors"],
    message: "Nama perwakilan harus menjadi pengunjung pertama.",
  });
export type PublicBookingInput = z.output<typeof publicBookingInputSchema>;
export function publicBookingCreateSchema(now = Date.now()) {
  return publicBookingInputSchema.superRefine((data, ctx) => {
    if (!officers.some((officer) => officer.id === data.officerId && officer.active))
      ctx.addIssue({
        code: "custom",
        path: ["officerId"],
        message: "Pilih pejabat aktif yang tersedia.",
      });
    const currentDate = new Intl.DateTimeFormat("en-CA", {
      timeZone: "Asia/Makassar",
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
    }).format(now);
    if (data.date < currentDate)
      ctx.addIssue({
        code: "custom",
        path: ["date"],
        message: "Pilih tanggal yang valid, hari ini atau sesudahnya.",
      });
    if (Date.parse(`${data.date}T${data.time}:00+08:00`) <= now)
      ctx.addIssue({ code: "custom", path: ["time"], message: scheduleError.message });
  });
}
export const visitNoteFields = {
  visitorInstructions: z.string().trim().max(2000, "Maksimal 2000 karakter.").optional(),
  coordinationNote: z.string().trim().max(2000, "Maksimal 2000 karakter.").optional(),
};
export const staffAppointmentSchema = z
  .object({ ...bookingFields, ...visitNoteFields })
  .refine(futureStart, scheduleError);
export type StaffAppointmentDraft = z.input<typeof staffAppointmentSchema>;
export const staffAppointmentSchemaFor = (available: Officer[]) =>
  staffAppointmentSchema.refine(activeOfficer(available), officerError);
export const standardStaffAppointmentInputSchema = z
  .object({
    ...bookingFields,
    ...visitNoteFields,
    visitKind: z.literal("standard").optional(),
    visitors: namedRoster,
    ndName: documentName,
    attachmentToken: z.string().min(16).max(4096).optional(),
  })
  .refine(futureStart, scheduleError)
  .refine((data) => data.visitors[0] === data.name, {
    path: ["visitors"],
    message: "Nama perwakilan harus menjadi pengunjung pertama.",
  });
const optionalText = (schema: z.ZodString) =>
  z
    .union([schema, z.string().trim().length(0)])
    .optional()
    .transform((value) => value || undefined);
export const restrictedAppointmentSchema = z
  .object({
    ...visitNoteFields,
    visitKind: z.literal("restricted"),
    date: visitFields.date,
    time: visitFields.time,
    officerId: optionalText(z.string().trim().min(1)),
    name: optionalText(bookingFields.name),
    organization: optionalText(bookingFields.organization.max(255)),
    serviceId: optionalText(
      z.string().trim().min(1).max(100, "No ID Dinas maksimal 100 karakter."),
    ),
  })
  .strict()
  .refine(futureStart, scheduleError);
export const staffAppointmentInputSchema = z.union([
  restrictedAppointmentSchema,
  standardStaffAppointmentInputSchema,
]);
export type RestrictedAppointmentDraft = z.input<typeof restrictedAppointmentSchema>;
export type StaffAppointmentInput = z.input<typeof staffAppointmentInputSchema>;
