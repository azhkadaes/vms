import { Button, ErrorNotice } from "@vms/ui";
import { Camera, ScanLine } from "lucide-react";
import { useEffect, useRef, useState } from "react";
export default function Scanner({ onResult }: { onResult: (value: string) => unknown }) {
  const video = useRef<HTMLVideoElement>(null);
  const stream = useRef<MediaStream | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const [error, setError] = useState<unknown>();
  const [active, setActive] = useState(false);
  const stopped = useRef(false);
  function stop() {
    stopped.current = true;
    clearTimeout(timer.current);
    stream.current?.getTracks().forEach((t) => t.stop());
    stream.current = null;
  }
  useEffect(() => () => stop(), []);
  async function start() {
    setError(null);
    const Detector = (window as any).BarcodeDetector;
    if (!Detector) {
      setError(
        "Pemindaian QR belum didukung browser ini. Gunakan pilihan kode kunjungan atau browser yang mendukung BarcodeDetector.",
      );
      return;
    }
    try {
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
      if (video.current) {
        video.current.srcObject = media;
        await video.current.play();
      }
      setActive(true);
      const detector = new Detector({ formats: ["qr_code"] });
      const tick = async () => {
        if (stopped.current || !video.current) return;
        try {
          const codes = await detector.detect(video.current);
          if (stopped.current) return;
          if (codes.length) {
            stop();
            setActive(false);
            onResult(codes[0].rawValue);
            return;
          }
        } catch {
          /* frame not ready; next frame remains cancellable */
        }
        if (!stopped.current) timer.current = setTimeout(tick, 180);
      };
      void tick();
    } catch {
      stop();
      setActive(false);
      setError("Kamera tidak dapat dibuka. Periksa izin kamera, atau gunakan kode kunjungan.");
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
        kind="secondary"
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
