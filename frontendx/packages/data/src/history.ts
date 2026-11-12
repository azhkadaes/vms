export const publicVisitActions = [
  "received",
  "approved",
  "rejected",
  "rescheduled",
  "cancelled",
  "checked_in",
  "checked_out",
  "updated",
] as const;
export type PublicVisitAction = (typeof publicVisitActions)[number];
export interface PublicVisitEvent {
  at: string;
  action: PublicVisitAction;
  publicMessage?: string;
}
export interface StoredVisitEvent {
  at: string;
  message: string;
  action?: string;
  publicMessage?: string;
}

// Legacy or unknown messages never become visitor copy.
export function publicVisitHistory(events: StoredVisitEvent[]): PublicVisitEvent[] {
  return events.map((event) => {
    const action = publicVisitActions.find((action) => action === event.action) || "updated";
    return {
      at: event.at,
      action,
      ...(action === "rescheduled" && event.publicMessage
        ? { publicMessage: event.publicMessage }
        : {}),
    };
  });
}
