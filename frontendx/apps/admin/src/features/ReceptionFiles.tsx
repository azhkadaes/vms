import type { Role, StaffBooking } from "@vms/data";
import { loadArrivalFile } from "@vms/data/attachments";
import { Button } from "@vms/ui";
import { FileText, Image } from "lucide-react";
import { useCallback, useState } from "react";
import FileViewer from "./FileViewer";

type FileKind = Parameters<typeof loadArrivalFile>[3];

export default function ReceptionFiles({
  booking,
  role,
  credential,
}: {
  booking: StaffBooking;
  role: Role;
  credential: string;
}) {
  const [viewer, setViewer] = useState<{ kind: FileKind; trigger: HTMLButtonElement }>();
  const kind = viewer?.kind ?? "attachment";
  const load = useCallback(
    (signal: AbortSignal) => loadArrivalFile(booking.id, role, credential, kind, signal),
    [booking.id, role, credential, kind],
  );
  return (
    <section className="section-gap" aria-label="Dokumen verifikasi">
      <h3>Dokumen verifikasi</h3>
      <div className="reception-file-actions">
        {booking.attachmentAvailable ? (
          <Button
            kind="ghost"
            type="button"
            onClick={(event) => setViewer({ kind: "attachment", trigger: event.currentTarget })}
          >
            <FileText size={16} aria-hidden="true" /> Lihat Nota Dinas
          </Button>
        ) : (
          <p>Nota Dinas tidak tersedia.</p>
        )}
        {booking.visitorPhotoAvailable ? (
          <Button
            kind="ghost"
            type="button"
            onClick={(event) => setViewer({ kind: "visitor-photo", trigger: event.currentTarget })}
          >
            <Image size={16} aria-hidden="true" /> Lihat foto pengunjung
          </Button>
        ) : (
          <p>Foto pengunjung tidak tersedia.</p>
        )}
      </div>
      {viewer && (
        <FileViewer
          title={kind === "attachment" ? "Nota Dinas" : "Foto pengunjung"}
          description={kind === "attachment" ? booking.ndName : booking.name}
          file={
            kind === "attachment"
              ? { kind: "pdf", filename: booking.ndName || "Nota Dinas.pdf" }
              : { kind: "image", alt: "Foto pengunjung" }
          }
          load={load}
          onClose={() => setViewer(undefined)}
          returnFocusTo={viewer.trigger}
        />
      )}
    </section>
  );
}
