import type { ComponentProps } from "react";
import { Calendar as BaseCalendar } from "../ui/calendar";

/** Keeps the shared WITA calendar contract while applying the requested compact surface. */
export function Calendar({ className = "", ...props }: ComponentProps<typeof BaseCalendar>) {
  return <BaseCalendar className={`lightswind-calendar ${className}`} {...props} />;
}
