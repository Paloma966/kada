import type { Metadata } from "next";
import { Toaster } from "sonner";
import { I18nProvider } from "@/lib/i18n/context";
import { ThemeProvider, ThemeScript } from "@/lib/theme";
import { LocaleHtmlLang } from "@/components/LanguageSwitcher";
import "./globals.css";

// Server-rendered defaults use English; the client provider switches the visible
// copy to the persisted preference immediately after hydration.
export const metadata: Metadata = {
  title: "Kada - Short link management platform",
  description:
    "Smart short link management and analytics platform, compatible with WeChat/QQ/Xiaohongshu/SMS",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    // suppressHydrationWarning: ThemeScript sets data-theme while the HTML is parsed, so the attribute is
    // already corrected by the time React hydrates and the server's value must not win.
    <html lang="en" suppressHydrationWarning>
      <head>
        <ThemeScript />
      </head>
      <body className="min-h-screen bg-subtle text-strong">
        <ThemeProvider>
          <I18nProvider>
            <LocaleHtmlLang />
            {children}
            <Toaster
              position="top-center"
              richColors
              closeButton
              toastOptions={{
                duration: 3000,
              }}
            />
          </I18nProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
