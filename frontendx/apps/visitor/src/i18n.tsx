import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { UiLanguageProvider } from "@vms/ui/language";
import { SegmentedControl } from "@vms/ui/segmented-control";

export type Language = "id" | "en";

const storageKey = "vms-visitor-language";

const LanguageContext = createContext<{
  language: Language;
  setLanguage: (language: Language) => void;
}>({ language: "id", setLanguage: () => {} });

export function LanguageProvider({ children }: { children: ReactNode }) {
  const [language, setLanguage] = useState<Language>(() => {
    try {
      return localStorage.getItem(storageKey) === "en" ? "en" : "id";
    } catch {
      return "id";
    }
  });

  useEffect(() => {
    document.documentElement.lang = language;
    try {
      localStorage.setItem(storageKey, language);
    } catch {
      /* Storage may be unavailable. */
    }
  }, [language]);

  return (
    <LanguageContext.Provider value={{ language, setLanguage }}>
      <UiLanguageProvider language={language}>{children}</UiLanguageProvider>
    </LanguageContext.Provider>
  );
}

export function useLanguage() {
  const { language, setLanguage } = useContext(LanguageContext);
  return {
    language,
    setLanguage,
    t: (indonesian: string, english: string) => (language === "en" ? english : indonesian),
  };
}

const languageOptions = [
  { value: "id", label: "ID", accessibleLabel: "ID · Bahasa Indonesia", lang: "id" },
  { value: "en", label: "EN", accessibleLabel: "EN · English", lang: "en" },
] as const;

export function LanguageSwitch({ className }: { className?: string }) {
  const { language, setLanguage, t } = useLanguage();
  return (
    <SegmentedControl
      className={className}
      label={t("Bahasa halaman", "Page language")}
      options={languageOptions}
      value={language}
      onChange={setLanguage}
    />
  );
}
