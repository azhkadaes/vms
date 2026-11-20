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
  const checkedIn = booking.status === "checked_in";
  const [media, setMedia] = useState<{ key: string; url: string }>({
    key: "",
    url: "",
  });
  const [error, setError] = useState<unknown>();
  const [retry, setRetry] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    let expiresAt: number | undefined;
    let timer: ReturnType<typeof setTimeout>;

    function renewIfDue() {
      if (expiresAt === undefined || controller.signal.aborted) return;
      clearTimeout(timer);
      const remaining = expiresAt - performance.now();
      if (remaining <= 0) void load();
      else timer = setTimeout(renewIfDue, Math.min(Math.ceil(remaining), 2_147_483_647));
    }

    async function load() {
      expiresAt = undefined;
      setMedia({ key, url: "" });
      setError(undefined);
      try {
        const requestedAt = performance.now();
        const admission = await repository.admission(booking, controller.signal);
        if (controller.signal.aborted) return;
        const deadline = requestedAt + Date.parse(admission.expiresAt) - admission.serverTime;
        const url = admission.imagePng
          ? `data:image/png;base64,${admission.imagePng}`
          : await (await import("qrcode")).toDataURL(admission.payload, { width: 280 });
        if (controller.signal.aborted) return;
        if (!checkedIn && deadline <= performance.now())
          throw new Error("QR tidak tersedia. Coba lagi.");
        setMedia({ key, url });
        if (!checkedIn) {
          expiresAt = deadline;
          renewIfDue();
        }
      } catch (error) {
        if (!controller.signal.aborted) setError(error);
      }
    }

    document.addEventListener("visibilitychange", renewIfDue);
    void load();
    return () => {
      controller.abort();
      clearTimeout(timer);
      document.removeEventListener("visibilitychange", renewIfDue);
    };
  }, [
    booking.id,
    booking.code,
    booking.email,
    booking.version,
    booking.timing?.serverTime,
    checkedIn,
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
          alt={`${preview ? t("QR demo untuk", "Demo QR code for") : t("QR kunjungan untuk", "Visit QR code for")} ${booking.name}`}
        />
      ) : !error ? (
        <p className="qr-display__placeholder" role="status">
          {t("Menyiapkan QR…", "Preparing QR code…")}
        </p>
      ) : null}
      {!ticket && (
        <>
          <strong>{booking.name}</strong>
          <strong>{booking.code}</strong>
        </>
      )}
      <p>
        {preview
          ? t("QR preview · bukan izin masuk nyata.", "Preview QR code · not a real entry permit.")
          : checkedIn
            ? t("Tunjukkan QR ini saat check out.", "Show this QR code at check-out.")
            : t(
                "Gunakan QR yang sama untuk check in dan check out.",
                "Use the same QR code for check-in and check-out.",
              )}
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
    </div>
  );
}
