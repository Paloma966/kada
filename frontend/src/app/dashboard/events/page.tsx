"use client";

import { useState } from "react";
import useSWR from "swr";
import { ExternalLink, MousePointerClick, ChevronLeft, ChevronRight } from "lucide-react";
import { analyticsAPI } from "@/lib/api";
import { getToken } from "@/lib/auth";

const platformLabels: Record<string, string> = {
  browser: "浏览器",
  wechat: "微信",
  qq: "QQ",
  weibo: "微博",
  xiaohongshu: "小红书",
  sms: "短信",
  unknown: "其他",
};

export default function EventsPage() {
  const token = getToken();
  const [page, setPage] = useState(1);

  const { data, isLoading } = useSWR(
    token ? `analytics-events-${page}` : null,
    () => analyticsAPI.events(token!, page, 20)
  );

  const events = data?.events ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.ceil(total / 20);

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">事件</h1>
        <p className="text-sm text-gray-500 mt-1">点击事件记录</p>
      </div>

      <div className="bg-white rounded-xl border border-gray-100 shadow-sm overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-100 bg-gray-50/50">
                <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">时间</th>
                <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">短链</th>
                <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">平台</th>
                <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">IP</th>
                <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">来源</th>
              </tr>
            </thead>
            <tbody>
              {isLoading ? (
                Array.from({ length: 5 }).map((_, i) => (
                  <tr key={i} className="border-b border-gray-50 animate-pulse">
                    {Array.from({ length: 5 }).map((_, j) => (
                      <td key={j} className="px-5 py-3"><div className="h-4 bg-gray-100 rounded w-20" /></td>
                    ))}
                  </tr>
                ))
              ) : events.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-5 py-16 text-center">
                    <MousePointerClick className="size-8 text-gray-300 mx-auto mb-2" />
                    <p className="text-sm text-gray-500">暂无点击事件</p>
                  </td>
                </tr>
              ) : (
                events.map((e: {
                  id: number;
                  short_code: string;
                  original_url: string;
                  platform: string;
                  ip: string;
                  referer: string;
                  created_at: string;
                }) => (
                  <tr key={e.id} className="border-b border-gray-50 hover:bg-gray-50/50 transition">
                    <td className="px-5 py-3 text-gray-600 whitespace-nowrap font-mono text-xs">
                      {new Date(e.created_at).toLocaleString("zh-CN")}
                    </td>
                    <td className="px-5 py-3">
                      <div className="flex items-center gap-1.5">
                        <code className="text-indigo-600 font-mono text-xs">{e.short_code}</code>
                        <a href={e.original_url} target="_blank" className="text-gray-300 hover:text-gray-500">
                          <ExternalLink className="size-3" />
                        </a>
                      </div>
                    </td>
                    <td className="px-5 py-3">
                      <span className="text-xs px-2 py-0.5 rounded-full bg-gray-100 text-gray-600">
                        {platformLabels[e.platform] || e.platform || "未知"}
                      </span>
                    </td>
                    <td className="px-5 py-3 text-gray-500 font-mono text-xs">{e.ip || "-"}</td>
                    <td className="px-5 py-3 text-gray-400 text-xs max-w-[180px] truncate">
                      {e.referer || "直接访问"}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {totalPages > 1 && (
          <div className="flex items-center justify-between px-5 py-3 border-t border-gray-100">
            <p className="text-xs text-gray-500">共 {total} 条记录</p>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setPage(p => Math.max(1, p - 1))}
                disabled={page <= 1}
                className="p-1.5 rounded-lg text-gray-500 hover:bg-gray-100 disabled:opacity-30 disabled:cursor-not-allowed transition"
              >
                <ChevronLeft className="size-4" />
              </button>
              <span className="text-xs text-gray-600 tabular-nums">{page} / {totalPages}</span>
              <button
                onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
                className="p-1.5 rounded-lg text-gray-500 hover:bg-gray-100 disabled:opacity-30 disabled:cursor-not-allowed transition"
              >
                <ChevronRight className="size-4" />
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
