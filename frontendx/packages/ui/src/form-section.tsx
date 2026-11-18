import type { ComponentPropsWithoutRef } from "react";

export function FormSection({
  legend,
  className = "",
  children,
  ...props
}: ComponentPropsWithoutRef<"fieldset"> & { legend: string }) {
  return (
    <fieldset {...props} className={`form-section ${className}`}>
      <legend>{legend}</legend>
      {children}
    </fieldset>
  );
}
