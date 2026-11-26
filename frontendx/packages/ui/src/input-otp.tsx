import { forwardRef, useRef, useState, type InputHTMLAttributes } from "react";

type Props = Omit<InputHTMLAttributes<HTMLInputElement>, "value" | "onChange"> & {
  value?: string;
  onValueChange?: (value: string) => void;
  length?: number;
};

export const InputOTP = forwardRef<HTMLInputElement, Props>(function InputOTP(
  {
    value,
    onValueChange,
    length = 8,
    className = "",
    onFocus,
    onSelect,
    onClick,
    onInvalid,
    "aria-invalid": ariaInvalid,
    ...props
  },
  ref,
) {
  const [internal, setInternal] = useState("");
  const [caret, setCaret] = useState<number>();
  const [formatInvalid, setFormatInvalid] = useState(false);
  const slots = useRef<HTMLSpanElement>(null);
  const digits = value ?? internal;
  const activeSlot = Math.min(caret ?? digits.length, digits.length, length - 1);
  const invalid = formatInvalid || ariaInvalid === true || ariaInvalid === "true";

  return (
    <div className={`input-otp ${className}`} data-invalid={invalid || undefined}>
      <input
        {...props}
        ref={ref}
        type="text"
        inputMode="numeric"
        autoComplete={props.autoComplete || "off"}
        pattern={`[0-9]{${length}}`}
        maxLength={length}
        value={digits}
        aria-invalid={invalid ? true : ariaInvalid}
        onFocus={(event) => {
          setCaret(event.currentTarget.selectionStart ?? digits.length);
          onFocus?.(event);
        }}
        onSelect={(event) => {
          setCaret(event.currentTarget.selectionStart ?? digits.length);
          onSelect?.(event);
        }}
        onClick={(event) => {
          onClick?.(event);
          if (event.defaultPrevented || event.detail !== 1 || props.readOnly || !slots.current)
            return;
          const index = Array.from(slots.current.children).findIndex((slot) => {
            const { right } = slot.getBoundingClientRect();
            return event.clientX <= right;
          });
          const start = index < 0 ? digits.length : Math.min(index, digits.length);
          event.currentTarget.setSelectionRange(start, Math.min(start + 1, digits.length));
          setCaret(start);
        }}
        onInvalid={(event) => {
          setFormatInvalid(true);
          onInvalid?.(event);
        }}
        onChange={(event) => {
          const input = event.currentTarget;
          const next = input.value.replace(/\D/g, "").slice(0, length);
          if (value === undefined) setInternal(next);
          setCaret(Math.min(input.selectionStart ?? next.length, next.length));
          setFormatInvalid(false);
          onValueChange?.(next);
        }}
      />
      <span ref={slots} className="input-otp-slots" aria-hidden="true">
        {Array.from({ length }, (_, index) => (
          <span
            className="input-otp-slot"
            data-active={index === activeSlot || undefined}
            key={index}
          >
            {digits[index] || ""}
          </span>
        ))}
      </span>
    </div>
  );
});
