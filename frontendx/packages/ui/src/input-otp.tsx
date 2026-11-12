import { forwardRef, useState, type InputHTMLAttributes } from "react";

type Props = Omit<InputHTMLAttributes<HTMLInputElement>, "value" | "onChange"> & {
  value?: string;
  onValueChange?: (value: string) => void;
  length?: number;
};

/** Single native field with Lightswind-style digit slots; paste and autofill stay intact. */
export const InputOTP = forwardRef<HTMLInputElement, Props>(function InputOTP(
  { value, onValueChange, length = 8, className = "", ...props }, ref,
) {
  const [internal, setInternal] = useState("");
  const digits = value ?? internal;
  return <div className={`input-otp ${className}`}>
    <input {...props} ref={ref} type="text" inputMode="numeric" autoComplete={props.autoComplete || "off"}
      pattern={`[0-9]{${length}}`} maxLength={length} value={digits}
      onChange={(event) => {
        const next = event.target.value.replace(/\D/g, "").slice(0, length);
        if (value === undefined) setInternal(next);
        onValueChange?.(next);
      }} />
    <span className="input-otp-slots" aria-hidden="true">
      {Array.from({ length }, (_, index) => <span className="input-otp-slot" key={index}>{digits[index] || ""}</span>)}
    </span>
  </div>;
});
