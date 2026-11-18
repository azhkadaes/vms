import { ImagePlus, Upload } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { Button } from "./shell";
import { FieldMessage } from "./field-message";

export function UploadField({
  id,
  label,
  accept,
  file,
  error,
  hint,
  choose,
  remove,
  photoPreviewLabel,
  required,
  disabled,
  onFileChange,
}: {
  id: string;
  label: string;
  accept: string;
  file: File | null;
  error?: string | null;
  hint: string;
  choose: string;
  remove: string;
  photoPreviewLabel?: string;
  required?: boolean;
  disabled?: boolean;
  onFileChange: (file: File | null) => void;
}) {
  const input = useRef<HTMLInputElement>(null);
  const [url, setUrl] = useState<string>();
  useEffect(() => {
    const preview = photoPreviewLabel && file ? URL.createObjectURL(file) : undefined;
    setUrl(preview);
    return () => {
      if (preview) URL.revokeObjectURL(preview);
    };
  }, [photoPreviewLabel, file]);

  const Icon = photoPreviewLabel ? ImagePlus : Upload;
  return (
    <div className="field upload-field">
      <label htmlFor={id}>
        {label}
        {required && (
          <span className="required-mark" aria-hidden="true">
            {" "}*
          </span>
        )}
      </label>
      <div className="upload-field-actions">
        <span className="button secondary upload-field-picker">
          <Icon size={16} aria-hidden="true" />
          {choose}
          <input
            ref={input}
            id={id}
            type="file"
            accept={accept}
            disabled={disabled}
            required={required && !file}
            aria-required={required}
            aria-invalid={!!error}
            aria-describedby={`${id}-hint${file ? ` ${id}-file` : ""}${error ? ` ${id}-error` : ""}`}
            onChange={(event) => {
              onFileChange(event.currentTarget.files?.[0] ?? null);
              event.currentTarget.value = "";
            }}
          />
        </span>
        {(file || error) && (
          <Button
            type="button"
            kind="ghost"
            disabled={disabled}
            onClick={() => {
              onFileChange(null);
              input.current?.focus();
            }}
          >
            {remove}
          </Button>
        )}
      </div>
      {file && (
        <div className="upload-field-file" id={`${id}-file`}>
          {url && <img src={url} alt={photoPreviewLabel} width={56} height={56} />}
          <span>{file.name}</span>
        </div>
      )}
      <FieldMessage id={`${id}-hint`}>{hint}</FieldMessage>
      {error && (
        <FieldMessage id={`${id}-error`} tone="error">
          {error}
        </FieldMessage>
      )}
    </div>
  );
}
