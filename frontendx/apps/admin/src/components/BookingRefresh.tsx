import { useEffect, useSyncExternalStore } from "react";
import { onlineManager, useQueryClient } from "@tanstack/react-query";
import { ApiError, preview, sessionKey, useBookings, useSession } from "@vms/data";
import DataRefresh from "./DataRefresh";

export default function BookingRefresh({ mode }: { mode: "staff" | "reception" }) {
  return preview ? <PreviewRefresh mode={mode} /> : <ProductionRefresh mode={mode} />;
}
function ProductionRefresh({ mode }: { mode: "staff" | "reception" }) {
  const client = useQueryClient();
  const { session } = useSession();
  const key = sessionKey(session);
  const cache = client.getQueryCache();
  const queries = () => cache.findAll({ queryKey: ["bookings", key], type: "active" });
  const snapshot = useSyncExternalStore(
    (notify) => cache.subscribe(notify),
    () =>
      JSON.stringify(
        queries().map((query) => ({
          updated: query.state.dataUpdatedAt,
          data: query.state.data !== undefined,
          status: query.state.status,
          fetching: query.state.fetchStatus === "fetching",
          denied:
            query.state.error instanceof ApiError && [401, 403].includes(query.state.error.status),
        })),
      ),
    () => "[]",
  );
  const states: {
    updated: number;
    data: boolean;
    status: string;
    fetching: boolean;
    denied: boolean;
  }[] = JSON.parse(snapshot);
  async function refetch() {
    await client.refetchQueries({ queryKey: ["bookings", key], type: "active" });
    return { isSuccess: queries().every((query) => query.state.status === "success") };
  }
  useEffect(() => {
    const timer = setInterval(
      () => {
        if (document.visibilityState === "visible" && onlineManager.isOnline())
          void client.refetchQueries({ queryKey: ["bookings", key], type: "active" });
      },
      mode === "reception" ? 15000 : 30000,
    );
    return () => clearInterval(timer);
  }, [client, key, mode]);
  return (
    <DataRefresh
      q={{
        data: states.some((state) => state.data) ? true : undefined,
        dataUpdatedAt: states.length ? Math.min(...states.map((state) => state.updated)) : 0,
        isPending: !states.length || states.some((state) => state.status === "pending"),
        isError: states.some((state) => state.status === "error"),
        isFetching: states.some((state) => state.fetching),
        error: states.some((state) => state.denied) ? new ApiError(403, "Akses berubah.") : null,
        refetch,
      }}
    />
  );
}
function PreviewRefresh({ mode }: { mode: "staff" | "reception" }) {
  const q = useBookings({ refresh: mode });
  return <DataRefresh q={q} />;
}
