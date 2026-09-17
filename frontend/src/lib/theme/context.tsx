"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useHydrated } from "@/lib/useHydrated";

/**
 * Theme handling.
 *
 * `data-theme` on <html> is the single source of truth, so the theme an inline script applied before the
 * first paint and the theme React knows about cannot disagree. This provider reads that attribute rather
 * than keeping a second copy of the answer, and only decides *what* it should be.
 *
 * Resolution order:
 *   1. an explicit choice, stored in localStorage;
 *   2. otherwise the browser's `prefers-color-scheme`, re-read when it changes.
 */

export type Theme = "light" | "dark";

const STORAGE_KEY = "kada.theme";
const DARK_QUERY = "(prefers-color-scheme: dark)";

type ThemeValue = {
  /** The theme currently applied. */
  theme: Theme;
  /** True while following the OS setting, false once the user has chosen explicitly. */
  followsSystem: boolean;
  setTheme: (theme: Theme) => void;
  /** Drops the stored choice and goes back to following the OS. */
  useSystemTheme: () => void;
};

const ThemeContext = createContext<ThemeValue | null>(null);

function systemTheme(): Theme {
  return window.matchMedia(DARK_QUERY).matches ? "dark" : "light";
}

/** The stored choice, or null when the user has never chosen. */
function readStoredTheme(): Theme | null {
  try {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    return stored === "dark" || stored === "light" ? stored : null;
  } catch {
    return null;
  }
}

/**
 * The theme actually in the DOM.
 *
 * ThemeScript has already set it, so reading the attribute back keeps this in step with the painted page -
 * including the "following the system" case, which this component then never has to compute at all.
 */
function appliedTheme(): Theme {
  return document.documentElement.getAttribute("data-theme") === "dark" ? "dark" : "light";
}

function applyTheme(theme: Theme) {
  document.documentElement.setAttribute("data-theme", theme);
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const hydrated = useHydrated();
  const [, forceRender] = useState(0);

  const theme: Theme = hydrated ? appliedTheme() : "light";
  const followsSystem = hydrated && readStoredTheme() === null;

  // Follow the OS while there is no explicit choice, so the app switches with the system (or with
  // devtools' emulation) instead of only on load.
  useEffect(() => {
    if (!followsSystem) return;

    const media = window.matchMedia(DARK_QUERY);
    const onChange = () => {
      applyTheme(media.matches ? "dark" : "light");
      forceRender((n) => n + 1);
    };

    media.addEventListener("change", onChange);
    return () => media.removeEventListener("change", onChange);
  }, [followsSystem]);

  const setTheme = useCallback((next: Theme) => {
    applyTheme(next);
    try {
      window.localStorage.setItem(STORAGE_KEY, next);
    } catch {
      // Storage can be unavailable (private mode); the applied attribute still holds for this session.
    }
    forceRender((n) => n + 1);
  }, []);

  const useSystemTheme = useCallback(() => {
    try {
      window.localStorage.removeItem(STORAGE_KEY);
    } catch {
      // See above.
    }
    applyTheme(systemTheme());
    forceRender((n) => n + 1);
  }, []);

  const value = useMemo<ThemeValue>(
    () => ({ theme, followsSystem, setTheme, useSystemTheme }),
    [theme, followsSystem, setTheme, useSystemTheme]
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): ThemeValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error("useTheme must be used within a ThemeProvider");
  }
  return ctx;
}

/**
 * Applies the theme before the first paint.
 *
 * This is a plain inline <script> in <head> rather than next/script. That is deliberate and measured:
 * with `strategy="beforeInteractive"` the theme landed 62ms AFTER the first frame - Next queues the
 * inline body through its own loader, which runs too late - and the result was a visible flash of the
 * light theme on every load in dark mode. A synchronous inline script blocks parsing, so the attribute
 * is set before anything is painted. `scripts/check-theme-timing.mjs` measures this and fails if the
 * theme is applied after the first frame.
 *
 * It reads the same two sources in the same order as the provider, so the two always agree; the
 * `suppressHydrationWarning` on <html> tells React to keep the attribute this sets.
 */
export function ThemeScript() {
  const script = `(function(){try{var s=localStorage.getItem(${JSON.stringify(STORAGE_KEY)});var m=window.matchMedia(${JSON.stringify(DARK_QUERY)}).matches;document.documentElement.setAttribute("data-theme",(s==="dark"||s==="light")?s:(m?"dark":"light"))}catch(e){try{document.documentElement.setAttribute("data-theme",window.matchMedia(${JSON.stringify(DARK_QUERY)}).matches?"dark":"light")}catch(e2){}}})()`;

  return <script dangerouslySetInnerHTML={{ __html: script }} />;
}
