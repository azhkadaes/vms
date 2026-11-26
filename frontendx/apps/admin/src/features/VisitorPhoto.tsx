import type { Role } from "@vms/data";
import { visitFiles } from "@vms/data/visit-files";
import { Button, ErrorNotice, Modal } from "@vms/ui";
import { Eye } from "lucide-react";
import { useEffect, useRef, useState } from "react";

export default function VisitorPhoto({
  id,
  role,
  name,
}: {
  id: string;
  role: Role;
  name?: string;
}) {
  const [open, setOpen] = useState(false);
  const trigger = useRef<HTMLButtonElement | null>(null);
  return (
    <section className="visitor-photo-confirmation section-gap" aria-label="Foto pengunjung">
      <h3>Foto pengunjung</h3>
      <Button
        kind="secondary"
        type="button"
        onClick={(event) => {
          trigger.current = event.currentTarget;
          setOpen(true);
        }}
      >
        <Eye size={16} aria-hidden="true" /> Lihat foto
      </Button>
      {open && (
        <Modal
          open
          layer="confirmation"
          className="visitor-photo-viewer"
          title="Foto pengunjung"
          description={name || "Foto wajah perwakilan"}
          onClose={() => setOpen(false)}
          returnFocusTo={trigger.current}
        >
          <PhotoContent id={id} role={role} name={name} />
        </Modal>
      )}
    </section>
  );
}

function PhotoContent({ id, role, name }: { id: string; role: Role; name?: string }) {
  const [attempt, setAttempt] = useState(0);
  const [photo, setPhoto] = useState<{ id: string; url: string }>();
  const [error, setError] = useState<unknown>();
  useEffect(() => {
    const controller = new AbortController();
    let url: string | undefined;
    setPhoto(undefined);
    setError(undefined);
    void visitFiles
      .open(id, "visitor-photo", role, { signal: controller.signal })
      .then((blob) => {
        if (controller.signal.aborted) return;
        url = URL.createObjectURL(blob);
        setPhoto({ id, url });
      })
      .catch((error) => {
        if (!controller.signal.aborted) setError(error);
      });
    return () => {
      controller.abort();
      if (url) URL.revokeObjectURL(url);
    };
  }, [id, role, attempt]);
  if (error)
    return (
      <div>
        <ErrorNotice error={error} />
        <Button kind="ghost" type="button" onClick={() => setAttempt((value) => value + 1)}>
          Coba lagi
        </Button>
      </div>
    );

  if (photo?.id === id)
    return (
      <img
        src={photo.url}
        alt={`Foto pengunjung ${name || "perwakilan"}`}
        onError={() => setError(new Error("Foto tidak dapat ditampilkan. Coba lagi."))}
      />
    );

  return (
    <p role="status" className="muted">
      Memuat foto pengunjung…
    </p>
  );
}
