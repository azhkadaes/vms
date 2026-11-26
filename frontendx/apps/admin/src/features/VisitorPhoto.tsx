import { loadVisitorPhoto } from "@vms/data/attachments";
import { Button } from "@vms/ui";
import { Eye } from "lucide-react";
import { useCallback, useRef, useState } from "react";
import FileViewer from "./FileViewer";

export default function VisitorPhoto({
  id,
  role,
  name,
}: {
  id: string;
  role: string;
  name?: string;
}) {
  const [open, setOpen] = useState(false);
  const trigger = useRef<HTMLButtonElement | null>(null);
  const load = useCallback((signal: AbortSignal) => loadVisitorPhoto(id, role, signal), [id, role]);
  return (
    <section className="visitor-photo-confirmation section-gap" aria-label="Foto pengunjung">
      <h3>Foto pengunjung</h3>
      <Button
        kind="ghost"
        type="button"
        onClick={(event) => {
          trigger.current = event.currentTarget;
          setOpen(true);
        }}
      >
        <Eye size={16} aria-hidden="true" /> Lihat foto
      </Button>
      {open && (
        <FileViewer
          title="Foto pengunjung"
          description={name || "Foto wajah perwakilan"}
          file={{ kind: "image", alt: `Foto pengunjung ${name || "perwakilan"}` }}
          load={load}
          onClose={() => setOpen(false)}
          returnFocusTo={trigger.current}
        />
      )}
    </section>
  );
}
