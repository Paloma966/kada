"use client";

import { useEffect } from "react";
import { Languages } from "lucide-react";
import { useI18n, LOCALE_LABELS } from "@/lib/i18n";

/**
 * Language switcher.
 *
 * Shows the language you would switch to, so it reads as an action rather than a
 * state. Each target language is named in its own language ("中文" / "English"),
 * which is the convention for language pickers.
 */
export function LanguageSwitcher({ className = "" }: { className?: string }) {
  const { locale, setLocale } = useI18n();
  const next = locale === "en" ? "zh" : "en";
  const label = `Switch language to ${LOCALE_LABELS[next]}`;

  return (
    <button
      type="button"
      onClick={() => setLocale(next)}
      title={label}
      aria-label={label}
      className={`inline-flex items-center gap-1.5 rounded-lg border border-gray-200 bg-white px-2.5 py-1.5 text-sm font-medium text-gray-600 transition-colors hover:bg-gray-100 hover:text-gray-900 ${className}`}
    >
      <Languages className="size-4" />
      <span>{LOCALE_LABELS[next]}</span>
    </button>
  );
}

/**
 * Keeps <html lang> in sync with the active locale. The root layout renders a
 * server-side default which this corrects once the client knows the preference.
 */
export function LocaleHtmlLang() {
  const { locale } = useI18n();

  useEffect(() => {
    document.documentElement.lang = locale === "en" ? "en" : "zh-CN";
  }, [locale]);

  return null;
}
