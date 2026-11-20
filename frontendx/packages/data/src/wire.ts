import { z } from "zod";
import { roles, statuses } from "./domain.ts";
import { publicVisitActions } from "./history.ts";

const timestamp = z.string().refine((value) => Number.isFinite(Date.parse(value)));
const timing = z.object({
  serverTime: z.number().finite(),
  opensAt: z.number().finite(),
  endsAt: z.number().finite(),
  displayStatus: z.enum([...statuses, "missed"]),
  canCheckIn: z.boolean(),
  canCheckOut: z.boolean(),
  canCancel: z.boolean(),
});
const schedule = {
  id: z.string().min(1),
  date: z.iso.date(),
  time: z.string().regex(/^(?:[01]\d|2[0-3]):[0-5]\d$/),
  duration: z.number().int().min(15).max(240),
  room: z.string(),
  status: z.enum(statuses),
  version: z.number().int().positive(),
  rescheduled: z.boolean().optional(),
  timing,
};
const details = {
  code: z.string().regex(/^\d{8}$/),
  name: z.string(),
  organization: z.string(),
  email: z.string(),
  phone: z.string(),
  officerId: z.string(),
  host: z.string().optional(),
  pejabat: z.object({ nama: z.string() }).optional(),
  vip: z.boolean(),
  purpose: z.string(),
  visitors: z.array(z.string()).max(20),
  identityType: z.string(),
};
const optionalDetails = z.object(details).partial().shape;
export const publicBookingResponse = z
  .object({
    ...schedule,
    ...details,
    visitKind: z.literal("standard").optional(),
    ndName: z.string().optional(),
    visitorInstructions: z.string().max(2000).optional(),
    rejectionReason: z.string().optional(),
    cancellationReason: z.string().optional(),
    history: z.array(
      z
        .object({ at: z.string(), action: z.string(), publicMessage: z.string().optional() })
        .transform((event) => ({
          at: event.at,
          action: publicVisitActions.find((action) => action === event.action) || "updated",
          ...(event.action === "rescheduled" && event.publicMessage
            ? { publicMessage: event.publicMessage }
            : {}),
        })),
    ),
  })
  .transform(({ pejabat, ...booking }, context) => {
    const host = pejabat?.nama ?? booking.host;
    if (host === undefined) {
      context.addIssue({ code: "custom", path: ["pejabat"], message: "Pejabat tidak tersedia." });
      return z.NEVER;
    }
    return { ...booking, host };
  });
export const staffBookingResponse = z
  .object({
    ...schedule,
    ...optionalDetails,
    visitKind: z.enum(["standard", "restricted"]).optional(),
    reference: z.string().optional(),
    restrictedView: z.enum(["full", "assigned-pa", "reception", "arrival"]).optional(),
    serviceId: z.string().optional(),
    serviceIdLast4: z.string().max(4).optional(),
    internalRejectionNote: z.string().optional(),
    visitorInstructions: z.string().max(2000).optional(),
    coordinationNote: z.string().max(2000).optional(),
    internalCancellationNote: z.string().optional(),
    cancellationReason: z.string().optional(),
    ndName: z.string().optional(),
    rejectionReason: z.string().optional(),
    createdBy: z.object({ name: z.string(), role: z.enum(roles) }).optional(),
    attachmentAvailable: z.boolean().optional(),
    visitorPhotoAvailable: z.boolean().optional(),
    visitorCount: z.number().int().min(0).max(20).optional(),
    visitorPhotoName: z.string().max(255).optional(),
    history: z.array(z.object({ at: timestamp, message: z.string() })).optional(),
  })
  .superRefine((record, context) => {
    if (record.visitKind !== "restricted") return;
    if (!record.restrictedView) {
      context.addIssue({
        code: "custom",
        path: ["restrictedView"],
        message: "Lingkup data terbatas harus ditentukan.",
      });
      return;
    }
    const allowed = new Set([
      ...Object.keys(schedule),
      "visitKind",
      "reference",
      "restrictedView",
      ...(record.restrictedView === "reception"
        ? []
        : ["name", "officerId", "host", "pejabat", "serviceIdLast4"]),
      ...(record.restrictedView === "assigned-pa" || record.restrictedView === "full"
        ? ["organization"]
        : []),
      ...(record.restrictedView === "full"
        ? [
            "code",
            "serviceId",
            "history",
            "createdBy",
            "rejectionReason",
            "internalRejectionNote",
            "cancellationReason",
            "internalCancellationNote",
            "visitorInstructions",
            "coordinationNote",
          ]
        : []),
    ]);
    for (const [key, value] of Object.entries(record))
      if (value !== undefined && !allowed.has(key))
        context.addIssue({
          code: "custom",
          path: [key],
          message: "Proyeksi data tidak sesuai kewenangan.",
        });
  })
  .transform(({ pejabat, ...booking }) => ({
    ...booking,
    ...(pejabat ? { host: pejabat.nama } : {}),
  }));
export const catalogsResponse = z.object({
  officers: z.array(
    z.object({
      id: z.string().min(1),
      name: z.string().min(1),
      vip: z.boolean(),
      active: z.boolean(),
      jobTitle: z.string().max(255).optional(),
      directorate: z.string().max(255).optional(),
      deputy: z.string().max(255).optional(),
    }),
  ),
  rooms: z
    .array(
      z.union([
        z.string().min(1),
        z.object({ id: z.string().min(1), name: z.string().min(1), active: z.boolean() }),
      ]),
    )
    .transform((rooms) =>
      rooms.flatMap((room) => (typeof room === "string" ? [room] : room.active ? [room.name] : [])),
    ),
  privacyNotice: z
    .object({ version: z.string().min(1), id: z.string().min(1), en: z.string().min(1) })
    .optional(),
});
const count = z.number().int().nonnegative();
export const staffReadModelResponse = z
  .object({
    items: z.array(staffBookingResponse).max(100),
    total: count,
    totalScoped: count,
    page: z.number().int().positive(),
    pageSize: z.number().int().min(1).max(100),
    metrics: z
      .object({ today: count, pending: count, arrivals: count, inside: count, completed: count })
      .optional(),
  })
  .refine(
    (value) =>
      value.totalScoped >= value.total &&
      value.items.length <= value.pageSize &&
      value.items.length <= value.total,
  );
export const admissionResponse = z.object({
  payload: z.string().min(1).max(4096),
  imagePng: z.string().min(1),
  expiresAt: timestamp,
  serverTime: z.number().finite(),
});
export const resolutionResponse = z.discriminatedUnion("outcome", [
  z.object({ outcome: z.literal("not_committed") }),
  z.object({ outcome: z.literal("committed"), booking: publicBookingResponse }),
]);
