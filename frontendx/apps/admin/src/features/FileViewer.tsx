import { Button, ErrorNotice, Modal } from "@vms/ui";
import { useEffect, useState } from "react";

type LoadFile = (signal: AbortSignal) => Promise<Blob>;

export default function FileViewer({
  title,
  description,
  file,
  load,
  onClose,
  returnFocusTo,
}: {
  title: string;
  description?: string;
  file: { kind: "pdf"; filename: string } | { kind: "image"; alt: string };
  load: LoadFile;
  onClose: () => void;
  returnFocusTo: HTMLElement | null;
}) {
  const [attempt, setAttempt] = useState(0);
  const [result, setResult] = useState<{ load: LoadFile; url?: string; error?: unknown }>();
  const current = result?.load === load ? result : undefined;

  useEffect(() => {
    const controller = new AbortController();
    let url: string | undefined;
    void load(controller.signal)
      .then((blob) => {
        if (controller.signal.aborted) return;
        url = URL.createObjectURL(blob);
        setResult({ load, url });
      })
      .catch((error) => {
        if (!controller.signal.aborted) setResult({ load, error });
      });
    return () => {
      controller.abort();
      if (url) URL.revokeObjectURL(url);
    };
  }, [load, attempt]);

  return (
    <Modal
      open
      layer="confirmation"
      className="file-viewer"
      title={title}
      description={description}
      onClose={onClose}
      returnFocusTo={returnFocusTo}
    >
      <div className="file-viewer-frame">
        {current?.error ? (
          <div className="file-viewer-status">
            <ErrorNotice error={current.error} />
            <Button
              kind="ghost"
              type="button"
              onClick={() => {
                setResult(undefined);
                setAttempt((value) => value + 1);
              }}
            >
              Coba lagi
            </Button>
          </div>
        ) : !current?.url ? (
          <p role="status" className="muted">
            Memuat berkas…
          </p>
        ) : file.kind === "image" ? (
          <img
            src={current.url}
            alt={file.alt}
            onError={() => setResult({ load, error: new Error("Foto tidak dapat ditampilkan.") })}
          />
        ) : navigator.pdfViewerEnabled ? (
          <iframe src={current.url} title={`Pratinjau ${title}`} />
        ) : (
          <a href={current.url} download={file.filename}>
            Unduh {title}
          </a>
        )}
      </div>
    </Modal>
  );
}
