import { preview, repository, type PublicBooking } from "@vms/data";
import { Button, ErrorNotice } from "@vms/ui";
import { useEffect, useState } from "react";
import { useLanguage } from "../i18n";

export default function AdmissionQR({
  booking,
  ticket = false,
}: {
  booking: PublicBooking;
  ticket?: boolean;
}) {
  const { t } = useLanguage();
  const key = `${booking.id}:${booking.version}`;
  const [media, setMedia] = useState<{ key: string; url: string; expired?: boolean }>({
    key: "",
    url: "",
  });
  const [error, setError] = useState<unknown>();
  const [retry, setRetry] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    let alive = true,
      usable = true;
    let timer: ReturnType<typeof setTimeout>;
    setMedia({ key, url: "" });
    setError(undefined);
    async function load() {
      try {
        const requestedAt = performance.now();
        const admission = await repository.admission(booking, controller.signal);
        if (!alive) return;
        const lifetime =
          Date.parse(admission.expiresAt) -
          admission.serverTime -
          (performance.now() - requestedAt);
        const received = performance.now();
        function expire() {
          const remaining = lifetime - (performance.now() - received);
          if (remaining > 0) timer = setTimeout(expire, Math.min(remaining, 2_147_483_647));
          else {
            usable = false;
            if (alive) setMedia({ key, url: "", expired: true });
          }
        }
        expire();
        const qrcode = await import("qrcode");
        const url = await qrcode.toDataURL(admission.payload, {
          width: 280,
          margin: 3,
          color: { dark: "#0a2342", light: "#fffdf8" },
        });
        if (alive && usable) setMedia({ key, url });
      } catch (error) {
        if (alive && !controller.signal.aborted) setError(error);
      }
    }
    void load();
    return () => {
      alive = false;
      controller.abort();
      clearTimeout(timer);
    };
  }, [
    booking.id,
    booking.code,
    booking.email,
    booking.version,
    booking.timing?.serverTime,
    key,
    retry,
  ]);
  const current = media.key === key ? media : null;
  return (
    <div className="qr-display">
      {current?.url ? (
        <img
          src={current.url}
          width={280}
          height={280}
          alt={`${preview ? t("QR demo untuk", "Demo QR code for") : t("QR masuk untuk", "Admission QR code for")} ${booking.name}`}
        />
      ) : current?.expired ? (
        <p role="status">
          {t(
            "QR telah kedaluwarsa. Perbarui status kunjungan untuk memeriksa izin masuk.",
            "This QR code has expired. Update the visit status to check admission.",
          )}
        </p>
      ) : (
        !error && <p role="status">{t("Menyiapkan QR…", "Preparing QR code…")}</p>
      )}
      {!ticket && (
        <>
          <strong>{booking.name}</strong>
          <strong>{booking.code}</strong>
        </>
      )}
      <p>
        {preview
          ? t("QR preview · bukan izin masuk nyata.", "Preview QR code · not a real entry permit.")
          : t("QR diterbitkan oleh layanan kunjungan.", "QR code issued by the visit service.")}
        <br />
        {t(
          "Pemindaian tetap diikuti validasi oleh petugas.",
          "Staff must still validate the code after scanning.",
        )}
      </p>
      <ErrorNotice error={error} />
      {!!error && (
        <Button type="button" kind="secondary" onClick={() => setRetry((value) => value + 1)}>
          {t("Coba lagi", "Try again")}
        </Button>
      )}
      {!error && (current?.url || current?.expired) && (
        <Button type="button" kind="secondary" onClick={() => setRetry((value) => value + 1)}>
          {t("Perbarui QR", "Renew QR code")}
        </Button>
      )}
    </div>
  );
}
