import type { ReactNode } from "react";
import { Button } from "./shell";

type Segment<Value extends string> = {
  value: Value;
  label: ReactNode;
  accessibleLabel?: string;
  lang?: string;
};

export function SegmentedControl<Value extends string>({
  label,
  options,
  value,
  onChange,
  className = "",
}: {
  label: string;
  options: readonly Segment<Value>[];
  value: Value | undefined;
  onChange: (value: Value) => void;
  className?: string;
}) {
  return (
    <div className={`segmented-control ${className}`} role="group" aria-label={label}>
      {options.map((option) => (
        <Button
          key={option.value}
          type="button"
          kind={value === option.value ? "secondary" : "ghost"}
          aria-pressed={value === option.value}
          aria-label={option.accessibleLabel}
          lang={option.lang}
          onClick={() => onChange(option.value)}
        >
          {option.label}
        </Button>
      ))}
    </div>
  );
}
