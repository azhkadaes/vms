import { Button, ErrorNotice } from "@vms/ui";
import jsQR from "jsqr";
import { Camera, CameraOff, ScanLine } from "lucide-react";
import { useEffect, useRef, useState } from "react";

type State = "idle" | "starting" | "active";

function cameraError(err: unknown) {
  // Browsers only expose the camera on HTTPS or localhost; a LAN http:// address has no mediaDevices.
  if (!window.isSecureContext)
    return "Kamera hanya bisa dibuka lewat alamat HTTPS. Buka backoffice melalui https, atau gunakan kode kunjungan.";
  const name = err instanceof DOMException ? err.name : "";
  if (name === "NotFoundError" || name === "OverconstrainedError" || !navigator.mediaDevices)
    return "Tidak ada kamera yang terdeteksi. Sambungkan kamera, atau gunakan kode kunjungan.";
  if (name === "NotAllowedError" || name === "SecurityError")
    return "Izin kamera ditolak. Izinkan kamera di pengaturan browser, atau gunakan kode kunjungan.";
  if (name === "NotReadableError")
    return "Kamera sedang dipakai aplikasi lain. Tutup aplikasi tersebut, atau gunakan kode kunjungan.";
  return "Kamera tidak dapat dibuka. Periksa izin kamera, atau gunakan kode kunjungan.";
}

export default function Scanner({ onResult }: { onResult: (value: string) => unknown }) {
  const video = useRef<HTMLVideoElement>(null);
  const stream = useRef<MediaStream | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  // Each start gets a run number; stop() bumps it so late camera grants from an old run are dropped.
  const run = useRef(0);
  const [error, setError] = useState<unknown>();
  const [state, setState] = useState<State>("idle");

  function stop() {
    run.current += 1;
    clearTimeout(timer.current);
    stream.current?.getTracks().forEach((t) => t.stop());
    stream.current = null;
    if (video.current) video.current.srcObject = null;
  }

  async function start() {
    stop();
    const current = run.current;
    setError(null);
    setState("starting");
    try {
      const element = video.current;
      const canvas = document.createElement("canvas");
      const context = canvas.getContext("2d", { willReadFrequently: true });
      if (!element || !context || !navigator.mediaDevices?.getUserMedia)
        throw new Error("Kamera tidak tersedia.");
      const media = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: "environment", width: { ideal: 1280 }, height: { ideal: 1280 } },
        audio: false,
      });
      if (current !== run.current) {
        media.getTracks().forEach((t) => t.stop());
        return;
      }
      stream.current = media;
      element.srcObject = media;
      await element.play();
      if (current !== run.current) return;
      setState("active");
      const tick = () => {
        if (current !== run.current) return;
        if (element.readyState >= HTMLMediaElement.HAVE_CURRENT_DATA) {
          const { videoWidth, videoHeight } = element;
          const scale = Math.min(1, 960 / Math.max(videoWidth, videoHeight));
          const width = Math.max(1, Math.round(videoWidth * scale));
          const height = Math.max(1, Math.round(videoHeight * scale));
          if (canvas.width !== width) canvas.width = width;
          if (canvas.height !== height) canvas.height = height;
          context.drawImage(element, 0, 0, width, height);
          const frame = context.getImageData(0, 0, width, height);
          const result = jsQR(frame.data, width, height, { inversionAttempts: "dontInvert" });
          if (result?.data) {
            stop();
            setState("idle");
            onResult(result.data);
            return;
          }
        }
        timer.current = setTimeout(tick, 180);
      };
      tick();
    } catch (err) {
      if (current !== run.current) return;
      stop();
      setState("idle");
      setError(cameraError(err));
    }
  }

  useEffect(() => {
    void start();
    return () => stop();
  }, []);

  const active = state === "active";
  return (
    <div className="scanner">
      <div className={`camera-frame${active ? " is-active" : ""}`}>
        <video ref={video} playsInline muted aria-label="Tampilan kamera pemindai QR" />
        {active ? (
          <span className="camera-guide" aria-hidden="true" />
        ) : (
          <div className="camera-placeholder">
            <ScanLine size={40} aria-hidden="true" />
            {state === "starting" ? (
              <span role="status">Membuka kamera…</span>
            ) : (
              <Button type="button" kind="secondary" onClick={() => void start()}>
                <Camera size={17} aria-hidden="true" />
                Aktifkan kamera
              </Button>
            )}
          </div>
        )}
      </div>
      <ErrorNotice error={error} />
      <p className="scanner-hint" role={active ? "status" : undefined}>
        {active
          ? "Arahkan kode QR ke dalam bingkai. Pemindaian berjalan otomatis."
          : "Kamera hanya aktif selama pemindaian. Gambar tidak disimpan."}
      </p>
      {active && (
        <Button
          type="button"
          kind="ghost"
          className="scanner-stop"
          onClick={() => {
            stop();
            setState("idle");
          }}
        >
          <CameraOff size={16} aria-hidden="true" />
          Hentikan kamera
        </Button>
      )}
    </div>
  );
}
