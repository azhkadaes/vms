import { FieldMessage, Select } from "@vms/ui";
import type { UseFormRegisterReturn } from "react-hook-form";

export function AppointmentRoomField({
  id,
  rooms,
  registration,
  error,
}: {
  id: string;
  rooms: readonly (string | { id: string; name: string })[];
  registration: UseFormRegisterReturn<"room">;
  error?: string;
}) {
  return (
    <div className="field">
      <label htmlFor={id}>Ruangan kunjungan (opsional)</label>
      <Select
        {...registration}
        id={id}
        aria-invalid={!!error}
        aria-describedby={error ? `${id}-error` : undefined}
      >
        <option value="">Belum ditetapkan</option>
        {rooms.map((room) => {
          const { id, name } = typeof room === "string" ? { id: room, name: room } : room;
          return (
            <option key={id} value={id}>
              {name}
            </option>
          );
        })}
      </Select>
      {error && (
        <FieldMessage id={`${id}-error`} tone="error">
          {error}
        </FieldMessage>
      )}
    </div>
  );
}
