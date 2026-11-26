import { FileText, ImagePlus, Upload, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { Button } from "./shell";
import { FieldMessage } from "./field-message";
import { useUiLanguage } from "./language";
import { Modal } from "./index";

function sizeLabel(bytes: number) {
  return bytes >= 1024 * 1024
    ? `${(bytes / (1024 * 1024)).toFixed(1)} MB`
    : `${Math.max(1, Math.round(bytes / 1024))} KB`;
}

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
  const { t } = useUiLanguage();
  const input = useRef<HTMLInputElement>(null);
  const previewTrigger = useRef<HTMLButtonElement>(null);
  const [url, setUrl] = useState<string>();
  const [previewing, setPreviewing] = useState(false);
  useEffect(() => {
    const preview = file ? URL.createObjectURL(file) : undefined;
    setUrl(preview);
    if (!file) setPreviewing(false);
    return () => {
      if (preview) URL.revokeObjectURL(preview);
    };
  }, [file]);

  const isPhoto = !!photoPreviewLabel;
  const isPdf = file?.type === "application/pdf";
  const Icon = isPhoto ? ImagePlus : Upload;
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
      {file && (
        <div className="upload-field-file" id={`${id}-file`}>
          <button
            ref={previewTrigger}
            type="button"
            className="upload-field-thumb"
            disabled={!url}
            aria-label={t(`Pratinjau ${file.name}`, `Preview ${file.name}`)}
            onClick={() => setPreviewing(true)}
          >
            {isPhoto && url ? (
              <img src={url} alt={photoPreviewLabel} width={40} height={40} />
            ) : (
              <FileText size={20} aria-hidden="true" />
            )}
          </button>
          <span className="upload-field-meta">
            <span className="upload-field-name">{file.name}</span>
            <span className="upload-field-size">{sizeLabel(file.size)}</span>
          </span>
          <Button
            type="button"
            kind="ghost"
            className="upload-field-view"
            disabled={!url}
            onClick={() => setPreviewing(true)}
          >
            {t("Lihat", "View")}
          </Button>
          <Button
            type="button"
            kind="ghost"
            className="upload-field-remove"
            aria-label={remove}
            title={remove}
            disabled={disabled}
            onClick={() => {
              onFileChange(null);
              input.current?.focus();
            }}
          >
            <X size={16} aria-hidden="true" />
          </Button>
        </div>
      )}
      <div className="upload-field-actions">
        <span className={`button ${file ? "ghost" : "secondary"} upload-field-picker`}>
          <Icon size={16} aria-hidden="true" />
          {file ? t("Ganti berkas", "Replace file") : choose}
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
        {!file && error && (
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
      <FieldMessage id={`${id}-hint`}>{hint}</FieldMessage>
      {error && (
        <FieldMessage id={`${id}-error`} tone="error">
          {error}
        </FieldMessage>
      )}
      {file && url && previewing && (
        <Modal
          open
          layer="confirmation"
          className={`upload-preview-modal${isPdf ? " is-document" : ""}`}
          title={isPhoto ? (photoPreviewLabel ?? label) : label}
          description={`${file.name} · ${sizeLabel(file.size)}`}
          onClose={() => setPreviewing(false)}
          returnFocusTo={previewTrigger.current}
        >
          {isPdf ? (
            navigator.pdfViewerEnabled ? (
              <iframe className="upload-preview-pdf" src={url} title={file.name} />
            ) : (
              <div className="upload-preview-stage">
                <p>
                  {t(
                    "Browser ini tidak dapat menampilkan PDF di dalam halaman.",
                    "This browser cannot show PDFs inside the page.",
                  )}
                </p>
                <a className="button secondary" href={url} target="_blank" rel="noopener">
                  {t("Buka di tab baru", "Open in new tab")}
                </a>
              </div>
            )
          ) : (
            <div className="upload-preview-stage">
              <img src={url} alt={photoPreviewLabel ?? file.name} />
            </div>
          )}
        </Modal>
      )}
    </div>
  );
}
