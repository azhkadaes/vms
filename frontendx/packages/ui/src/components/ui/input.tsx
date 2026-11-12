// Adapted from shadcn/ui's Input (MIT; LICENSE.shadcn).
// https://ui.shadcn.com/r/styles/new-york-v4/input.json
import { forwardRef, type InputHTMLAttributes } from "react";
export const Input = forwardRef<HTMLInputElement, InputHTMLAttributes<HTMLInputElement>>(
  function Input({ className = "", type, ...props }, ref) {
    return <input ref={ref} type={type} data-slot="input"
      className={`shadcn-input ${className}`} {...props} />;
  },
);
