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
        const qrcode = await import("qrcode");
        const url = await qrcode.toDataURL(admission.payload, {
          width: 280,
          margin: 3,
          color: { dark: "#0a2342", light: "#fffdf8" },
        });
        if (controller.signal.aborted) return;
        if (deadline <= performance.now())
          throw new Error("QR tidak tersedia. Coba lagi.");
        setMedia({ key, url });
        expiresAt = deadline;
        renewIfDue();
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
    </div>
  );
}
