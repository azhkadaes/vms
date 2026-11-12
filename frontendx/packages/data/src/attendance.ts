import type { Status } from "./domain.ts";

export type DisplayStatus = Status | "missed";
export interface VisitTiming {
  serverTime: number;
  opensAt: number;
  endsAt: number;
  displayStatus: DisplayStatus;
  canCheckIn: boolean;
  canCheckOut: boolean;
  canCancel: boolean;
}
type TimedVisit = {
  date: string;
  time: string;
  duration: number;
  status: Status;
  timing?: VisitTiming;
};

export function getVisitTiming(visit: TimedVisit, now = Date.now()): VisitTiming {
  const startsAt = Date.parse(`${visit.date}T${visit.time}:00+08:00`);
  const opensAt = startsAt - 30 * 60_000;
  const endsAt = startsAt + visit.duration * 60_000;
  const missed = visit.status === "approved" && now > endsAt;
  return {
    serverTime: now,
    opensAt,
    endsAt,
    displayStatus: missed ? "missed" : visit.status,
    canCheckIn: visit.status === "approved" && now >= opensAt && now <= endsAt,
    canCheckOut: visit.status === "checked_in",
    canCancel: canVisitorCancel(visit, now),
  };
}
export function visitTiming(visit: TimedVisit): VisitTiming {
  return visit.timing || getVisitTiming(visit);
}
export function canVisitorCancel(
  visit: Pick<TimedVisit, "date" | "time" | "status">,
  now = Date.now(),
) {
  return (
    ["pending", "approved"].includes(visit.status) &&
    now < Date.parse(`${visit.date}T${visit.time}:00+08:00`)
  );
}
