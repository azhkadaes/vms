import { z } from "zod";
import { apiBase } from "./config.ts";
import { ApiError } from "./errors.ts";
import { createHttpClient } from "./http.ts";

const id = z.string().regex(/^[1-9][0-9]{0,9}$/);
const notificationSchema = z.object({
  id,
  visitId: id,
  event: z.enum(["created", "approved", "updated", "rescheduled", "cancelled", "checked_out"]),
  at: z.iso.datetime({ offset: true }),
  readAt: z.iso.datetime({ offset: true }).nullable(),
});
const pageSchema = z.object({
  items: z.array(notificationSchema),
  unreadCount: z.number().int().nonnegative(),
  nextBefore: id.optional(),
});
export type StaffNotification = z.infer<typeof notificationSchema>;
const request = createHttpClient(apiBase);

export async function listNotifications(before: string, signal: AbortSignal) {
  const path = before
    ? `/staff/notifications?before=${encodeURIComponent(before)}`
    : "/staff/notifications";
  const result = pageSchema.safeParse(await request(path, { authenticated: true, signal }));
  if (!result.success) throw new ApiError(502, "Daftar notifikasi tidak valid.");
  return result.data;
}

export async function markNotificationRead(id: string) {
  await request(`/staff/notifications/${encodeURIComponent(id)}/read`, {
    method: "POST",
    authenticated: true,
  });
}
