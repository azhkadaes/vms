import { useEffect, useState } from "react";
import { CheckCircle2, CircleAlert, Info, X } from "lucide-react";

type Tone = "success" | "error" | "info";
type Toast = { id: number; message: string; tone: Tone; description?: string };
const icons = { success: CheckCircle2, error: CircleAlert, info: Info };
const eventName = "vms:toast";
let nextId = 0;

export function toast(message: string, tone: Tone = "success", description?: string) {
  window.dispatchEvent(
    new CustomEvent<Toast>(eventName, { detail: { id: ++nextId, message, tone, description } }),
  );
}

export function Toaster() {
  const [items, setItems] = useState<Toast[]>([]);
  useEffect(() => {
    const listener = (event: Event) => {
      const item = (event as CustomEvent<Toast>).detail;
      setItems((current) => [...current.slice(-2), item]);
      window.setTimeout(() => setItems((current) => current.filter((i) => i.id !== item.id)), 5000);
    };
    window.addEventListener(eventName, listener);
    return () => window.removeEventListener(eventName, listener);
  }, []);
  return (
    <div className="toast-stack" aria-live="polite" aria-atomic="false">
      {items.map((item) => {
        const Icon = icons[item.tone];
        return (
          <div className={`toast-card toast-${item.tone}`} key={item.id} role="status">
            <Icon size={18} aria-hidden="true" />
            <span className="toast-content">
              {item.description ? (
                <>
                  <strong>{item.message}</strong>
                  <span>{item.description}</span>
                </>
              ) : (
                item.message
              )}
            </span>
            <button
              type="button"
              aria-label="Tutup notifikasi"
              onClick={() => setItems((current) => current.filter((i) => i.id !== item.id))}
            >
              <X size={16} />
            </button>
          </div>
        );
      })}
    </div>
  );
}
