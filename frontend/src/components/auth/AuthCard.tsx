import { Link2 } from "lucide-react";
import StarfieldCanvas from "@/components/StarfieldCanvas";

interface AuthCardProps {
  title: string;
  subtitle: string;
  footer: React.ReactNode;
  children: React.ReactNode;
}

export function AuthCard({ title, subtitle, footer, children }: AuthCardProps) {
  return (
    <div className="relative min-h-dvh overflow-x-hidden bg-deep-space">
      {/* 星空背景固定铺满，滚动时背景不动 */}
      <StarfieldCanvas className="fixed inset-0 h-full w-full" />
      <div className="relative z-10 flex min-h-dvh items-center justify-center px-4 py-12">
        <div className="w-full max-w-md">
          <div className="mb-8 flex flex-col items-center text-center">
            <div className="mb-4 flex size-11 items-center justify-center rounded-xl bg-indigo-600">
              <Link2 className="size-5 text-white" />
            </div>
            <h1 className="text-2xl font-semibold tracking-tight text-white">{title}</h1>
            <p className="mt-2 text-sm text-indigo-200/80">{subtitle}</p>
          </div>
          {/* 透明玻璃卡片 */}
          <div className="rounded-xl border border-white/10 bg-white/[0.04] p-6 backdrop-blur-lg sm:p-8">
            {children}
          </div>
          <p className="mt-6 text-center text-sm text-indigo-100/70">{footer}</p>
        </div>
      </div>
    </div>
  );
}
