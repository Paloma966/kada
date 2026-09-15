"use client";

import Link from "next/link";
import { Link2 } from "lucide-react";
import StarfieldCanvas from "@/components/StarfieldCanvas";
import { LanguageSwitcher } from "@/components/LanguageSwitcher";
import { useT } from "@/lib/i18n";

export default function HomePage() {
  const t = useT();

  return (
    <main className="relative h-dvh w-full overflow-hidden overscroll-none bg-deep-space text-white">
      {/* canvas is a replaced element, so inset-0 does not stretch it; h-full w-full is required to fill the viewport */}
      <StarfieldCanvas className="absolute inset-0 h-full w-full" />

      <div className="absolute inset-0 z-10 flex flex-col">
        <header className="flex items-center justify-between px-6 py-5 sm:px-8">
          <div className="flex items-center gap-2.5">
            <div className="flex size-8 items-center justify-center rounded-lg bg-indigo-600">
              <Link2 className="size-4 text-white" />
            </div>
            <span className="text-lg font-semibold tracking-wider">KADA</span>
          </div>
          <div className="flex items-center gap-2">
            <Link
              href="/login"
              className="rounded-lg px-4 py-2 text-sm font-medium text-indigo-100 transition hover:bg-white/10 hover:text-white"
            >
              {t("登录")}
            </Link>
            <Link
              href="/register"
              className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-indigo-500"
            >
              {t("免费注册")}
            </Link>
            <LanguageSwitcher onDark />
          </div>
        </header>

        <div className="flex flex-1 flex-col items-center justify-center px-4 text-center">
          <h1 className="text-6xl font-black tracking-[0.18em] text-white drop-shadow-[0_0_28px_rgba(99,102,241,0.45)] sm:text-8xl">
            KADA
          </h1>
          <p className="mt-6 text-base font-medium tracking-[0.5em] text-indigo-200/90 sm:text-lg">
            {t("短链接平台")}
          </p>
        </div>
      </div>
    </main>
  );
}
