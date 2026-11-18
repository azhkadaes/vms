import { preview, roles, useSession, type Role } from "@vms/data";
import { Select, Button, ErrorNotice, Modal } from "@vms/ui";
import { useState } from "react";

export function RoleSelection() {
  const { pendingSession, switchRole, signOut } = useSession();
  const [choice, setChoice] = useState<Role | "">("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>();
  return (
    <main className="login-page">
      <Modal
        open
        dismissible={false}
        onClose={() => {}}
        title="Pilih peran aktif"
        description="Pilih peran untuk membuka ruang kerja. Anda dapat menggantinya melalui sidebar."
      >
        <ErrorNotice error={error} />
        <form
          onSubmit={async (event) => {
            event.preventDefault();
            if (!choice || busy) return;
            setBusy(true);
            setError(undefined);
            try {
              await switchRole(choice);
            } catch (error) {
              setError(error);
            } finally {
              setBusy(false);
            }
          }}
        >
          <fieldset disabled={busy} className="management-fieldset">
            <legend>Peran yang tersedia</legend>
            {pendingSession?.availableRoles.map((role) => (
              <label className="check-label" key={role}>
                <input
                  type="radio"
                  name="active-role"
                  value={role}
                  required
                  checked={choice === role}
                  onChange={() => setChoice(role)}
                />
                {role}
              </label>
            ))}
          </fieldset>
          <div className="form-actions">
            <Button
              type="button"
              kind="secondary"
              disabled={busy}
              onClick={() => void signOut().catch(setError)}
            >
              Keluar
            </Button>
            <Button disabled={busy || !choice}>{busy ? "Membuka…" : "Buka ruang kerja"}</Button>
          </div>
        </form>
      </Modal>
    </main>
  );
}

export function RoleSwitcher() {
  const { session, switchRole } = useSession();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>();
  const available = preview ? roles : session?.availableRoles || [];
  if (available.length < 2) return null;
  return (
    <div className="sidebar-role-preview">
      <label htmlFor="active-role">{preview ? "Pratinjau peran" : "Ganti peran"}</label>
      <Select
        id="active-role"
        value={session?.role || ""}
        disabled={busy}
        aria-busy={busy}
        onChange={async (event) => {
          const next = available.find((role) => role === event.target.value);
          if (!next) return;
          setBusy(true);
          setError(undefined);
          try {
            await switchRole(next);
          } catch (error) {
            setError(error);
          } finally {
            setBusy(false);
          }
        }}
      >
        {available.map((role) => (
          <option key={role}>{role}</option>
        ))}
      </Select>
      <ErrorNotice error={error} />
    </div>
  );
}
