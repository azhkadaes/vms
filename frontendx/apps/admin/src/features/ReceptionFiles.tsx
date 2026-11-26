import type { Role, StaffBooking } from "@vms/data";
import { visitFiles, type VisitFileKind } from "@vms/data/visit-files";
import { Button, ErrorNotice, Modal } from "@vms/ui";
import { FileText, Image } from "lucide-react";
import { useEffect, useState } from "react";

type FileKind = VisitFileKind;

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
  if (!booking.attachmentAvailable && !booking.visitorPhotoAvailable) return null;
  return (
    <>
      <div className="reception-file-actions section-gap">
        {booking.attachmentAvailable && (
          <Button
            kind="secondary"
            type="button"
            onClick={(event) => setViewer({ kind: "attachment", trigger: event.currentTarget })}
          >
            <FileText size={16} aria-hidden="true" /> Lihat Nota Dinas
          </Button>
        )}
        {booking.visitorPhotoAvailable && (
          <Button
            kind="secondary"
            type="button"
            onClick={(event) => setViewer({ kind: "visitor-photo", trigger: event.currentTarget })}
          >
            <Image size={16} aria-hidden="true" /> Lihat foto pengunjung
          </Button>
        )}
      </div>
      {viewer && (
        <Modal
          open
          layer="confirmation"
          className="visitor-photo-viewer"
          title={viewer.kind === "attachment" ? "Nota Dinas" : "Foto pengunjung"}
          description={viewer.kind === "attachment" ? booking.ndName : booking.name}
          onClose={() => setViewer(undefined)}
          returnFocusTo={viewer.trigger}
        >
          <FileContent id={booking.id} role={role} credential={credential} kind={viewer.kind} />
        </Modal>
      )}
    </>
  );
}

function FileContent({
  id,
  role,
  credential,
  kind,
}: {
  id: string;
  role: Role;
  credential: string;
  kind: FileKind;
}) {
  const [url, setUrl] = useState<string>();
  const [error, setError] = useState<unknown>();
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    let objectUrl: string | undefined;
    void visitFiles
      .open(id, kind, role, { credential, signal: controller.signal })
      .then((blob) => {
        if (controller.signal.aborted) return;
        objectUrl = URL.createObjectURL(blob);
        setUrl(objectUrl);
      })
      .catch((error) => {
        if (!controller.signal.aborted) setError(error);
      });
    return () => {
      controller.abort();
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [id, role, credential, kind, attempt]);

  if (error)
    return (
      <>
        <ErrorNotice error={error} />
        <Button
          kind="ghost"
          type="button"
          onClick={() => {
            setError(undefined);
            setUrl(undefined);
            setAttempt((value) => value + 1);
          }}
        >
          Coba lagi
        </Button>
      </>
    );
  if (!url) return <p role="status">Memuat berkas…</p>;
  if (kind === "visitor-photo")
    return (
      <img
        src={url}
        alt="Foto pengunjung"
        onError={() => setError(new Error("Foto tidak dapat ditampilkan."))}
      />
    );
  if (!navigator.pdfViewerEnabled)
    return <a href={url} download="Nota Dinas.pdf">Unduh Nota Dinas</a>;
  return <iframe className="reception-pdf-viewer" src={url} title="Pratinjau Nota Dinas" />;
}
