import { ApiError, preview, sessionKey, type Session } from "@vms/data";
import { management } from "@vms/data/management";
import { Button, ErrorNotice, Loading } from "@vms/ui";
import { Popover, PopoverTrigger, PopoverContent, PopoverClose } from "@vms/ui/popover";
import { X } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import EmployeeDetails from "./EmployeeDetails";

export default function ProfilePreview({ session }: { session: Session }) {
  const [open, setOpen] = useState(false);
  const trigger = useRef<HTMLButtonElement>(null);
  const content = useRef<HTMLDivElement>(null);
  const closing = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const restoringFocus = useRef(false);
  function cancelClose() { clearTimeout(closing.current); }
  function show() { cancelClose(); setOpen(true); }
  function scheduleClose() {
    cancelClose();
    closing.current = setTimeout(() => {
      const focused = document.activeElement;
      if (!trigger.current?.contains(focused) && !content.current?.contains(focused)) setOpen(false);
    }, 180);
  }
  useEffect(() => () => clearTimeout(closing.current), []);
  const id = session.id;
  const profile = useQuery({
    queryKey: ["staff-profile", sessionKey(session)],
    queryFn: async () => {
      if (!id) throw new ApiError(401, "Sesi berakhir. Silakan masuk kembali.");
      const employee = await management.employee(id);
      if (employee.id !== id) throw new ApiError(502, "Profil pegawai tidak dapat ditampilkan.");
      return employee;
    },
    enabled: open && !preview,
    retry: false,
    refetchOnWindowFocus: false,
  });
  const initials = session.name
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((name) => name[0])
    .join("")
    .toUpperCase();

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
      <Button
        ref={trigger}
        type="button"
        kind="ghost"
        size="icon"
        aria-label="Buka profil saya"
        aria-haspopup="dialog"
        aria-expanded={open}
        onPointerEnter={(event) => { if (event.pointerType !== "touch") show(); }}
        onPointerLeave={scheduleClose}
        onFocus={() => { if (!restoringFocus.current) show(); }}
        onBlur={scheduleClose}
        onClick={(event) => { event.preventDefault(); show(); }}
      >
        <span className="avatar" aria-hidden="true">
          {initials}
        </span>
      </Button>
      </PopoverTrigger>
      <PopoverContent
        ref={content}
        align="end"
        sideOffset={8}
        className="profile-hover-card"
        aria-label="Profil saya"
        onPointerEnter={cancelClose}
        onPointerLeave={scheduleClose}
        onBlur={scheduleClose}
        onOpenAutoFocus={(event) => event.preventDefault()}
        onCloseAutoFocus={(event) => {
          event.preventDefault();
          if (content.current?.contains(document.activeElement)) {
            restoringFocus.current = true;
            trigger.current?.focus();
            restoringFocus.current = false;
          }
        }}
      >
        <div className="profile-hover-heading">
          <h2>Profil saya</h2>
          <PopoverClose asChild>
            <Button kind="ghost" size="icon" aria-label="Tutup profil"><X size={16} /></Button>
          </PopoverClose>
        </div>
        {preview ? (
          <EmployeeDetails
            employee={{
              id: "preview",
              name: session.name,
              email: "",
              role: session.role,
              active: true,
              assignedOfficerIds: [],
              version: 1,
            }}
            showAssignments={false}
          />
        ) : profile.isError ? (
          <div className="form-stack">
            <ErrorNotice error={profile.error} />
            <Button type="button" kind="secondary" onClick={() => void profile.refetch()}>
              Coba lagi
            </Button>
          </div>
        ) : profile.data ? (
          <EmployeeDetails employee={profile.data} showAssignments={false} />
        ) : (
          <Loading />
        )}
      </PopoverContent>
    </Popover>
  );
}
