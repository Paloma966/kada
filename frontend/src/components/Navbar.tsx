"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { Link2, BarChart3, LogOut } from "lucide-react";
import { getUser, removeToken } from "@/lib/auth";

export default function Navbar() {
  const router = useRouter();
  const user = getUser();

  const handleLogout = () => {
    removeToken();
    router.push("/login");
  };

  return (
    <nav className="sticky top-0 z-50 bg-white border-b border-gray-200">
      <div className="max-w-7xl mx-auto px-4 h-14 flex items-center justify-between">
        <Link href="/dashboard" className="flex items-center gap-2 font-bold text-lg text-indigo-600">
          <Link2 className="w-5 h-5" />
          Kada
        </Link>

        <div className="flex items-center gap-4">
          <Link
            href="/dashboard"
            className="flex items-center gap-1 text-sm text-gray-600 hover:text-indigo-600 transition"
          >
            <Link2 className="w-4 h-4" /> 链接
          </Link>
          <span className="text-sm text-gray-400">
            {user?.name || user?.email || "用户"}
          </span>
          <button
            onClick={handleLogout}
            className="flex items-center gap-1 text-sm text-gray-400 hover:text-red-500 transition"
          >
            <LogOut className="w-4 h-4" /> 退出
          </button>
        </div>
      </div>
    </nav>
  );
}
