import { createContext, useContext, useState, useCallback, useMemo } from "react";
import en from "@/locales/en.json";
import zh from "@/locales/zh.json";

const messages = { en, zh };

const LocaleContext = createContext(null);

function translate(locale, key, params) {
  const str = messages[locale]?.[key] ?? messages.en[key] ?? key;
  if (!params) return str;
  return str.replace(/\{(\w+)\}/g, (_, k) => (params[k] != null ? params[k] : `{${k}}`));
}

export function LocaleProvider({ children }) {
  const [locale, setLocaleState] = useState(() => {
    if (typeof window !== "undefined") {
      return localStorage.getItem("locale") || "zh";
    }
    return "zh";
  });

  const setLocale = useCallback((l) => {
    setLocaleState(l);
    localStorage.setItem("locale", l);
  }, []);

  const t = useCallback((key, params) => translate(locale, key, params), [locale]);

  const value = useMemo(() => ({ t, locale, setLocale }), [t, locale, setLocale]);

  return <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>;
}

export function useLocale() {
  const ctx = useContext(LocaleContext);
  if (!ctx) throw new Error("useLocale must be used within LocaleProvider");
  return ctx;
}
