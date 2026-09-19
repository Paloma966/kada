"use client";

import { useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import { LogOut, Menu } from "lucide-react";
import { toast } from "sonner";
import { Sidebar } from "./Sidebar";
import { LanguageSwitcher } from "./LanguageSwitcher";
import { getToken, getUser, removeToken } from "@/lib/auth";
import { useT } from "@/lib/i18n";
import { useHydrated } from "@/lib/useHydrated";
import type { User as UserType } from "@/lib/auth";

export function AppLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const t = useT();
  // The token lives in localStorage, which does not exist while the server renders. Reading it directly
  // would make the server's output (the spinner) disagree with the client's first render (the shell), and
  // React would throw the tree away as a hydration mismatch. Gating on `hydrated` keeps the two in step;
  // `user` is read through the same gate, since it comes from localStorage too.
  const hydrated = useHydrated();
  const user: UserType | null = hydrated ? getUser() : null;
  const [showDropdown, setShowDropdown] = useState(false);
  const [sidebarOpen, setSidebarOpen] = useState(false);

  const ready = hydrated && !!getToken() && !!user;

  useEffect(() => {
    if (hydrated && !ready) {
      router.push("/login");
    }
  }, [hydrated, ready, router]);

  // Close the mobile sidebar when the route changes (React's documented
  // "adjust state when a prop changes" pattern).
  const [prevPath, setPrevPath] = useState(pathname);
  if (prevPath !== pathname) {
    setPrevPath(pathname);
    setSidebarOpen(false);
  }

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
    toast.success(t("已退出"));
    router.push("/login");
  };

  if (!ready) {
    return (
      <div className="flex h-screen items-center justify-center bg-gray-50">
        <div className="flex flex-col items-center gap-3">
          <div className="size-8 animate-spin rounded-full border-2 border-indigo-600 border-t-transparent" />
          <p className="text-sm text-faint">{t("加载中...")}</p>
        </div>
      </div>
    );
  }

  const pageTitle = (() => {
    if (pathname === "/dashboard") return t("链接");
    if (pathname.startsWith("/dashboard/links/new")) return t("创建链接");
    if (pathname.match(/^\/dashboard\/links\/\d+$/)) return t("链接详情");
    if (pathname === "/dashboard/analytics") return t("分析");
    if (pathname === "/dashboard/domains") return t("域名");
    if (pathname === "/dashboard/ai") return t("AI 助手");
    if (pathname === "/dashboard/events") return t("事件");
    if (pathname === "/dashboard/customers") return t("客户");
    if (pathname === "/dashboard/folders") return t("文件夹");
    if (pathname === "/dashboard/tags") return t("标签");
    if (pathname === "/dashboard/utm") return t("UTM 模板");
    if (pathname === "/dashboard/settings") return t("设置");
    return "";
  })();

  return (
    <div className="flex h-screen overflow-hidden bg-gray-50">
      {/* Desktop Sidebar - always visible on lg+ */}
      <div className="hidden lg:block">
        <Sidebar />
      </div>

      {/* Mobile Sidebar Overlay */}
      {sidebarOpen && (
        <div className="fixed inset-0 z-40 lg:hidden">
          {/* Backdrop */}
          <div
            className="absolute inset-0 bg-black/40 backdrop-blur-sm"
            onClick={() => setSidebarOpen(false)}
          />
          {/* Sidebar */}
          <div className="absolute left-0 top-0 h-full w-[240px] z-50 shadow-xl">
            <Sidebar onCloseMobile={() => setSidebarOpen(false)} />
          </div>
        </div>
      )}

      {/* Right: Top bar + Content */}
      <div className="flex flex-1 flex-col min-w-0">
        {/* Top Bar */}
        <header className="flex h-14 items-center justify-between border-b border-line bg-canvas px-4 sm:px-6 shrink-0">
          <div className="flex items-center gap-3">
            {/* Mobile menu button */}
            <button
              onClick={() => setSidebarOpen(true)}
              className="lg:hidden p-1.5 -ml-1 rounded-lg text-muted hover:bg-muted-surface transition"
            >
              <Menu className="size-5" />
            </button>
            <h2 className="text-sm font-medium text-body">{pageTitle}</h2>
          </div>

          {/* Right: language switcher + user */}
          <div className="flex items-center gap-2">
            <LanguageSwitcher />
            <div className="relative">
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  setShowDropdown(!showDropdown);
                }}
                className="flex items-center gap-2 rounded-lg px-2 py-1.5 text-sm transition-colors hover:bg-muted-surface"
              >
                <div className="flex size-8 items-center justify-center rounded-full bg-indigo-100 text-sm font-medium text-indigo-600 shrink-0">
                  {(user?.name || user?.email || "U")[0].toUpperCase()}
                </div>
                <span className="hidden sm:block text-sm font-medium text-body max-w-[120px] truncate">
                  {user?.name || user?.email || t("用户")}
                </span>
              </button>

              {/* Dropdown */}
              {showDropdown && (
                <div
                  className="absolute right-0 top-full mt-1 w-48 rounded-lg border border-line bg-canvas py-1 shadow-lg z-50"
                  onClick={(e) => e.stopPropagation()}
                >
                  <div className="px-3 py-2 border-b border-line">
                    <p className="text-sm font-medium text-strong truncate">
                      {user?.name || t("用户")}
                    </p>
                    {user?.email && (
                      <p className="text-xs text-muted truncate">{user.email}</p>
                    )}
                    {user?.phone && (
                      <p className="text-xs text-muted">{user.phone}</p>
                    )}
                  </div>
                  <button
                    onClick={handleLogout}
                    className="flex w-full items-center gap-2 px-3 py-2 text-sm text-muted hover:bg-gray-50 transition-colors"
                  >
                    <LogOut className="size-4" />
                    {t("退出登录")}
                  </button>
                </div>
              )}
            </div>
          </div>
        </header>

        {/* Content */}
        <main className="flex-1 overflow-y-auto">
          {/* 所有页面统一布局：居中 + 边距（AI 页也遵循，不再贴满） */}
          <div className="mx-auto max-w-5xl px-4 sm:px-6 py-4 sm:py-6">
            {children}
          </div>
        </main>
      </div>
    </div>
  );
}
