import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { UiLanguageProvider } from "@vms/ui/language";

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

export function LanguageSwitch({
  className = "",
  compact = false,
}: {
  className?: string;
  compact?: boolean;
}) {
  const { language, setLanguage } = useLanguage();
  if (compact)
    return (
      <select
        className={`language-select ${className}`}
        aria-label={language === "id" ? "Bahasa halaman" : "Page language"}
        value={language}
        onChange={(event) => setLanguage(event.target.value === "en" ? "en" : "id")}
      >
        <option value="id" lang="id" label="ID">
          Bahasa Indonesia
        </option>
        <option value="en" lang="en" label="EN">
          English
        </option>
      </select>
    );
  return (
    <div
      className={`language-switch ${className}`}
      role="group"
      aria-label={language === "id" ? "Bahasa halaman" : "Page language"}
    >
      <button
        type="button"
        lang="id"
        aria-label="ID · Bahasa Indonesia"
        aria-pressed={language === "id"}
        onClick={() => setLanguage("id")}
      >
        ID
      </button>
      <span aria-hidden="true">|</span>
      <button
        type="button"
        lang="en"
        aria-label="EN · English"
        aria-pressed={language === "en"}
        onClick={() => setLanguage("en")}
      >
        EN
      </button>
    </div>
  );
}
