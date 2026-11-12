import { Button, ErrorNotice } from "./index";
import { useUiLanguage } from "./language";

export function ReferenceDataNotice({
  loading,
  error,
  onRetry,
}: {
  loading: boolean;
  error: unknown;
  onRetry: () => void;
}) {
  const { t } = useUiLanguage();
  if (loading)
    return (
      <p role="status" className="hint">
        {t("Memuat tujuan dan ruangan…", "Loading destinations and rooms…")}
      </p>
    );
  if (error)
    return (
      <div className="section-gap">
        <ErrorNotice error={error} />
        <Button type="button" kind="secondary" onClick={onRetry}>
          {t("Coba lagi", "Try again")}
        </Button>
      </div>
    );
  return null;
}
