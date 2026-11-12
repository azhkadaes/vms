import { emailSchema } from "@vms/data/validation";
import { Button, EmailInput, ErrorNotice, RequiredFieldsHint, RequiredMark } from "@vms/ui";
import { localizeMessage } from "@vms/ui/language";
import { useRef, useState } from "react";
import { useLanguage } from "../i18n";

export default function SubmissionRecovery({
  email,
  onEmailChange,
  checking,
  onRecover,
  error,
  headingLevel = 2,
}: {
  email: string;
  onEmailChange: (email: string) => void;
  checking: boolean;
  onRecover: (email: string) => Promise<void>;
  error: unknown;
  headingLevel?: 2 | 3;
}) {
  const { language, t } = useLanguage();
  const [emailError, setEmailError] = useState("");
  const input = useRef<HTMLInputElement>(null);
  const Heading = headingLevel === 2 ? "h2" : "h3";
  return (
    <section
      className="panel submission-recovery section-gap"
      aria-labelledby="submission-recovery-title"
      aria-busy={checking}
    >
      <Heading id="submission-recovery-title" tabIndex={-1}>
        {t("Periksa pengiriman sebelumnya", "Check the previous submission")}
      </Heading>
      <p>
        {t(
          "Hasil pengiriman belum pasti. Periksa terlebih dahulu agar perubahan Anda tidak membuat permohonan kedua.",
          "The submission result is uncertain. Check it first so your changes do not create a second request.",
        )}
      </p>
      <form
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          if (checking) return;
          const result = emailSchema.safeParse(email);
          if (!result.success) {
            setEmailError(result.error.issues[0].message);
            input.current?.focus();
            return;
          }
          setEmailError("");
          void onRecover(result.data);
        }}
      >
        <RequiredFieldsHint />
        <div className="field">
          <label htmlFor="submission-email">
            {t("Email pengajuan sebelumnya", "Previous submission email")} <RequiredMark />
          </label>
          <EmailInput
            ref={input}
            id="submission-email"
            required
            value={email}
            disabled={checking}
            onValueChange={(value) => {
              onEmailChange(value);
              if (emailSchema.safeParse(value).success) setEmailError("");
            }}
            aria-invalid={!!emailError}
            aria-describedby={emailError ? "submission-email-error" : "submission-email-hint"}
          />
          <small id="submission-email-hint">
            {t(
              "Gunakan email saat pengiriman pertama, meskipun isian formulir sudah diubah.",
              "Use the email from the first submission, even if you have edited the form.",
            )}
          </small>
          {emailError && (
            <span id="submission-email-error" className="field-error" role="alert">
              {localizeMessage(emailError, language)}
            </span>
          )}
        </div>
        <ErrorNotice error={error} />
        <div className="form-actions">
          <Button type="submit" disabled={checking}>
            {checking
              ? t("Memeriksa pengiriman…", "Checking submission…")
              : t("Periksa hasil pengiriman", "Check submission result")}
          </Button>
        </div>
      </form>
    </section>
  );
}
