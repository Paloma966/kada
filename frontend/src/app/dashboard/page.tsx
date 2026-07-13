"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { Plus, Copy, ExternalLink, Trash2 } from "lucide-react";
import useSWR from "swr";
import { linksAPI } from "@/lib/api";
import { getToken } from "@/lib/auth";

interface LinkItem {
  id: number;
  short_code: string;
  short_url: string;
  original_url: string;
  title: string;
  click_count: number;
  is_active: boolean;
  created_at: string;
}

export default function DashboardPage() {
  const token = getToken();
  const [page, setPage] = useState(1);

  const { data, error, isLoading, mutate } = useSWR(
    token ? [`links`, page] : null,
    () => linksAPI.list(token!, page)
  );

  const handleDelete = async (id: number) => {
    if (!confirm("确定删除这个短链接？") || !token) return;
    try {
      await linksAPI.delete(token, id);
      mutate();
    } catch (e) {
      console.error(e);
    }
  };

  const copyToClipboard = (url: string) => {
    navigator.clipboard.writeText(url);
  };

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">我的链接</h1>
          <p className="text-sm text-gray-500 mt-1">
            {data?.total_count || 0} 个短链接
          </p>
        </div>
        <Link
          href="/dashboard/links/new"
          className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 transition"
        >
          <Plus className="w-4 h-4" /> 创建链接
        </Link>
      </div>

      {/* Link List */}
      {isLoading ? (
        <div className="text-center py-12 text-gray-400">加载中...</div>
      ) : error ? (
        <div className="text-center py-12 text-red-500">加载失败</div>
      ) : !data?.links?.length ? (
        <div className="text-center py-16 bg-white rounded-xl shadow-sm">
          <div className="text-4xl mb-4">🔗</div>
          <h3 className="text-lg font-medium text-gray-900">还没有短链接</h3>
          <p className="text-sm text-gray-500 mt-1">创建你的第一个短链接</p>
          <Link
            href="/dashboard/links/new"
            className="inline-flex items-center gap-2 mt-4 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 transition"
          >
            <Plus className="w-4 h-4" /> 创建链接
          </Link>
        </div>
      ) : (
        <div className="space-y-3">
          {data.links.map((link: LinkItem) => (
            <div
              key={link.id}
              className="bg-white rounded-xl p-4 shadow-sm hover:shadow-md transition flex items-center justify-between gap-4"
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <h3 className="font-medium text-gray-900 truncate">
                    {link.title || link.short_code}
                  </h3>
                  <span className={`px-2 py-0.5 text-xs rounded-full ${link.is_active ? "bg-green-100 text-green-700" : "bg-gray-100 text-gray-500"}`}>
                    {link.is_active ? "启用" : "停用"}
                  </span>
                </div>
                <div className="flex items-center gap-3 mt-1 text-sm text-gray-500">
                  <span className="text-indigo-600 font-mono">{link.short_url}</span>
                  <button
                    onClick={() => copyToClipboard(link.short_url)}
                    className="hover:text-indigo-600 transition"
                  >
                    <Copy className="w-3.5 h-3.5" />
                  </button>
                </div>
                <p className="text-xs text-gray-400 mt-1 truncate">{link.original_url}</p>
              </div>

              <div className="flex items-center gap-4 text-sm">
                <div className="text-center">
                  <div className="font-semibold text-gray-900">{link.click_count.toLocaleString()}</div>
                  <div className="text-xs text-gray-400">点击</div>
                </div>
                <a
                  href={link.short_url}
                  target="_blank"
                  className="p-2 text-gray-400 hover:text-indigo-600 transition"
                >
                  <ExternalLink className="w-4 h-4" />
                </a>
                <button
                  onClick={() => handleDelete(link.id)}
                  className="p-2 text-gray-400 hover:text-red-500 transition"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            </div>
          ))}

          {/* Pagination */}
          {data.total_count > 20 && (
            <div className="flex justify-center gap-2 pt-4">
              <button
                onClick={() => setPage(p => Math.max(1, p - 1))}
                disabled={page === 1}
                className="px-4 py-2 text-sm rounded-lg bg-white border disabled:opacity-50"
              >
                上一页
              </button>
              <span className="px-4 py-2 text-sm text-gray-500">第 {page} 页</span>
              <button
                onClick={() => setPage(p => p + 1)}
                disabled={data.links.length < 20}
                className="px-4 py-2 text-sm rounded-lg bg-white border disabled:opacity-50"
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
