"use client";

import { useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import { LogOut, User } from "lucide-react";
import { toast } from "sonner";
import { Sidebar } from "./Sidebar";
import { getToken, getUser, removeToken } from "@/lib/auth";
import type { User as UserType } from "@/lib/auth";

export function AppLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [ready, setReady] = useState(false);
  const [user, setUser] = useState<UserType | null>(null);
  const [showDropdown, setShowDropdown] = useState(false);

  useEffect(() => {
    const token = getToken();
    const u = getUser();
    if (!token || !u) {
      router.push("/login");
    } else {
      setUser(u);
      setReady(true);
    }
  }, [router]);

  // Close dropdown when clicking outside
  useEffect(() => {
    const handler = () => setShowDropdown(false);
    if (showDropdown) {
      document.addEventListener("click", handler);
      return () => document.removeEventListener("click", handler);
    }
  }, [showDropdown]);

  const handleLogout = () => {
    removeToken();
    toast.success("已退出");
    router.push("/login");
  };

  if (!ready) {
    return (
      <div className="flex h-screen items-center justify-center bg-gray-50">
        <div className="flex flex-col items-center gap-3">
          <div className="size-8 animate-spin rounded-full border-2 border-indigo-600 border-t-transparent" />
          <p className="text-sm text-gray-400">加载中...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-screen overflow-hidden bg-gray-50">
      {/* Left Sidebar */}
      <Sidebar />

      {/* Right: Top bar + Content */}
      <div className="flex flex-1 flex-col min-w-0">
        {/* Top Bar */}
        <header className="flex h-14 items-center justify-between border-b border-gray-200 bg-white px-6 shrink-0">
          {/* Breadcrumb / page title */}
          <div>
            <h2 className="text-sm font-medium text-gray-700">
              {(function () {
                if (pathname === "/dashboard") return "链接";
                if (pathname.startsWith("/dashboard/links/new")) return "创建链接";
                if (pathname.match(/^\/dashboard\/links\/\d+$/)) return "链接详情";
                if (pathname === "/dashboard/analytics") return "分析";
                if (pathname === "/dashboard/domains") return "域名";
                if (pathname === "/dashboard/events") return "事件";
                if (pathname === "/dashboard/customers") return "客户";
                if (pathname === "/dashboard/folders") return "文件夹";
                if (pathname === "/dashboard/tags") return "标签";
                if (pathname === "/dashboard/utm") return "UTM 模板";
                return "";
              })()}
            </h2>
          </div>

          {/* Right: User */}
          <div className="relative">
            <button
              onClick={(e) => {
                e.stopPropagation();
                setShowDropdown(!showDropdown);
              }}
              className="flex items-center gap-2 rounded-lg px-2 py-1.5 text-sm transition-colors hover:bg-gray-100"
            >
              <div className="flex size-8 items-center justify-center rounded-full bg-indigo-100 text-sm font-medium text-indigo-600">
                {(user?.name || user?.email || "U")[0].toUpperCase()}
              </div>
              <span className="hidden sm:block text-sm font-medium text-gray-700">
                {user?.name || user?.email || "用户"}
              </span>
            </button>

            {/* Dropdown */}
            {showDropdown && (
              <div
                className="absolute right-0 top-full mt-1 w-48 rounded-lg border border-gray-200 bg-white py-1 shadow-lg z-50"
                onClick={(e) => e.stopPropagation()}
              >
                <div className="px-3 py-2 border-b border-gray-100">
                  <p className="text-sm font-medium text-gray-900 truncate">
                    {user?.name || "用户"}
                  </p>
                  {user?.email && (
                    <p className="text-xs text-gray-500 truncate">{user.email}</p>
                  )}
                  {user?.phone && (
                    <p className="text-xs text-gray-500">{user.phone}</p>
                  )}
                </div>
                <button
                  onClick={handleLogout}
                  className="flex w-full items-center gap-2 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 transition-colors"
                >
                  <LogOut className="size-4" />
                  退出登录
                </button>
              </div>
            )}
          </div>
        </header>

        {/* Content */}
        <main className="flex-1 overflow-y-auto">
          <div className="mx-auto max-w-5xl px-6 py-6">
            {children}
          </div>
        </main>
      </div>
    </div>
  );
}
