"use client";

import { useState } from "react";
import Link from "next/link";
import { Plus, Search, Link2 } from "lucide-react";
import useSWR from "swr";
import { toast } from "sonner";
import { linksAPI } from "@/lib/api";
import { getToken } from "@/lib/auth";
import { LinkCard, LinkCardPlaceholder } from "@/components/LinkCard";
import type { LinkItem } from "@/components/LinkCard";

const PAGE_SIZE = 20;

export default function DashboardPage() {
  const token = getToken();
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");

  const { data, error, isLoading, mutate } = useSWR(
    token ? [`links`, page, search] : null,
    () => linksAPI.list(token!, page, PAGE_SIZE)
  );

  const links: LinkItem[] = data?.links ?? [];
  const totalCount: number = data?.total_count ?? 0;
  const totalPages = Math.ceil(totalCount / PAGE_SIZE);

  const handleDelete = async (id: number) => {
    if (!token) return;
    try {
      await linksAPI.delete(token, id);
      mutate();
    } catch {
      toast.error("删除失败");
      throw new Error("Delete failed");
    }
  };

  return (
    <div>
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">我的链接</h1>
          <p className="text-sm text-gray-500 mt-1">{totalCount} 个短链接</p>
        </div>
        <Link
          href="/dashboard/links/new"
          className="inline-flex items-center justify-center gap-2 rounded-lg bg-indigo-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-indigo-700 transition shadow-sm"
        >
          <Plus className="w-4 h-4" />
          创建链接
        </Link>
      </div>

      {/* Search (placeholder for now) */}
      <div className="relative mb-4">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
        <input
          type="text"
          value={search}
          onChange={(e) => { setSearch(e.target.value); setPage(1); }}
          placeholder="搜索链接..."
          className="w-full sm:w-80 pl-9 pr-4 py-2.5 rounded-lg border border-gray-200 text-sm focus:border-indigo-300 focus:outline-none focus:ring-2 focus:ring-indigo-100 transition"
        />
      </div>

      {/* Content */}
      {error ? (
        <div className="text-center py-16 bg-white rounded-xl border border-gray-100">
          <div className="text-4xl mb-4">😞</div>
          <h3 className="text-lg font-medium text-gray-900">加载失败</h3>
          <p className="text-sm text-gray-500 mt-1">请检查网络连接后重试</p>
          <button
            onClick={() => mutate()}
            className="mt-4 text-sm text-indigo-600 hover:text-indigo-500 font-medium"
          >
            重新加载
          </button>
        </div>
      ) : isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <LinkCardPlaceholder key={i} />
          ))}
        </div>
      ) : links.length === 0 ? (
        <div className="text-center py-16 bg-white rounded-xl border border-gray-100">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-indigo-50 mb-4">
            <Link2 className="w-8 h-8 text-indigo-400" />
          </div>
          <h3 className="text-lg font-medium text-gray-900">
            {search ? "没有匹配的链接" : "还没有短链接"}
          </h3>
          <p className="text-sm text-gray-500 mt-1">
            {search ? "试试其他关键词" : "创建你的第一个短链接开始使用"}
          </p>
          {!search && (
            <Link
              href="/dashboard/links/new"
              className="inline-flex items-center gap-2 mt-4 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 transition"
            >
              <Plus className="w-4 h-4" /> 创建链接
            </Link>
          )}
        </div>
      ) : (
        <div className="space-y-3">
          {links.map((link) => (
            <LinkCard key={link.id} link={link} onDelete={handleDelete} />
          ))}

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-1 pt-4">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
                className="px-3 py-1.5 text-sm rounded-lg border border-gray-200 hover:bg-gray-50 disabled:opacity-40 disabled:cursor-not-allowed transition"
              >
                上一页
              </button>

              {generatePageNumbers(page, totalPages).map((p, i) =>
                p === null ? (
                  <span key={`dot-${i}`} className="px-2 text-gray-400">...</span>
                ) : (
                  <button
                    key={p}
                    onClick={() => setPage(p)}
                    className={`w-8 h-8 text-sm rounded-lg transition ${
                      p === page
                        ? "bg-indigo-600 text-white font-medium"
                        : "border border-gray-200 hover:bg-gray-50 text-gray-600"
                    }`}
                  >
                    {p}
                  </button>
                )
              )}

              <button
                onClick={() => setPage((p) => p + 1)}
                disabled={page >= totalPages}
                className="px-3 py-1.5 text-sm rounded-lg border border-gray-200 hover:bg-gray-50 disabled:opacity-40 disabled:cursor-not-allowed transition"
              >
                下一页
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

/** Generate smart page numbers like: 1 ... 4 [5] 6 ... 10 */
function generatePageNumbers(current: number, total: number): (number | null)[] {
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1);

  const pages: (number | null)[] = [];
  pages.push(1);

  if (current > 3) pages.push(null);

  for (let i = Math.max(2, current - 1); i <= Math.min(total - 1, current + 1); i++) {
    pages.push(i);
  }

  if (current < total - 2) pages.push(null);

  pages.push(total);
  return pages;
}
