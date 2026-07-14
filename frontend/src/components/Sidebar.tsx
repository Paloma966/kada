"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Link2,
  Globe,
  BarChart3,
  MousePointerClick,
  Users,
  Settings,
  X,
  type LucideIcon,
} from "lucide-react";
import { cn } from "@/lib/utils";

interface NavItem {
  name: string;
  href: string;
  icon: LucideIcon;
  isActive?: (pathname: string, href: string) => boolean;
}

interface NavSection {
  name?: string;
  items: NavItem[];
}

const NAV_SECTIONS: NavSection[] = [
  {
    items: [
      { name: "链接", href: "/dashboard", icon: Link2,
        isActive: (pathname: string) =>
          pathname === "/dashboard" || pathname.startsWith("/dashboard/links"), },
      { name: "域名", href: "/dashboard/domains", icon: Globe },
    ],
  },
  {
    name: "洞察",
    items: [
      { name: "分析", href: "/dashboard/analytics", icon: BarChart3 },
      { name: "事件", href: "/dashboard/events", icon: MousePointerClick },
      { name: "客户", href: "/dashboard/customers", icon: Users },
    ],
  },
];

export function Sidebar({ onCloseMobile }: { onCloseMobile?: () => void }) {
  const pathname = usePathname();

  const handleClick = () => {
    // Close mobile sidebar after navigation
    setTimeout(() => onCloseMobile?.(), 100);
  };

  return (
    <aside className="flex h-full w-[240px] shrink-0 flex-col border-r border-gray-200 bg-white">
      {/* Logo */}
      <div className="flex h-14 items-center justify-between px-4 border-b border-gray-100">
        <div className="flex items-center gap-2.5">
          <div className="flex size-8 items-center justify-center rounded-lg bg-indigo-600">
            <Link2 className="size-4 text-white" />
          </div>
          <span className="font-bold text-lg text-gray-900">Kada</span>
        </div>
        {onCloseMobile && (
          <button onClick={onCloseMobile} className="lg:hidden p-1.5 rounded-lg text-gray-400 hover:bg-gray-100 transition">
            <X className="size-4" />
          </button>
        )}
      </div>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto px-3 py-4">
        {NAV_SECTIONS.map((section, secIdx) => (
          <div key={secIdx} className={cn(secIdx > 0 && "mt-6")}>
            {section.name && (
              <p className="mb-1.5 px-3 text-xs font-medium text-gray-400 uppercase tracking-wider">
                {section.name}
              </p>
            )}
            <div className="space-y-0.5">
              {section.items.map((item) => {
                const isActive = item.isActive
                  ? item.isActive(pathname, item.href)
                  : pathname.startsWith(item.href);
                const Icon = item.icon;

                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    onClick={handleClick}
                    className={cn(
                      "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                      isActive
                        ? "bg-indigo-50 text-indigo-700"
                        : "text-gray-600 hover:bg-gray-100 hover:text-gray-900"
                    )}
                  >
                    <Icon className={cn("size-4 shrink-0", isActive && "text-indigo-600")} />
                    <span>{item.name}</span>
                  </Link>
                );
              })}
            </div>
          </div>
        ))}
      </nav>

      {/* Bottom: Settings + version */}
      <div className="border-t border-gray-100 px-3 py-3 space-y-2">
        <Link
          href="/dashboard/settings"
          onClick={handleClick}
          className={cn(
            "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
            pathname === "/dashboard/settings"
              ? "bg-indigo-50 text-indigo-700"
              : "text-gray-600 hover:bg-gray-100 hover:text-gray-900"
          )}
        >
          <Settings className={cn("size-4 shrink-0", pathname === "/dashboard/settings" && "text-indigo-600")} />
          <span>设置</span>
        </Link>
        <p className="px-3 text-xs text-gray-400">Kada v0.2</p>
      </div>
    </aside>
  );
}
