import { FieldMessage } from "./field-message";
import { forwardRef, useImperativeHandle, useRef, useState } from "react";
import { ChevronDown, Clock3 } from "lucide-react";
import {
  TimePicker as ReuiTimePicker,
  TimePickerTrigger,
  TimePickerValue,
  TimePickerContent,
  TimePickerColumns,
  TimePickerFooter,
  TimePickerClear,
  type TimePickerI18nOverrides,
} from "./components/reui/time-picker";
import { RequiredMark } from "./index";
import { useUiLanguage } from "./language";

export type TimePickerHandle = { focus: () => void };
type TimePickerProps = {
  id: string;
  label: string;
  name?: string;
  value?: string;
  defaultValue?: string;
  onChange?: (value: string) => void;
  onBlur?: () => void;
  required?: boolean;
  disabled?: boolean;
  readOnly?: boolean;
  min?: string;
  max?: string;
  form?: string;
  error?: string;
};
const labels: TimePickerI18nOverrides = {
  labels: {
    placeholder: "Pilih waktu", hour: "Jam", minute: "Menit", second: "Detik",
    clear: "Hapus", confirm: "Pilih", panelLabel: "Pilih waktu", openPicker: "Buka pemilih waktu",
  },
};
const englishLabels: TimePickerI18nOverrides = {
  labels: { placeholder: "Choose time", panelLabel: "Choose time" },
};

export const TimePicker = forwardRef<TimePickerHandle, TimePickerProps>(
  function TimePicker({ id, label, error, value, defaultValue, onChange, onBlur, required, disabled, readOnly, name, min = "08:00", max = "16:00", form }, ref) {
    const { language, t } = useUiLanguage();
    const root = useRef<HTMLDivElement>(null);
    const trigger = useRef<HTMLButtonElement>(null);
    const [portal, setPortal] = useState<HTMLElement>();
    const [open, setOpen] = useState(false);
    const [nativeError, setNativeError] = useState(false);
    const message = error || (nativeError ? t("Pilih waktu mulai.", "Choose a start time.") : undefined);
    useImperativeHandle(ref, () => ({ focus: () => trigger.current?.focus() }), []);
    return <div ref={root} className="field time-picker" onInvalidCapture={(event) => {
      event.preventDefault();
      setNativeError(true);
      trigger.current?.focus();
    }}>
      <label id={`${id}-label`} htmlFor={id}>{label} {required && <RequiredMark />}</label>
      <ReuiTimePicker id={id} name={name} form={form}
        value={value === undefined ? undefined : value || null} defaultValue={defaultValue || null}
        onValueChange={(next) => { setNativeError(false); onChange?.(next || ""); }}
        hourCycle={24} granularity="minute" minuteStep={1} fade={false}
        min={min} max={max} disabled={disabled} readOnly={readOnly} required={required}
        invalid={!!message} i18n={language === "en" ? englishLabels : labels}
        aria-labelledby={`${id}-label`} aria-describedby={`${id}-hint${message ? ` ${id}-error` : ""}`}
        open={open} onOpenChange={(next) => {
          if (next) setPortal(root.current?.closest<HTMLElement>(".modal") ?? document.body);
          setOpen(next);
        }}>
        <TimePickerTrigger ref={trigger} className="time-picker-trigger picker-control"
          disabled={disabled || readOnly}>
          <Clock3 size={16} aria-hidden="true" />
          <TimePickerValue />
          <ChevronDown size={14} className="picker-chevron" aria-hidden="true" />
        </TimePickerTrigger>
        <TimePickerContent container={portal} className="time-picker-popover" onEscapeKeyDown={(event) => {
          event.preventDefault(); setOpen(false);
        }} onCloseAutoFocus={(event) => {
          event.preventDefault(); requestAnimationFrame(() => trigger.current?.focus({ preventScroll: true }));
        }}>
          <TimePickerColumns />
          <TimePickerFooter>
            <span className="picker-zone">WITA</span>
            <TimePickerClear />
          </TimePickerFooter>
        </TimePickerContent>
      </ReuiTimePicker>
      <FieldMessage id={`${id}-hint`}>
        {t(`Waktu mulai ${min.replace(":", ".")}–${max.replace(":", ".")} · WITA`, `Start time ${min}–${max} · WITA`)}
      </FieldMessage>
      {message && <FieldMessage id={`${id}-error`} tone="error">{message}</FieldMessage>}
    </div>;
  },
);
