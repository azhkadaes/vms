import { useId, type ComponentProps } from "react";

export default function VisitNotesFields({
  visitor = {},
  coordination = {},
  visitorError,
  coordinationError,
}: {
  visitor?: ComponentProps<"textarea">;
  coordination?: ComponentProps<"textarea">;
  visitorError?: string;
  coordinationError?: string;
}) {
  const id = useId();
  return (
    <div className="visit-note-fields span-2">
      <div className="field">
        <label htmlFor={`${id}-visitor`}>
          Instruksi pengunjung <small>(opsional)</small>
        </label>
        <textarea
          {...visitor}
          id={`${id}-visitor`}
          name="visitorInstructions"
          rows={3}
          maxLength={2000}
          aria-invalid={!!visitorError}
          aria-describedby={`${id}-visitor-hint${visitorError ? ` ${id}-visitor-error` : ""}`}
        />
        <small id={`${id}-visitor-hint`}>
          Terlihat oleh pengunjung pada detail kunjungan dan email pemberitahuan.
        </small>
        {visitorError && (
          <span id={`${id}-visitor-error`} className="field-error" role="alert">
            {visitorError}
          </span>
        )}
      </div>
      <div className="field">
        <label htmlFor={`${id}-coordination`}>
          Catatan koordinasi internal <small>(opsional)</small>
        </label>
        <textarea
          {...coordination}
          id={`${id}-coordination`}
          name="coordinationNote"
          rows={3}
          maxLength={2000}
          aria-invalid={!!coordinationError}
          aria-describedby={`${id}-coordination-hint${coordinationError ? ` ${id}-coordination-error` : ""}`}
        />
        <small id={`${id}-coordination-hint`}>
          Hanya untuk petugas yang berwenang. Tidak ditampilkan kepada pengunjung atau resepsionis.
        </small>
        {coordinationError && (
          <span id={`${id}-coordination-error`} className="field-error" role="alert">
            {coordinationError}
          </span>
        )}
      </div>
    </div>
  );
}
