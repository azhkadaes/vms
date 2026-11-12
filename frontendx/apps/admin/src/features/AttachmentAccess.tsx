import { downloadAttachment } from "@vms/data/attachments";
import { Button, ErrorNotice } from "@vms/ui";
import { useState } from "react";

export default function AttachmentAccess({ id, name }: { id: string; name: string }) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>();
  return (
    <div className="section-gap">
      <Button
        kind="secondary"
        disabled={busy}
        onClick={async () => {
          setBusy(true);
          setError(undefined);
          try {
            await downloadAttachment(id, name);
          } catch (error) {
            setError(error);
          } finally {
            setBusy(false);
          }
        }}
      >
        {busy ? "Mengunduh…" : "Unduh dokumen pendukung"}
      </Button>
      <ErrorNotice error={error} />
    </div>
  );
}
