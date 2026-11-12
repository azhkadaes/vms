import { useState } from "react";
import { createRoot } from "react-dom/client";
import { useQuery } from "@tanstack/react-query";
import { DataProvider } from "../../packages/data/src/query.tsx";
import { SessionProvider, useSession } from "../../packages/data/src/session.tsx";
import { useUnsavedChanges } from "../../packages/data/src/unsaved.ts";
import { sessionKey } from "../../packages/data/src/session-contract.ts";
import { RoleSwitcher } from "../../apps/admin/src/features/RoleControls.tsx";

// Existing editors are modal. This test-only inline editor makes the shared switch guard reachable.
function InlineDraft() {
  const { session, loading } = useSession();
  const [draft, setDraft] = useState("");
  const [busy, setBusy] = useState(false);
  useUnsavedChanges(draft.length > 0, busy);
  const cached = useQuery({
    queryKey: ["gap-2-cached-proof", sessionKey(session)],
    queryFn: async () => "Synthetic cached workspace",
    initialData: "Synthetic cached workspace",
    enabled: false,
  });
  if (loading || !session) return <p>Loading test session</p>;
  return <main>
    <h1>Test-only role-switch guard fixture</h1>
    <p>This inline editor is a test fixture, not a product screen.</p>
    <p>Confirmed role: <output id="confirmed-role">{session.role}</output></p>
    <p>Cached workspace: <output id="cached-proof">{cached.data}</output></p>
    <label htmlFor="guard-draft">Synthetic unsaved draft</label>
    <textarea id="guard-draft" value={draft} onChange={(event) => setDraft(event.target.value)} />
    <label><input type="checkbox" checked={busy} onChange={(event) => setBusy(event.target.checked)} />Simulated in-flight operation</label>
    <RoleSwitcher />
  </main>;
}

createRoot(document.getElementById("root")!).render(
  <DataProvider><SessionProvider><InlineDraft /></SessionProvider></DataProvider>,
);
