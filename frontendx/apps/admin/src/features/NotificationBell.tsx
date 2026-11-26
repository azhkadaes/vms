import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { sessionKey, type Session } from "@vms/data";
import {
  listNotifications,
  markNotificationRead,
  type StaffNotification,
} from "@vms/data/notifications";
import { Button, Modal, QueryState, ErrorNotice } from "@vms/ui";
import { toast } from "@vms/ui/toast";
import { Bell } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";

const titles = {
  created: "Kunjungan baru",
  approved: "Kunjungan disetujui",
  updated: "Detail kunjungan diperbarui",
  rescheduled: "Jadwal kunjungan diubah",
  cancelled: "Kunjungan dibatalkan",
  checked_out: "Kunjungan selesai",
} satisfies Record<StaffNotification["event"], string>;
const dateTime = new Intl.DateTimeFormat("id-ID", {
  dateStyle: "medium",
  timeStyle: "short",
  timeZone: "Asia/Makassar",
});

export default function NotificationBell({ session }: { session: Session }) {
  const [open, setOpen] = useState(false);
  const [pages, setPages] = useState<string[]>([]);
  const trigger = useRef<HTMLButtonElement>(null);
  const seen = useRef<string[]>([]);
  const cache = useQueryClient();
  const navigate = useNavigate();
  const scope = ["staff-notifications", sessionKey(session)] as const;
  const before = pages.at(-1) || "";
  const latest = useQuery({
    queryKey: [...scope, ""],
    queryFn: ({ signal }) => listNotifications("", signal),
    refetchOnWindowFocus: true,
    refetchInterval: 30000,
    retry: false,
  });
  const older = useQuery({
    queryKey: [...scope, before],
    queryFn: ({ signal }) => listNotifications(before, signal),
    enabled: open && !!before,
    retry: false,
  });
  const current = before ? older : latest;
  const nextBefore = current.data?.nextBefore;
  const unread = latest.data?.unreadCount || 0;
  const storageKey = `vms:notification-toasts:${session.id}:${session.role}`;

  useEffect(() => {
    if (!latest.data || latest.error) return;
    let previouslySeen = seen.current;
    try {
      const saved: unknown = JSON.parse(localStorage.getItem(storageKey) || "[]");
      if (Array.isArray(saved))
        previouslySeen = previouslySeen.concat(saved.filter((value) => typeof value === "string"));
    } catch {
      /* The current tab still deduplicates toasts. */
    }
    const fresh = latest.data.items.filter(
      (item) => !item.readAt && !previouslySeen.includes(item.id),
    );
    if (!fresh.length) return;
    seen.current = [...new Set([...fresh.map((item) => item.id), ...previouslySeen])].slice(0, 100);
    try {
      localStorage.setItem(storageKey, JSON.stringify(seen.current));
    } catch {
      /* Leave the history available. */
    }
    if (fresh.length > 3) {
      toast(
        `${fresh.length} notifikasi kunjungan baru`,
        "info",
        "Buka riwayat notifikasi untuk melihat rinciannya.",
      );
    } else {
      for (const item of fresh.slice().reverse())
        toast(titles[item.event], "info", `Kunjungan #${item.visitId}`);
    }
  }, [latest.data, latest.error, storageKey]);

  const markRead = useMutation({
    mutationFn: async (item: StaffNotification) => {
      if (!item.readAt) await markNotificationRead(item.id);
    },
    onSuccess: async () => {
      await cache.invalidateQueries({ queryKey: scope });
    },
  });

  function close() {
    setOpen(false);
    setPages([]);
    markRead.reset();
  }

  return (
    <>
      <Button
        ref={trigger}
        type="button"
        kind="ghost"
        className="notification-bell"
        aria-label={unread ? `Notifikasi, ${unread} belum dibaca` : "Notifikasi"}
        aria-haspopup="dialog"
        onClick={() => {
          setOpen(true);
          void latest.refetch();
        }}
      >
        <Bell size={19} aria-hidden="true" />
        {unread > 0 && <span aria-hidden="true">{unread > 99 ? "99+" : unread}</span>}
      </Button>
      <Modal
        open={open}
        onClose={close}
        title="Notifikasi"
        className="notification-history"
        returnFocusTo={trigger.current}
      >
        <QueryState pending={current.isPending} error={current.error} retry={current.refetch}>
          {current.data?.items.length ? (
            <ul className="notification-list">
              {current.data.items.map((item) => (
                <li key={item.id}>
                  <Button
                    type="button"
                    kind="action-card"
                    className="notification-row"
                    data-unread={!item.readAt || undefined}
                    disabled={markRead.isPending}
                    onClick={async () => {
                      try {
                        await markRead.mutateAsync(item);
                        close();
                        navigate(`/booking?selected=${item.visitId}`);
                      } catch {
                        /* Keep the dialog open for retry. */
                      }
                    }}
                  >
                    <strong>{titles[item.event]}</strong>
                    <small>
                      Kunjungan #{item.visitId} · {dateTime.format(new Date(item.at))} WITA
                    </small>
                    {!item.readAt && <span className="notification-unread">Belum dibaca</span>}
                  </Button>
                </li>
              ))}
            </ul>
          ) : (
            <p className="muted">Belum ada notifikasi.</p>
          )}
          <ErrorNotice error={markRead.error} />
          {(before || nextBefore) && (
            <nav aria-label="Halaman notifikasi" className="form-actions">
              {before && (
                <Button type="button" kind="ghost" onClick={() => setPages(pages.slice(0, -1))}>
                  Lebih baru
                </Button>
              )}
              {nextBefore && (
                <Button type="button" kind="ghost" onClick={() => setPages([...pages, nextBefore])}>
                  Lebih lama
                </Button>
              )}
            </nav>
          )}
        </QueryState>
      </Modal>
    </>
  );
}
