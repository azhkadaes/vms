// Adapted from shadcn/ui's Calendar and CalendarDayButton (MIT; LICENSE.shadcn).
// https://ui.shadcn.com/r/styles/new-york-v4/calendar.json
// Source structure/data slots retained; utility styling uses the app's scoped CSS.
import { useEffect, useRef, type ComponentProps } from "react";
import { ChevronDown, ChevronLeft, ChevronRight } from "lucide-react";
import { DayPicker, getDefaultClassNames, type DayButton } from "react-day-picker";
import { id } from "date-fns/locale/id";
import { enUS } from "date-fns/locale/en-US";
import { Button } from "./button";
import { useUiLanguage } from "../../language";

export function Calendar({ className = "", classNames, components, ...props }: ComponentProps<typeof DayPicker>) {
  const { language, t } = useUiLanguage();
  return <DayPicker showOutsideDays captionLayout="dropdown" fixedWeeks
    locale={language === "en" ? enUS : id} timeZone="Asia/Makassar"
    className={`shadcn-calendar ${className}`}
    classNames={{ ...getDefaultClassNames(), ...classNames }}
    components={{
      Root: ({ className, rootRef, ...props }) => <div ref={rootRef} data-slot="calendar" className={className} {...props} />,
      Chevron: ({ orientation, className }) => {
        const Icon = orientation === "left" ? ChevronLeft : orientation === "right" ? ChevronRight : ChevronDown;
        return <Icon className={className} size={16} aria-hidden="true" />;
      },
      DayButton: CalendarDayButton,
      ...components,
    }}
    labels={{
      labelPrevious: () => t("Bulan sebelumnya", "Previous month"),
      labelNext: () => t("Bulan berikutnya", "Next month"),
      labelMonthDropdown: () => t("Pilih bulan", "Choose month"),
      labelYearDropdown: () => t("Pilih tahun", "Choose year"),
    }} {...props} />;
}

export function CalendarDayButton({ className = "", day, modifiers, ...props }: ComponentProps<typeof DayButton>) {
  const ref = useRef<HTMLButtonElement>(null);
  useEffect(() => { if (modifiers.focused) ref.current?.focus(); }, [modifiers.focused]);
  return <Button ref={ref} variant="ghost" size="icon"
    data-day={`${day.date.getFullYear()}-${String(day.date.getMonth() + 1).padStart(2, "0")}-${String(day.date.getDate()).padStart(2, "0")}`}
    data-selected-single={modifiers.selected || undefined}
    data-today={modifiers.today || undefined}
    className={`shadcn-calendar-day ${className}`} {...props} />;
}
