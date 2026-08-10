import Link from "next/link";
import { Link2 } from "lucide-react";
import StarfieldCanvas from "@/components/StarfieldCanvas";

export default function HomePage() {
  return (
    <main className="relative h-dvh w-full overflow-hidden overscroll-none bg-deep-space text-white">
      <StarfieldCanvas className="absolute inset-0" />

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
              登录
            </Link>
            <Link
              href="/register"
              className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-indigo-500"
            >
              免费注册
            </Link>
          </div>
        </header>

        <div className="flex flex-1 flex-col items-center justify-center px-4 text-center">
          <h1 className="text-6xl font-black tracking-[0.18em] text-white drop-shadow-[0_0_28px_rgba(99,102,241,0.45)] sm:text-8xl">
            KADA
          </h1>
          <p className="mt-6 text-base font-medium tracking-[0.5em] text-indigo-200/90 sm:text-lg">
            短链接平台
          </p>
        </div>
      </div>
    </main>
  );
}
