import { changePassword, useSession } from "@vms/data";
import { Button, ErrorNotice, RequiredFieldsHint, RequiredMark } from "@vms/ui";
import { useRef, useState } from "react";
import { focusServerField } from "./ManagementDialog";

export default function PasswordChange() {
  const { session, refreshSession, signOut } = useSession();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>();
  const [saved, setSaved] = useState(false);
  const lock = useRef(false);
  return (
    <main className="login-page">
      <section className="panel login-panel">
        <h1>Ganti kata sandi awal</h1>
        <p>Perbarui kata sandi untuk melanjutkan ke ruang kerja.</p>
        <ErrorNotice error={error} />
        {saved ? (
          <>
            <p role="status">Kata sandi diperbarui. Periksa sesi untuk melanjutkan.</p>
            <Button onClick={() => void refreshSession()}>Periksa sesi</Button>
          </>
        ) : (
          <form
            onSubmit={async (event) => {
              event.preventDefault();
              if (lock.current) return;
              const form = event.currentTarget;
              const data = new FormData(form);
              const current = String(data.get("currentPassword"));
              const next = String(data.get("newPassword"));
              if (current === next) {
                setError(new Error("Gunakan kata sandi baru yang berbeda."));
                form.querySelector<HTMLInputElement>('[name="newPassword"]')?.focus();
                return;
              }
              lock.current = true;
              setBusy(true);
              setError(undefined);
              try {
                await changePassword(current, next);
                form.reset();
                setSaved(true);
                await refreshSession();
              } catch (error) {
                setError(error);
                focusServerField(form, error);
              } finally {
                lock.current = false;
                setBusy(false);
              }
            }}
          >
            <RequiredFieldsHint />
            <div className="field">
              <label htmlFor="current-password">
                Kata sandi saat ini <RequiredMark />
              </label>
              <input
                id="current-password"
                name="currentPassword"
                type="password"
                autoComplete="current-password"
                required
              />
            </div>
            <div className="field">
              <label htmlFor="new-password">
                Kata sandi baru <RequiredMark />
              </label>
              <input
                id="new-password"
                name="newPassword"
                type="password"
                autoComplete="new-password"
                required
                minLength={session?.passwordMinLength}
                maxLength={256}
                aria-describedby="password-rule"
              />
              <small id="password-rule">
                Minimal {session?.passwordMinLength} karakter, sesuai aturan layanan.
              </small>
            </div>
            <Button type="submit" disabled={busy}>
              {busy ? "Menyimpan…" : "Simpan kata sandi"}
            </Button>
          </form>
        )}
        <Button kind="ghost" disabled={busy} onClick={() => void signOut().catch(setError)}>
          Keluar
        </Button>
      </section>
    </main>
  );
}
