"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Link2, BarChart3, ChevronLeft } from "lucide-react";
import { cn } from "@/lib/utils";

const NAV_ITEMS = [
  {
    name: "链接",
    href: "/dashboard",
    icon: Link2,
    exact: false,
  },
  {
    name: "统计",
    href: "/dashboard/analytics",
    icon: BarChart3,
    exact: false,
  },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="flex h-full w-[240px] shrink-0 flex-col border-r border-gray-200 bg-white">
      {/* Logo */}
      <div className="flex h-14 items-center gap-2 px-4 border-b border-gray-100">
        <div className="flex size-8 items-center justify-center rounded-lg bg-indigo-600">
          <Link2 className="size-4 text-white" />
        </div>
        <span className="font-bold text-lg text-gray-900">Kada</span>
      </div>

      {/* Nav Items */}
      <nav className="flex-1 px-3 py-4 space-y-1">
        {NAV_ITEMS.map((item) => {
          const isActive = item.exact
            ? pathname === item.href
            : pathname.startsWith(item.href);
          const Icon = item.icon;

          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                isActive
                  ? "bg-indigo-50 text-indigo-700"
                  : "text-gray-600 hover:bg-gray-100 hover:text-gray-900"
              )}
            >
              <Icon className={cn("size-4", isActive && "text-indigo-600")} />
              {item.name}
            </Link>
          );
        })}
      </nav>

      {/* Bottom: no user info here - moved to top bar */}
      <div className="border-t border-gray-100 px-3 py-3">
        <p className="text-xs text-gray-400 px-2">Kada v0.2</p>
      </div>
    </aside>
  );
}
