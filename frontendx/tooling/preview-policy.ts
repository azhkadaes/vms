/**
 * The preview server's fake of the Go visit policy (backend/internal/policy/visit.go,
 * AllowedActions). The real rules live in Go and reach the client as `allowedActions`
 * (ADR-0003); this copy only lets the preview adapter answer the same way (ADR-0001).
 */
import { getVisitTiming, visitTiming } from "../packages/data/src/attendance.ts";
import {
  isRestricted,
  scope,
  visitActions,
  type Session,
  type VisitAction,
} from "../packages/data/src/domain.ts";

type PolicyVisit = Parameters<typeof scope>[0][number] &
  Parameters<typeof visitTiming>[0] & { officerId?: string; vip?: boolean };

const rescheduleWindowMs = 7 * 24 * 60 * 60 * 1000;

export function can(s: Session, b: PolicyVisit, action: VisitAction, now?: number) {
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
      ? isRestricted(b) || !!b.vip
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
          timing.serverTime <= timing.endsAt + rescheduleWindowMs)
      : b.status === "pending")
  );
}

/** What the preview server sends as `allowedActions`, in the Go order. */
export function allowedActions(s: Session, b: PolicyVisit, now: number): VisitAction[] {
  return visitActions.filter((action) => can(s, b, action, now));
}
