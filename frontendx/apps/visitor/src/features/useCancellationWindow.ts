import { canVisitorCancel, type PublicBooking } from "@vms/data";
import { useEffect, useMemo, useReducer } from "react";

export function useCancellationWindow(booking: PublicBooking | null) {
  // Anchor the deadline to service time rather than the device wall clock.
  const clock = useMemo(
    () => ({ at: performance.now(), server: booking?.timing?.serverTime ?? Date.now() }),
    [booking],
  );
  const [, redraw] = useReducer((value) => value + 1, 0);
  useEffect(() => {
    if (!booking || !["pending", "approved"].includes(booking.status)) return;
    const start = Date.parse(`${booking.date}T${booking.time}:00+08:00`);
    let timer: ReturnType<typeof setTimeout>;
    function update() {
      clearTimeout(timer);
      redraw();
      const remaining = start - (clock.server + performance.now() - clock.at);
      if (remaining > 0) timer = setTimeout(update, Math.min(remaining, 2_147_483_647));
    }
    update();
    window.addEventListener("focus", update);
    return () => {
      clearTimeout(timer);
      window.removeEventListener("focus", update);
    };
  }, [booking, clock]);
  return !!booking && canVisitorCancel(booking, clock.server + performance.now() - clock.at);
}
