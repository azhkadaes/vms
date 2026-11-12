import { forwardRef, lazy, Suspense } from "react";
import { RequiredMark } from "./index";
import { useUiLanguage } from "./language";

export type DatePickerHandle = { focus: () => void };
export type DatePickerProps = {
  id: string;
  name?: string;
  label: string;
  value?: string;
  defaultValue?: string;
  onChange?: (value: string) => void;
  onBlur?: () => void;
  min?: string;
  max?: string;
  required?: boolean;
  disabled?: boolean;
  readOnly?: boolean;
  error?: string;
};

const Control = lazy(() => import("./date-picker-control"));

// Load the accessible date controls only when a form actually needs them.
export const DatePicker = forwardRef<DatePickerHandle, DatePickerProps>(
  function DatePicker(props, ref) {
    const { t } = useUiLanguage();
    return (
      <Suspense fallback={
        <div className="field date-picker" aria-busy="true">
          <span className="field-label">{props.label} {props.required && <RequiredMark />}</span>
          <div className="date-picker-control" role="status">{t("Memuat kalender…", "Loading calendar…")}</div>
          <small>{t("Tanggal · WITA", "Date · WITA")}</small>
        </div>
      }>
        <Control {...props} ref={ref} />
      </Suspense>
    );
  },
);
