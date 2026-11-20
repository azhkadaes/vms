import { Button, ErrorNotice } from "@vms/ui";
import jsQR from "jsqr";
import { Camera, ScanLine } from "lucide-react";
import { useEffect, useRef, useState } from "react";
export default function Scanner({ onResult }: { onResult: (value: string) => unknown }) {
  const video = useRef<HTMLVideoElement>(null);
  const stream = useRef<MediaStream | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const [error, setError] = useState<unknown>();
  const [active, setActive] = useState(false);
  const [starting, setStarting] = useState(false);
  const stopped = useRef(false);
  function stop() {
    stopped.current = true;
    clearTimeout(timer.current);
    stream.current?.getTracks().forEach((t) => t.stop());
    stream.current = null;
    if (video.current) video.current.srcObject = null;
  }
  useEffect(() => () => stop(), []);
  async function start() {
    setError(null);
    setStarting(true);
    try {
      const element = video.current;
      const canvas = document.createElement("canvas");
      const context = canvas.getContext("2d", { willReadFrequently: true });
      if (!element || !context) throw new Error("Kamera tidak tersedia.");
      stopped.current = false;
      const media = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: "environment" },
        audio: false,
      });
      if (stopped.current) {
        media.getTracks().forEach((t) => t.stop());
        return;
      }
      stream.current = media;
      element.srcObject = media;
      await element.play();
      if (stopped.current) return;
      setActive(true);
      const tick = () => {
        if (stopped.current) return;
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
            setActive(false);
            onResult(result.data);
            return;
          }
        }
        timer.current = setTimeout(tick, 180);
      };
      tick();
    } catch {
      stop();
      setActive(false);
      setError("Kamera tidak dapat dibuka. Periksa izin kamera, atau gunakan kode kunjungan.");
    } finally {
      setStarting(false);
    }
  }
  return (
    <div className="scanner">
      <div className="camera-frame">
        <video ref={video} playsInline muted aria-label="Tampilan kamera pemindai QR" />
        {!active && <ScanLine size={52} />}
      </div>
      <ErrorNotice error={error} />
      <Button
        type="button"
        kind="secondary"
        disabled={starting}
        onClick={
          active
            ? () => {
                stop();
                setActive(false);
              }
            : start
        }
      >
        <Camera size={17} />
        {active ? "Hentikan kamera" : "Aktifkan kamera"}
      </Button>
      <p>Kamera hanya aktif selama pemindaian. Gambar tidak disimpan.</p>
    </div>
  );
}
