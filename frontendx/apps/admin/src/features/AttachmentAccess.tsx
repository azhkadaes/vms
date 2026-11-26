import type { Role } from "@vms/data";
import { saveVisitFile } from "@vms/data/visit-files";
import { Button, ErrorNotice } from "@vms/ui";
import { useState } from "react";

export default function AttachmentAccess({
  id,
  name,
  role,
}: {
  id: string;
  name: string;
  role: Role;
}) {
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
            await saveVisitFile(id, "attachment", role, name);
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
