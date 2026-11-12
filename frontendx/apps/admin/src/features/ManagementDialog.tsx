import { ApiError, useUnsavedChanges } from "@vms/data";
import { Button, ErrorNotice, Modal } from "@vms/ui";
import { useRef, useState, type ReactNode } from "react";

export default function ManagementDialog({
  title,
  description,
  children,
  dirty,
  busy,
  error,
  onClose,
  onReload,
  returnFocusTo,
}: {
  title: string;
  description: string;
  children: ReactNode | ((requestClose: () => void) => ReactNode);
  dirty: boolean;
  busy: boolean;
  error: unknown;
  onClose: () => void;
  onReload?: () => Promise<void>;
  returnFocusTo?: HTMLElement | null;
}) {
  const [confirm, setConfirm] = useState<"close" | "reload" | null>(null);
  useUnsavedChanges(dirty, busy);
  const [reloadError, setReloadError] = useState<unknown>();
  const [reloading, setReloading] = useState(false);
  const resume = useRef<HTMLElement | null>(null);
  function requestClose() {
    if (busy) return;
    if (!dirty) {
      onClose();
      return;
    }
    resume.current = document.activeElement as HTMLElement;
    setConfirm("close");
  }
  return (
    <>
      <Modal
        open
        title={title}
        description={description}
        onClose={requestClose}
        returnFocusTo={returnFocusTo}
      >
        <ErrorNotice error={error} />
        {error instanceof ApiError && error.status === 409 && onReload && (
          <Button
            type="button"
            kind="secondary"
            onClick={(event) => {
              resume.current = event.currentTarget;
              setConfirm("reload");
            }}
          >
            Muat versi terbaru
          </Button>
        )}
        <fieldset className="management-editor-fields" disabled={busy} aria-label={title}>
          {typeof children === "function" ? children(requestClose) : children}
        </fieldset>
      </Modal>
      <Modal
        open={!!confirm}
        title={confirm === "reload" ? "Muat versi terbaru?" : "Buang perubahan?"}
        description="Perubahan yang belum disimpan akan dihapus."
        onClose={() => !reloading && setConfirm(null)}
        returnFocusTo={resume.current}
      >
        <ErrorNotice error={reloadError} />
        <div className="form-actions">
          <Button type="button" kind="secondary" onClick={() => setConfirm(null)}>
            Lanjutkan mengisi
          </Button>
          <Button
            type="button"
            kind="danger"
            disabled={reloading}
            onClick={async () => {
              if (confirm === "close") {
                onClose();
                return;
              }
              setReloading(true);
              setReloadError(undefined);
              try {
                await onReload?.();
                setConfirm(null);
              } catch (error) {
                setReloadError(error);
              } finally {
                setReloading(false);
              }
            }}
          >
            {reloading
              ? "Memuat…"
              : confirm === "reload"
                ? "Buang perubahan dan muat"
                : "Buang perubahan"}
          </Button>
        </div>
      </Modal>
    </>
  );
}
export function focusServerField(form: HTMLFormElement, error: unknown) {
  if (error instanceof ApiError && error.fieldErrors) {
    const name = Object.keys(error.fieldErrors)[0];
    form.querySelector<HTMLElement>(`[name="${CSS.escape(name)}"]`)?.focus();
  }
}
