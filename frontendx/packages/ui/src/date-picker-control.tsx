import { forwardRef, lazy, Suspense, useImperativeHandle, useRef, useState } from "react";
import { CalendarDays, X } from "lucide-react";
import { TZDate } from "react-day-picker";
import { Button } from "./components/ui/button";
import { Popover, PopoverTrigger, PopoverContent, PopoverClose } from "./components/ui/popover";
import { RequiredMark } from "./index";
import type { DatePickerHandle, DatePickerProps } from "./date-picker";
import { useUiLanguage } from "./language";

const loadCalendar = () => import("./components/lightswind/calendar");
const Calendar = lazy(() => loadCalendar().then((module) => ({ default: module.Calendar })));
function preloadCalendar() {
  // A failed speculative fetch must not become an unhandled rejection.
  // Opening the calendar still reports a load failure through the route boundary.
  void loadCalendar().catch(() => undefined);
}

function parseDate(value?: string) {
  if (!value || !/^\d{4}-\d{2}-\d{2}$/.test(value)) return undefined;
  const [year, month, day] = value.split("-").map(Number);
  const date = new TZDate(year, month - 1, day, 12, 0, 0, "Asia/Makassar");
  return date.getFullYear() === year && date.getMonth() === month - 1 && date.getDate() === day
    ? date
    : undefined;
}
function isoDate(date: Date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
}
const DatePickerControl = forwardRef<DatePickerHandle, DatePickerProps>(function DatePicker(
  {
    id,
    name,
    label,
    value,
    defaultValue,
    onChange,
    onBlur,
    min,
    max,
    required,
    disabled,
    readOnly,
    error,
  },
  ref,
) {
  const { language, t } = useUiLanguage();
  const dateFormat = new Intl.DateTimeFormat(language === "en" ? "en-US" : "id-ID", {
    dateStyle: "long",
    timeZone: "Asia/Makassar",
  });
  const root = useRef<HTMLDivElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const [internalValue, setInternalValue] = useState(defaultValue || "");
  const [nativeError, setNativeError] = useState<"required" | "range" | null>(null);
  const [open, setOpen] = useState(false);
  const [portal, setPortal] = useState<HTMLElement>();
  const selectedValue = value === undefined ? internalValue : value;
  const selected = parseDate(selectedValue);
  const minimum = parseDate(min);
  const maximum = parseDate(max);
  const message =
    error ||
    (nativeError === "range"
      ? t("Pilih tanggal dalam rentang yang tersedia.", "Choose a date within the available range.")
      : nativeError === "required"
        ? t("Pilih tanggal kunjungan.", "Choose a visit date.")
        : undefined);
  useImperativeHandle(ref, () => ({ focus: () => trigger.current?.focus() }), []);
  function changeDate(date?: Date) {
    const next = date ? isoDate(date) : "";
    if (value === undefined) setInternalValue(next);
    setNativeError(null);
    onChange?.(next);
    setOpen(false);
  }
  return (
    <div ref={root} className="field date-picker">
      <label htmlFor={id}>
        {label} {required && <RequiredMark />}
      </label>
      <Popover
        open={open}
        onOpenChange={(next) => {
          if (next) setPortal(root.current?.closest<HTMLElement>(".modal") ?? document.body);
          setOpen(next);
        }}
      >
        <PopoverTrigger asChild>
          <Button
            ref={trigger}
            id={id}
            className="date-picker-trigger date-picker-control"
            disabled={disabled || readOnly}
            onPointerEnter={preloadCalendar}
            onFocus={preloadCalendar}
            aria-label={`${label}${required ? t(", wajib diisi", ", required") : ""}: ${selected ? dateFormat.format(selected) : t("Pilih tanggal", "Choose date")}`}
            aria-invalid={!!message}
            aria-describedby={`${id}-hint${message ? ` ${id}-error` : ""}`}
            data-empty={!selected || undefined}
          >
            <CalendarDays size={18} aria-hidden="true" />
            <span>
              {selected ? dateFormat.format(selected) : t("Pilih tanggal", "Choose date")}
            </span>
          </Button>
        </PopoverTrigger>
        <PopoverContent
          container={portal}
          className="date-picker-popover"
          aria-label={`${t("Pilih", "Choose")} ${label.toLowerCase()}`}
          onEscapeKeyDown={(event) => {
            event.preventDefault();
            setOpen(false);
          }}
          onCloseAutoFocus={(event) => {
            event.preventDefault();
            trigger.current?.focus({ preventScroll: true });
          }}
        >
          <Suspense
            fallback={
              <div className="calendar-loading" role="status">
                {t("Memuat kalender…", "Loading calendar…")}
              </div>
            }
          >
            <Calendar
              mode="single"
              selected={selected}
              onSelect={changeDate}
              defaultMonth={selected || minimum}
              autoFocus
              startMonth={minimum || new TZDate(1900, 0, 1, "Asia/Makassar")}
              endMonth={maximum || new TZDate(2100, 11, 31, "Asia/Makassar")}
              disabled={[
                ...(minimum ? [{ before: minimum }] : []),
                ...(maximum ? [{ after: maximum }] : []),
              ]}
            />
          </Suspense>
          <div className="date-picker-footer">
            <span>WITA · UTC+8</span>
            <PopoverClose asChild>
              <Button
                variant="ghost"
                size="icon"
                aria-label={t("Tutup kalender", "Close calendar")}
              >
                <X size={17} />
              </Button>
            </PopoverClose>
          </div>
        </PopoverContent>
      </Popover>
      <input
        className="visually-hidden date-picker-value"
        type="date"
        name={name}
        value={selectedValue}
        required={required}
        min={min}
        max={max}
        readOnly={readOnly}
        disabled={disabled}
        tabIndex={-1}
        aria-hidden="true"
        onChange={(event) => changeDate(parseDate(event.target.value))}
        onInvalid={(event) => {
          event.preventDefault();
          setNativeError(selected ? "range" : "required");
          trigger.current?.focus();
        }}
      />
      <small id={`${id}-hint`}>{t("Tanggal · WITA", "Date · WITA")}</small>
      {message && (
        <span id={`${id}-error`} className="field-error" role="alert">
          {message}
        </span>
      )}
    </div>
  );
});
export default DatePickerControl;
