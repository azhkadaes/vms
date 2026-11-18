import type { ComponentPropsWithoutRef } from "react";

export function FieldMessage({
  tone = "hint",
  className = "",
  ...props
}: ComponentPropsWithoutRef<"small"> & { tone?: "hint" | "error" }) {
  return (
    <small
      {...props}
      className={`${tone === "error" ? "field-error" : "form-help"} ${className}`}
      role={tone === "error" ? "alert" : props.role}
    />
  );
}
