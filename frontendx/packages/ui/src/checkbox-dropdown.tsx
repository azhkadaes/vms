import { ChevronDown } from "lucide-react";
import { useEffect, useId, useState, type ReactNode } from "react";
import { Popover, PopoverContent, PopoverTrigger } from "./components/ui/popover";

export type CheckboxDropdownOption = { value: string; label: ReactNode; count?: number };

/**
 * A filter button that opens a list of checkboxes. An empty selection means every option,
 * so the trigger reads `allLabel` and nothing is filtered out.
 */
export function CheckboxDropdown({
  label,
  allLabel,
  options,
  selected,
  onChange,
  className = "",
}: {
  label: string;
  allLabel: string;
  options: CheckboxDropdownOption[];
  selected: string[];
  onChange: (values: string[]) => void;
  className?: string;
}) {
  const id = useId();
  // Ticks show at once; the caller's value (often the URL) catches up a moment later.
  const [current, setCurrent] = useState(selected);
  const key = selected.join(",");
  useEffect(() => setCurrent(key ? key.split(",") : []), [key]);
  const chosen = options.filter((option) => current.includes(option.value));
  const summary =
    chosen.length === 0
      ? allLabel
      : chosen.length === 1
        ? chosen[0].label
        : `${chosen.length} dipilih`;
  function toggle(value: string, checked: boolean) {
    const next = checked ? [...current, value] : current.filter((v) => v !== value);
    // Keep the options' own order so the URL and the trigger read the same way every time.
    const ordered = options.map((o) => o.value).filter((v) => next.includes(v));
    setCurrent(ordered);
    onChange(ordered);
  }
  return (
    <div className={`field checkbox-dropdown ${className}`}>
      <span className="field-label" id={`${id}-label`}>
        {label}
      </span>
      <Popover>
        <PopoverTrigger asChild>
          <button
            type="button"
            className="checkbox-dropdown-trigger control"
            aria-labelledby={`${id}-label ${id}-value`}
            data-active={chosen.length > 0 || undefined}
          >
            <span id={`${id}-value`} className="checkbox-dropdown-value">
              {summary}
            </span>
            <ChevronDown size={16} aria-hidden="true" />
          </button>
        </PopoverTrigger>
        <PopoverContent className="checkbox-dropdown-popover" aria-labelledby={`${id}-label`}>
          <fieldset>
            <legend className="visually-hidden">{label}</legend>
            {options.map((option) => (
              <label key={option.value} className="checkbox-dropdown-option">
                <input
                  type="checkbox"
                  checked={current.includes(option.value)}
                  onChange={(event) => toggle(option.value, event.target.checked)}
                />
                <span>{option.label}</span>
                {option.count !== undefined && (
                  <span className="checkbox-dropdown-count">{option.count}</span>
                )}
              </label>
            ))}
          </fieldset>
          {chosen.length > 0 && (
            <button
              type="button"
              className="checkbox-dropdown-clear"
              onClick={() => {
                setCurrent([]);
                onChange([]);
              }}
            >
              Tampilkan semua
            </button>
          )}
        </PopoverContent>
      </Popover>
    </div>
  );
}
