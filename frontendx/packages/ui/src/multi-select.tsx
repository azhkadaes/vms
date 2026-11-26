import * as Popover from "@radix-ui/react-popover";
import { Check, ChevronDown, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";

export function SearchableMultiSelect<Value extends string>({
  id,
  name,
  submitName,
  label,
  itemLabel,
  options,
  selected,
  onChange,
  placeholder,
  required,
  disabled,
  error,
  multiple = true,
  remoteSearch,
}: {
  id: string;
  name?: string;
  submitName?: string;
  label: string;
  itemLabel: string;
  options: { value: Value; label: string }[];
  selected: Value[];
  onChange: (values: Value[]) => void;
  placeholder: string;
  required?: boolean;
  disabled?: boolean;
  error?: string;
  multiple?: boolean;
  remoteSearch?: {
    onChange: (value: string) => void;
    selectedLabel?: string;
    emptyMessage?: string;
    maxLength?: number;
  };
}) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [active, setActive] = useState(0);
  const input = useRef<HTMLInputElement>(null);
  const control = useRef<HTMLDivElement>(null);
  const list = useRef<HTMLDivElement>(null);
  const visible = remoteSearch
    ? options
    : options.filter((option) =>
        option.label.toLocaleLowerCase().includes(search.trim().toLocaleLowerCase()),
      );
  const activeIndex = Math.min(active, Math.max(0, visible.length - 1));
  const listId = `${id}-options`;
  const selectedOption = options.find((option) => option.value === selected[0]);
  function updateSearch(value: string) {
    setSearch(value);
    remoteSearch?.onChange(value);
  }
  function changeOpen(next: boolean) {
    setOpen(next);
    if (!next) {
      if (!remoteSearch) updateSearch("");
      setActive(0);
    }
  }
  useEffect(() => {
    if (disabled) changeOpen(false);
  }, [disabled]);
  useEffect(() => {
    list.current?.querySelector('[data-highlighted="true"]')?.scrollIntoView({ block: "nearest" });
  }, [activeIndex, open, search]);
  function choose(value: Value) {
    if (disabled) return;
    onChange(
      multiple
        ? selected.includes(value)
          ? selected.filter((item) => item !== value)
          : [...selected, value]
        : [value],
    );
    if (!remoteSearch) updateSearch("");
    setActive(0);
    input.current?.focus({ preventScroll: true });
    if (!multiple) changeOpen(false);
  }
  return (
    <div className="field searchable-multiselect">
      {submitName &&
        selected.map((value) => (
          <input key={value} type="hidden" name={submitName} value={value} disabled={disabled} />
        ))}
      <label id={`${id}-label`} htmlFor={id}>
        {label}{" "}
        {required && (
          <span className="required-mark" aria-hidden="true">
            *
          </span>
        )}
      </label>
      <Popover.Root open={open && !disabled} onOpenChange={changeOpen}>
        <Popover.Anchor asChild>
          <div
            ref={control}
            className="multiselect-control"
            data-disabled={disabled || undefined}
            data-invalid={!!error || undefined}
            data-single={!multiple || undefined}
            onClick={(event) => {
              if (!disabled && event.target === event.currentTarget) input.current?.focus();
            }}
          >
            {multiple &&
              selected.map((value) => {
                const text =
                  options.find((option) => option.value === value)?.label || "Tidak tersedia";
                return (
                  <span className="multiselect-tag" key={value}>
                    <span>{text}</span>
                    {!disabled && (
                      <button
                        type="button"
                        aria-label={`Hapus ${itemLabel} ${text}`}
                        onClick={() => {
                          choose(value);
                          if (!open) changeOpen(false);
                        }}
                      >
                        <X size={12} aria-hidden="true" />
                      </button>
                    )}
                  </span>
                );
              })}
            <input
              ref={input}
              id={id}
              name={name}
              role="combobox"
              aria-autocomplete="list"
              aria-expanded={open && !disabled}
              aria-controls={open && !disabled ? listId : undefined}
              aria-activedescendant={
                open && visible.length ? `${id}-option-${activeIndex}` : undefined
              }
              aria-required={required}
              aria-invalid={!!error}
              aria-describedby={error ? `${id}-error` : undefined}
              autoComplete="off"
              maxLength={remoteSearch?.maxLength}
              disabled={disabled}
              placeholder={multiple && selected.length ? "Cari…" : placeholder}
              value={
                multiple || open
                  ? search
                  : selectedOption?.label || remoteSearch?.selectedLabel || ""
              }
              onFocus={() => setOpen(true)}
              onClick={() => setOpen(true)}
              onChange={(event) => {
                event.stopPropagation();
                updateSearch(event.target.value);
                setActive(0);
                setOpen(true);
              }}
              onKeyDown={(event) => {
                if (["ArrowDown", "ArrowUp"].includes(event.key)) {
                  event.preventDefault();
                  setOpen(true);
                  setActive(
                    open
                      ? (activeIndex + (event.key === "ArrowDown" ? 1 : -1) + visible.length) %
                          Math.max(1, visible.length)
                      : 0,
                  );
                } else if (event.key === "Enter") {
                  event.preventDefault();
                  if (open && visible[activeIndex]) choose(visible[activeIndex].value);
                  else setOpen(true);
                } else if (event.key === "Escape" && open) {
                  event.preventDefault();
                  event.stopPropagation();
                  changeOpen(false);
                } else if (event.key === "Tab") {
                  changeOpen(false);
                } else if (event.key === "Backspace" && !search && selected.length) {
                  event.preventDefault();
                  if (multiple) choose(selected[selected.length - 1]);
                  else {
                    onChange([]);
                    setOpen(true);
                  }
                }
              }}
            />
            {!multiple && selected.length > 0 && !disabled && (
              <button
                type="button"
                className="multiselect-clear"
                aria-label={`Hapus ${itemLabel} ${selectedOption?.label || "terpilih"}`}
                onClick={() => {
                  onChange([]);
                  updateSearch("");
                  input.current?.focus();
                }}
              >
                <X size={14} aria-hidden="true" />
              </button>
            )}
            <button
              type="button"
              className="multiselect-toggle"
              disabled={disabled}
              aria-label={`Tampilkan pilihan ${label.toLowerCase()}`}
              aria-expanded={open && !disabled}
              onMouseDown={(event) => event.preventDefault()}
              onClick={() => {
                if (open) changeOpen(false);
                else {
                  input.current?.focus();
                  setOpen(true);
                }
              }}
            >
              <ChevronDown size={16} aria-hidden="true" />
            </button>
          </div>
        </Popover.Anchor>
        <Popover.Portal>
          <Popover.Content
            className="multiselect-popover"
            role="presentation"
            align="start"
            sideOffset={4}
            collisionPadding={12}
            onOpenAutoFocus={(event) => event.preventDefault()}
            onCloseAutoFocus={(event) => event.preventDefault()}
            onInteractOutside={(event) => {
              if (event.target instanceof Node && control.current?.contains(event.target))
                event.preventDefault();
            }}
            onEscapeKeyDown={(event) => {
              event.preventDefault();
              event.stopPropagation();
              changeOpen(false);
            }}
          >
            <div
              ref={list}
              id={listId}
              role="listbox"
              aria-labelledby={`${id}-label`}
              aria-multiselectable={multiple || undefined}
            >
              {visible.map((option, index) => (
                <div
                  key={option.value}
                  id={`${id}-option-${index}`}
                  role="option"
                  aria-selected={selected.includes(option.value)}
                  data-highlighted={index === activeIndex}
                  className="multiselect-option"
                  onMouseDown={(event) => event.preventDefault()}
                  onPointerMove={() => setActive(index)}
                  onClick={() => choose(option.value)}
                >
                  <span>{option.label}</span>
                  {multiple ? (
                    <span className="multiselect-check" aria-hidden="true">
                      {selected.includes(option.value) && <Check size={12} />}
                    </span>
                  ) : (
                    selected.includes(option.value) && <Check size={15} aria-hidden="true" />
                  )}
                </div>
              ))}
            </div>
            {!visible.length && (
              <p className="multiselect-empty" role="status">
                {remoteSearch?.emptyMessage || "Tidak ada pilihan yang cocok."}
              </p>
            )}
          </Popover.Content>
        </Popover.Portal>
      </Popover.Root>
      {error && (
        <span id={`${id}-error`} className="field-error" role="alert">
          {error}
        </span>
      )}
    </div>
  );
}
