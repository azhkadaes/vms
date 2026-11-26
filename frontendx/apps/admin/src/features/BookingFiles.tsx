import { ApiError, type Role, type StaffBooking } from "@vms/data";
import { visitFiles, type VisitFileKind } from "@vms/data/visit-files";
import { Button, ErrorNotice, Modal } from "@vms/ui";
import { Download, FileText, Image } from "lucide-react";
import { useEffect, useState } from "react";

type FileKind = VisitFileKind;

export default function BookingFiles({
  booking,
  role,
  credential,
  onAccessRejected,
}: {
  booking: StaffBooking;
  role: Role;
  credential?: string;
  onAccessRejected?: (error: ApiError) => void;
}) {
  const [viewer, setViewer] = useState<{ kind: FileKind; trigger: HTMLButtonElement }>();
  if (!booking.attachmentAvailable && !booking.visitorPhotoAvailable) return null;
  return (
    <>
      <div className="booking-file-actions section-gap">
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
        <BookingFileViewer
          key={`${booking.id}:${role}:${viewer.kind}`}
          id={booking.id}
          role={role}
          credential={credential}
          kind={viewer.kind}
          name={viewer.kind === "attachment" ? booking.ndName : booking.visitorPhotoName}
          description={viewer.kind === "attachment" ? booking.ndName : booking.name}
          onAccessRejected={onAccessRejected}
          onClose={() => setViewer(undefined)}
          returnFocusTo={viewer.trigger}
        />
      )}
    </>
  );
}

function BookingFileViewer({
  id,
  role,
  credential,
  kind,
  name,
  description,
  onAccessRejected,
  onClose,
  returnFocusTo,
}: {
  id: string;
  role: Role;
  credential?: string;
  kind: FileKind;
  name?: string;
  description?: string;
  onAccessRejected?: (error: ApiError) => void;
  onClose: () => void;
  returnFocusTo: HTMLButtonElement;
}) {
  const [file, setFile] = useState<{ url: string; type: string }>();
  const [error, setError] = useState<unknown>();
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    let objectUrl: string | undefined;
    setFile(undefined);
    setError(undefined);
    void visitFiles
      .open(id, kind, role, { credential, signal: controller.signal })
      .then((blob) => {
        if (controller.signal.aborted) return;
        objectUrl = URL.createObjectURL(blob);
        setFile({ url: objectUrl, type: blob.type });
      })
      .catch((error) => {
        if (controller.signal.aborted) return;
        if (
          onAccessRejected &&
          error instanceof ApiError &&
          [401, 403, 404, 409].includes(error.status)
        )
          onAccessRejected(error);
        else setError(error);
      });
    return () => {
      controller.abort();
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [id, role, credential, kind, attempt, onAccessRejected]);

  const title = kind === "attachment" ? "Nota Dinas" : "Foto pengunjung";
  const pdf = file?.type === "application/pdf";
  const extension = pdf ? "pdf" : file?.type === "image/png" ? "png" : "jpg";
  const filename = name?.replace(/[\\/\u0000-\u001f]/g, "_").trim() || `${title}.${extension}`;
  return (
    <Modal
      open
      layer="confirmation"
      className="booking-file-viewer"
      title={title}
      description={description}
      onClose={onClose}
      returnFocusTo={returnFocusTo}
      headerAction={
        file && (
          <a
            className="link-button secondary"
            href={file.url}
            download={filename}
            aria-label={kind === "attachment" ? "Unduh Nota Dinas" : "Unduh foto pengunjung"}
          >
            <Download size={16} aria-hidden="true" /> Unduh
          </a>
        )
      }
    >
      {error ? (
        <>
          <ErrorNotice error={error} />
          <Button kind="ghost" type="button" onClick={() => setAttempt((value) => value + 1)}>
            Coba lagi
          </Button>
        </>
      ) : !file ? (
        <p role="status">Memuat berkas…</p>
      ) : pdf ? (
        navigator.pdfViewerEnabled ? (
          <iframe className="booking-file-pdf" src={file.url} title="Pratinjau Nota Dinas" />
        ) : (
          <p>Pratinjau PDF tidak tersedia di browser ini.</p>
        )
      ) : (
        <img
          src={file.url}
          alt={title}
          onError={() => setError(new Error("Gambar tidak dapat ditampilkan."))}
        />
      )}
    </Modal>
  );
}
