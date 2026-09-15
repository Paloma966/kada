"use client";

import { useEffect } from "react";
import { useI18n, LOCALE_LABELS } from "@/lib/i18n";
import { Flag } from "@/components/Flags";

/**
 * Language switcher.
 *
 * Shows the language you would switch to, so it reads as an action rather than a state. The flag
 * identifies it at a glance and the two-letter code keeps it unambiguous: a flag names a country, not a
 * language, and English is not owned by the United Kingdom - so the code is what states the target, and
 * the accessible name spells it out in full.
 *
 * `onDark` is for the landing and sign-in screens, which are dark in both themes. There the control has to
 * stay a light chip with dark type; a themed surface would follow the page into black and disappear into
 * it, which is what happened when the sign-in screen's selected segment became a themed surface.
 */
export function LanguageSwitcher({
  className = "",
  onDark = false,
}: {
  className?: string;
  onDark?: boolean;
}) {
  const { locale, setLocale } = useI18n();
  const next = locale === "en" ? "zh" : "en";
  const label = `Switch language to ${LOCALE_LABELS[next]}`;

  const tone = onDark
    ? "border-white/20 bg-white/10 text-white hover:bg-white/20"
    : "border-line bg-canvas text-muted hover:bg-muted-surface hover:text-strong";

  return (
    <button
      type="button"
      onClick={() => setLocale(next)}
      title={label}
      aria-label={label}
      className={`inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1.5 text-sm font-medium transition-colors ${tone} ${className}`}
    >
      <Flag locale={next} />
      <span className="uppercase">{next}</span>
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
