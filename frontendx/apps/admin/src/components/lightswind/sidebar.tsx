import * as React from "react";

// Adapted from the component installed by `npx lightswind@latest add sidebar`.
// The back office uses scoped CSS, so the composition is retained without
// adding Tailwind or a motion runtime to the app shell.
type SidebarContextValue = {
  expanded: boolean;
  setExpanded: (value: boolean) => void;
};

const SidebarContext = React.createContext<SidebarContextValue | null>(null);

export function SidebarProvider({
  expanded,
  onExpandedChange,
  children,
}: {
  expanded: boolean;
  onExpandedChange: (value: boolean) => void;
  children: React.ReactNode;
}) {
  const value = React.useMemo(
    () => ({ expanded, setExpanded: onExpandedChange }),
    [expanded, onExpandedChange],
  );
  return <SidebarContext.Provider value={value}>{children}</SidebarContext.Provider>;
}

export function useSidebar() {
  const context = React.useContext(SidebarContext);
  if (!context) throw new Error("useSidebar must be used within SidebarProvider");
  return context;
}

export const SidebarRoot = React.forwardRef<HTMLElement, React.HTMLAttributes<HTMLElement>>(
  function SidebarRoot({ className = "", children, ...props }, ref) {
    const { expanded } = useSidebar();
    return (
      <aside
        ref={ref}
        className={`sidebar lw-sidebar ${className}`.trim()}
        data-expanded={expanded}
        {...props}
      >
        {children}
      </aside>
    );
  },
);

type SidebarDrag = {
  pointerId: number;
  startX: number;
  startWidth: number;
  width: number;
  expandedWidth: number;
  moved: boolean;
  shell: HTMLElement;
};

export function SidebarRail() {
  const { expanded, setExpanded } = useSidebar();
  const drag = React.useRef<SidebarDrag | null>(null);
  const suppressClick = React.useRef(false);
  const clearDrag = React.useCallback(() => {
    const current = drag.current;
    if (!current) return null;
    drag.current = null;
    current.shell.style.removeProperty("--sidebar-drag-width");
    delete current.shell.dataset.sidebarDragging;
    return current;
  }, []);

  React.useEffect(() => {
    const cancel = () => {
      if (clearDrag()) suppressClick.current = true;
    };
    window.addEventListener("resize", cancel);
    return () => {
      window.removeEventListener("resize", cancel);
      clearDrag();
    };
  }, [clearDrag]);

  return (
    <div
      className="lw-sidebar-rail"
      role="separator"
      tabIndex={0}
      aria-label="Ukuran navigasi"
      aria-controls="backoffice-sidebar"
      aria-orientation="vertical"
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={expanded ? 100 : 0}
      aria-valuetext={expanded ? "Navigasi diperluas" : "Navigasi diciutkan"}
      title="Geser untuk mengatur navigasi. Klik atau gunakan tombol panah untuk menciutkan atau memperluas."
      onPointerDown={(event) => {
        if (event.button !== 0 || !event.isPrimary) return;
        const shell = event.currentTarget.closest<HTMLElement>(".admin-shell");
        const sidebar = event.currentTarget.parentElement;
        if (!shell || !sidebar) return;
        event.preventDefault();
        event.currentTarget.focus({ preventScroll: true });
        event.currentTarget.setPointerCapture(event.pointerId);
        const width = sidebar.getBoundingClientRect().width;
        drag.current = {
          pointerId: event.pointerId,
          startX: event.clientX,
          startWidth: width,
          width,
          expandedWidth: parseFloat(
            getComputedStyle(shell).getPropertyValue("--sidebar-expanded-width"),
          ),
          moved: false,
          shell,
        };
        suppressClick.current = false;
      }}
      onPointerMove={(event) => {
        const current = drag.current;
        if (!current || current.pointerId !== event.pointerId) return;
        const delta = event.clientX - current.startX;
        if (!current.moved && Math.abs(delta) < 4) return;
        current.moved = true;
        current.width = Math.max(76, Math.min(current.expandedWidth, current.startWidth + delta));
        current.shell.dataset.sidebarDragging = "true";
        current.shell.style.setProperty("--sidebar-drag-width", `${current.width}px`);
      }}
      onPointerUp={(event) => {
        if (drag.current?.pointerId !== event.pointerId) return;
        const current = clearDrag();
        if (event.currentTarget.hasPointerCapture(event.pointerId))
          event.currentTarget.releasePointerCapture(event.pointerId);
        if (current?.moved) {
          suppressClick.current = true;
          setExpanded(current.width >= (76 + current.expandedWidth) / 2);
        }
      }}
      onPointerCancel={() => {
        suppressClick.current = true;
        clearDrag();
      }}
      onLostPointerCapture={() => {
        if (drag.current) {
          suppressClick.current = true;
          clearDrag();
        }
      }}
      onClick={() => {
        if (suppressClick.current) {
          suppressClick.current = false;
          return;
        }
        setExpanded(!expanded);
      }}
      onKeyDown={(event) => {
        if (event.key === "Escape" && drag.current) {
          event.preventDefault();
          suppressClick.current = true;
          clearDrag();
          return;
        }
        if (drag.current) return;
        if (!["ArrowLeft", "ArrowRight", "Home", "End", "Enter", " "].includes(event.key)) return;
        event.preventDefault();
        setExpanded(
          event.key === "ArrowLeft" || event.key === "Home"
            ? false
            : event.key === "ArrowRight" || event.key === "End"
              ? true
              : !expanded,
        );
      }}
    />
  );
}

export function SidebarHeader(props: React.HTMLAttributes<HTMLDivElement>) {
  return <div {...props} className={`lw-sidebar-header ${props.className || ""}`.trim()} />;
}

export function SidebarContent(props: React.HTMLAttributes<HTMLDivElement>) {
  return <div {...props} className={`lw-sidebar-content ${props.className || ""}`.trim()} />;
}

export function SidebarGroup(props: React.HTMLAttributes<HTMLDivElement>) {
  return <div {...props} className={`lw-sidebar-group ${props.className || ""}`.trim()} />;
}

export function SidebarGroupLabel(props: React.HTMLAttributes<HTMLDivElement>) {
  return <div {...props} className={`workspace-label ${props.className || ""}`.trim()} />;
}

export function SidebarGroupContent(props: React.HTMLAttributes<HTMLDivElement>) {
  return <div {...props} className={`lw-sidebar-group-content ${props.className || ""}`.trim()} />;
}

export function SidebarMenu(props: React.HTMLAttributes<HTMLElement>) {
  return <nav {...props} className={`lw-sidebar-menu ${props.className || ""}`.trim()} />;
}

export function SidebarMenuItem(props: React.HTMLAttributes<HTMLDivElement>) {
  return <div {...props} className={`lw-sidebar-item ${props.className || ""}`.trim()} />;
}

export function SidebarMenuButton({
  children,
  isActive,
  className = "",
}: {
  children: React.ReactElement<{ className?: string; "aria-current"?: string }>;
  isActive: boolean;
  className?: string;
}) {
  return React.cloneElement(children, {
    className:
      `lw-sidebar-link ${isActive ? "active" : ""} ${className} ${children.props.className || ""}`.trim(),
    "aria-current": isActive ? "page" : undefined,
  });
}

export function SidebarFooter(props: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      {...props}
      className={`sidebar-bottom lw-sidebar-footer ${props.className || ""}`.trim()}
    />
  );
}
