import { useEffect, useId } from "react";

const drafts = new Set<string>();
const operations = new Set<string>();

export function useUnsavedChanges(dirty: boolean, busy = false) {
  const id = useId();
  useEffect(() => {
    if (dirty) drafts.add(id);
    if (busy) operations.add(id);
    return () => {
      drafts.delete(id);
      operations.delete(id);
    };
  }, [id, dirty, busy]);
}

export function canSwitchRole() {
  if (operations.size) return false;
  return (
    !drafts.size ||
    window.confirm(
      "Buang perubahan yang belum disimpan dan ganti peran? Pilih Batal untuk tetap mengisi.",
    )
  );
}
