import { bookingName, dateLabel, preview, today, useBookings, visitTiming } from "@vms/data";
import {
  Button,
  QueryState,
  RequiredFieldsHint,
  RequiredMark,
  SearchableMultiSelect,
} from "@vms/ui";
import { InputOTP } from "@vms/ui/input-otp";
import { ArrowRight } from "lucide-react";
import { useState } from "react";
import { useDebouncedValue } from "../hooks/useDebouncedValue";

const pageSize = 25;

export default function ReceptionPinForm({
  mode,
  busy,
  pinRejected,
  onPinEdit,
  onResolve,
}: {
  mode: "checkin" | "checkout";
  busy: boolean;
  pinRejected: boolean;
  onPinEdit: () => void;
  onResolve: (pin: string, visitId: string) => void;
}) {
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [selected, setSelected] = useState<{ value: string; label: string }>();
  const needle = search.trim();
  const debounced = useDebouncedValue(needle, 250);
  const date = mode === "checkin" ? today() : undefined;
  const query = useBookings({
    filter: { view: "reception", mode, date, q: debounced || undefined, page, pageSize },
  });
  const visits = (query.data || []).filter(
    (visit) =>
      !preview ||
      ((mode === "checkin"
        ? visit.date === date && visitTiming(visit).displayStatus === "approved"
        : visit.status === "checked_in") &&
        bookingName(visit).toLowerCase().includes(needle.toLowerCase())),
  );
  const ready = !query.error && (preview || needle === debounced);
  const pending = query.isPending || (!preview && needle !== debounced);
  const options = ready
    ? visits.map((visit) => ({
        value: visit.id,
        label: [bookingName(visit), dateLabel(visit.date), `${visit.time} WITA`, visit.organization]
          .filter(Boolean)
          .join(" · "),
      }))
    : [];
  const visitId = !query.error && selected ? selected.value : "";

  function changePage(next: number) {
    setSelected(undefined);
    setPage(next);
  }

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        const pin = new FormData(event.currentTarget).get("code");
        if (!busy && visitId && typeof pin === "string" && /^\d{8}$/.test(pin))
          onResolve(pin, visitId);
      }}
    >
      <RequiredFieldsHint />
      <div className="field">
        <label htmlFor="booking-code">
          Kode kunjungan <RequiredMark />
        </label>
        <InputOTP
          id="booking-code"
          name="code"
          minLength={8}
          required
          readOnly={busy}
          aria-label="8 digit kode kunjungan"
          aria-invalid={pinRejected}
          aria-describedby={pinRejected ? "reception-resolve-error" : undefined}
          onValueChange={onPinEdit}
        />
      </div>
      <SearchableMultiSelect
        id="reception-visit"
        label="Nama pengunjung"
        itemLabel="pengunjung"
        placeholder="Cari nama atau referensi kunjungan…"
        multiple={false}
        required
        disabled={busy}
        options={options}
        selected={visitId ? [visitId] : []}
        onChange={([value]) => {
          setSelected(options.find((option) => option.value === value));
          onPinEdit();
        }}
        remoteSearch={{
          maxLength: 200,
          onChange: (value) => {
            setSearch(value);
            changePage(1);
          },
          selectedLabel: selected?.label,
          emptyMessage: pending
            ? "Memuat daftar pengunjung…"
            : query.error
              ? "Daftar pengunjung belum tersedia."
              : undefined,
        }}
      />
      <QueryState pending={false} error={query.error} retry={query.refetch}>
        {!preview && ready && query.total !== undefined && (page > 1 || query.total > pageSize) && (
          <nav className="form-actions" aria-label="Halaman daftar nama pengunjung">
            <Button
              type="button"
              kind="secondary"
              disabled={page === 1}
              onClick={() => changePage(page - 1)}
            >
              Sebelumnya
            </Button>
            <span>Halaman {page}</span>
            <Button
              type="button"
              kind="secondary"
              disabled={page * pageSize >= query.total}
              onClick={() => changePage(page + 1)}
            >
              Berikutnya
            </Button>
          </nav>
        )}
      </QueryState>
      <div className="form-actions">
        <Button type="submit" disabled={busy || !visitId}>
          {busy ? "Memeriksa…" : "Periksa kunjungan"}
          <ArrowRight size={16} />
        </Button>
      </div>
    </form>
  );
}
