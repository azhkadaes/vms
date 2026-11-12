import type { Booking } from "./domain.ts";

type VisitSlot = Pick<Booking, "date" | "time" | "duration" | "officerId" | "room">;
type Reservation = VisitSlot & Pick<Booking, "id" | "status">;

function startMinutes(time: string) {
  const [hour, minute] = time.split(":").map(Number);
  return hour * 60 + minute;
}

export function hasReservationConflict(
  candidate: VisitSlot,
  reservations: readonly Reservation[],
  excludeId?: string,
) {
  const start = startMinutes(candidate.time);
  const end = start + candidate.duration;
  return reservations.some((other) => {
    if (
      other.id === excludeId ||
      other.date !== candidate.date ||
      (other.status !== "approved" && other.status !== "checked_in")
    )
      return false;
    const sameOfficer = !!candidate.officerId && candidate.officerId === other.officerId;
    const sameRoom = !!candidate.room && candidate.room === other.room;
    if (!sameOfficer && !sameRoom) return false;
    const otherStart = startMinutes(other.time);
    return start < otherStart + other.duration && otherStart < end;
  });
}
