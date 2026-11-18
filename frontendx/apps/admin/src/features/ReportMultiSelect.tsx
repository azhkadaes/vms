import { selectedFilters } from "@vms/data/report-filter";
import { SearchableMultiSelect } from "@vms/ui";
import { useState } from "react";

export default function ReportMultiSelect({
  id,
  name,
  label,
  placeholder,
  options,
  value,
  empty = "all",
  disabled,
  error,
  onChange,
}: {
  id: string;
  name: string;
  label: string;
  placeholder: string;
  options: { value: string; label: string }[];
  value: string | string[];
  empty?: string;
  disabled?: boolean;
  error?: string;
  onChange?: () => void;
}) {
  const [selected, setSelected] = useState(() => selectedFilters(value, empty));
  return (
    <SearchableMultiSelect
      id={id}
      submitName={name}
      label={label}
      itemLabel={label.toLowerCase()}
      options={options}
      selected={selected}
      onChange={(values) => {
        setSelected(values);
        onChange?.();
      }}
      placeholder={placeholder}
      disabled={disabled}
      error={error}
    />
  );
}
