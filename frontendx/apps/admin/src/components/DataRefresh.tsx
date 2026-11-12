import { useState, useSyncExternalStore } from "react";
import { onlineManager } from "@tanstack/react-query";
import { RefreshCw } from "lucide-react";
import { ApiError } from "@vms/data";
import { Button } from "@vms/ui";

const subscribeOnline = (notify: () => void) => onlineManager.subscribe(notify);
const isOnline = () => onlineManager.isOnline();
const updateTime = new Intl.DateTimeFormat("id-ID", {
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
  timeZone: "Asia/Makassar",
});

export default function DataRefresh({
  q,
  label = "data kunjungan",
}: {
  label?: string;
  q: {
    data: unknown;
    dataUpdatedAt: number;
    isPending: boolean;
    isError: boolean;
    isFetching: boolean;
    error: unknown;
    refetch: (options?: { cancelRefetch: boolean }) => Promise<{ isSuccess: boolean }>;
  };
}) {
  const online = useSyncExternalStore(subscribeOnline, isOnline, () => true);
  const [announcement, setAnnouncement] = useState("");
  const hasData = q.data !== undefined;
  const accessError = q.error instanceof ApiError && [401, 403].includes(q.error.status);
  const state = !online
    ? "offline"
    : accessError
      ? "access-error"
      : q.isError
        ? "stale"
        : "current";
  const notice = !online
    ? hasData
      ? "Tidak tersambung. Menampilkan data terakhir; pembaruan dilanjutkan saat tersambung."
      : "Tidak tersambung. Data dimuat saat koneksi kembali."
    : accessError
      ? "Akses data berubah. Muat ulang atau masuk kembali."
      : q.isError
        ? hasData
          ? "Pembaruan gagal. Menampilkan data terakhir. Coba perbarui lagi."
          : "Data belum dapat dimuat. Coba perbarui lagi."
        : "";

  return (
    <section className="bookings-refresh" aria-label={`Pembaruan ${label}`} data-freshness={state}>
      <div className="bookings-refresh-copy">
        {q.dataUpdatedAt > 0 ? (
          <span>
            Terakhir diperbarui{" "}
            <time dateTime={new Date(q.dataUpdatedAt).toISOString()}>
              {updateTime.format(q.dataUpdatedAt)} WITA
            </time>
          </span>
        ) : (
          <span>
            {q.isPending && online ? `Memuat ${label}…` : "Belum ada pembaruan berhasil."}
          </span>
        )}
        {notice && <span className="bookings-refresh-notice">{notice}</span>}
        <span role="status" aria-atomic="true" className="visually-hidden">
          {notice || announcement}
        </span>
      </div>
      <Button
        type="button"
        kind="ghost"
        disabled={q.isFetching || !online}
        aria-label={q.isFetching ? `Memperbarui ${label}` : `Perbarui ${label}`}
        onClick={async () => {
          setAnnouncement("");
          const result = await q.refetch({ cancelRefetch: false });
          if (result.isSuccess)
            setAnnouncement(`${label[0].toUpperCase() + label.slice(1)} diperbarui.`);
        }}
      >
        <RefreshCw size={16} aria-hidden="true" />
        {q.isFetching ? "Memperbarui…" : "Perbarui"}
      </Button>
    </section>
  );
}
