import { ApiError, type Role } from "@vms/data";
import { visitFiles, type VisitFileKind } from "@vms/data/visit-files";
import { Button, ErrorNotice, Modal } from "@vms/ui";
import { ChevronRight, Download, ExternalLink, FileText, Image } from "lucide-react";
import { useEffect, useState } from "react";

const labels: Record<VisitFileKind, { title: string; open: string; fallback: string }> = {
  attachment: { title: "Nota Dinas", open: "Lihat Nota Dinas", fallback: "Nota Dinas.pdf" },
  "visitor-photo": {
    title: "Foto pengunjung",
    open: "Lihat foto pengunjung",
    fallback: "Foto pengunjung.jpg",
  },
};

/** Lists a visit's files as compact rows; each opens in a popup viewer without downloading. */
export default function VisitFileViewer({
  id,
  role,
  credential,
  attachmentName,
  showAttachment,
  showPhoto,
  visitorName,
  onAccessRejected,
}: {
  id: string;
  role: Role;
  credential?: string;
  attachmentName?: string;
  showAttachment: boolean;
  showPhoto: boolean;
  visitorName?: string;
  /** Reception only: a refused credential discards the resolved visit instead of showing an error. */
  onAccessRejected?: (error: ApiError) => void;
}) {
  const [viewer, setViewer] = useState<{ kind: VisitFileKind; trigger: HTMLButtonElement }>();
  if (!showAttachment && !showPhoto) return null;
  const kinds: VisitFileKind[] = [
    ...(showAttachment ? (["attachment"] as const) : []),
    ...(showPhoto ? (["visitor-photo"] as const) : []),
  ];
  const detail = (kind: VisitFileKind) =>
    kind === "attachment"
      ? attachmentName || "Dokumen PDF"
      : visitorName || "Foto wajah perwakilan";
  return (
    <section className="visit-files section-gap" aria-label="Berkas kunjungan">
      {kinds.map((kind) => {
        const Icon = kind === "attachment" ? FileText : Image;
        return (
          <Button
            key={kind}
            type="button"
            kind="action-card"
            className="visit-file-row"
            aria-label={labels[kind].open}
            onClick={(event) => setViewer({ kind, trigger: event.currentTarget })}
          >
            <Icon size={18} aria-hidden="true" className="visit-file-icon" />
            <span className="visit-file-text">
              <strong>{labels[kind].title}</strong>
              <span>{detail(kind)}</span>
            </span>
            <span className="visit-file-action" aria-hidden="true">
              Lihat <ChevronRight size={16} />
            </span>
          </Button>
        );
      })}
      {viewer && (
        <Modal
          open
          layer="confirmation"
          className={`visit-file-viewer${viewer.kind === "attachment" ? " is-document" : ""}`}
          title={labels[viewer.kind].title}
          description={detail(viewer.kind)}
          onClose={() => setViewer(undefined)}
          returnFocusTo={viewer.trigger}
        >
          <FileContent
            id={id}
            role={role}
            credential={credential}
            kind={viewer.kind}
            onAccessRejected={onAccessRejected}
            fileName={
              viewer.kind === "attachment" && attachmentName
                ? attachmentName
                : labels[viewer.kind].fallback
            }
          />
        </Modal>
      )}
    </section>
  );
}

function FileContent({
  id,
  role,
  credential,
  kind,
  fileName,
  onAccessRejected,
}: {
  id: string;
  role: Role;
  credential?: string;
  kind: VisitFileKind;
  fileName: string;
  onAccessRejected?: (error: ApiError) => void;
}) {
  const [url, setUrl] = useState<string>();
  const [error, setError] = useState<unknown>();
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    let objectUrl: string | undefined;
    setUrl(undefined);
    setError(undefined);
    void visitFiles
      .open(id, kind, role, { credential, signal: controller.signal })
      .then((blob) => {
        if (controller.signal.aborted) return;
        objectUrl = URL.createObjectURL(blob);
        setUrl(objectUrl);
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

  if (error)
    return (
      <div className="visit-file-stage is-message">
        <ErrorNotice error={error} />
        <Button kind="secondary" type="button" onClick={() => setAttempt((value) => value + 1)}>
          Coba lagi
        </Button>
      </div>
    );
  if (!url)
    return (
      <div className="visit-file-stage is-message">
        <p role="status" className="muted">
          Memuat berkas…
        </p>
      </div>
    );

  const download = (
    <a
      className="link-button ghost"
      href={url}
      download={fileName.replace(/[\\/\u0000-\u001f]/g, "_")}
    >
      <Download size={16} aria-hidden="true" /> Unduh {labels[kind].title}
    </a>
  );
  if (kind === "visitor-photo")
    return (
      <>
        <div className="visit-file-stage">
          <img
            src={url}
            alt="Foto pengunjung"
            onError={() => setError(new Error("Foto tidak dapat ditampilkan."))}
          />
        </div>
        <div className="visit-file-footer">{download}</div>
      </>
    );
  return (
    <>
      {navigator.pdfViewerEnabled ? (
        <iframe className="visit-file-pdf" src={url} title="Pratinjau Nota Dinas" />
      ) : (
        <div className="visit-file-stage is-message">
          <FileText size={32} aria-hidden="true" className="muted" />
          <p className="muted">Browser ini tidak dapat menampilkan PDF di dalam halaman.</p>
          <a className="link-button secondary" href={url} target="_blank" rel="noopener">
            <ExternalLink size={16} aria-hidden="true" /> Buka di tab baru
          </a>
        </div>
      )}
      <div className="visit-file-footer">{download}</div>
    </>
  );
}
