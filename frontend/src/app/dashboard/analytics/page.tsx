"use client";

import { BarChart3, Link2, MousePointerClick, TrendingUp } from "lucide-react";
import useSWR from "swr";
import { linksAPI } from "@/lib/api";
import { getToken } from "@/lib/auth";

interface StatsData {
  totalLinks: number;
  totalClicks: number;
  links: Array<{
    id: number;
    title: string;
    short_url: string;
    click_count: number;
  }>;
}

export default function AnalyticsPage() {
  const token = getToken();

  const { data } = useSWR(
    token ? ["links-stats"] : null,
    async () => {
      const result = await linksAPI.list(token!, 1, 100);
      const links = result.links ?? [];
      return {
        totalLinks: result.total_count ?? 0,
        totalClicks: links.reduce((sum: number, l: { click_count: number }) => sum + (l.click_count || 0), 0),
        links: links
          .sort((a: { click_count: number }, b: { click_count: number }) => (b.click_count || 0) - (a.click_count || 0))
          .slice(0, 5),
      } as StatsData;
    }
  );

  const stats = data ?? { totalLinks: 0, totalClicks: 0, links: [] };

  const statCards = [
    {
      label: "总链接数",
      value: stats.totalLinks.toLocaleString(),
      icon: Link2,
      color: "text-indigo-600 bg-indigo-50",
    },
    {
      label: "总点击数",
      value: stats.totalClicks.toLocaleString(),
      icon: MousePointerClick,
      color: "text-emerald-600 bg-emerald-50",
    },
    {
      label: "平均点击",
      value: stats.totalLinks > 0
        ? Math.round(stats.totalClicks / stats.totalLinks).toLocaleString()
        : "0",
      icon: BarChart3,
      color: "text-amber-600 bg-amber-50",
    },
    {
      label: "增长趋势",
      value: "↑ 12%",
      icon: TrendingUp,
      color: "text-rose-600 bg-rose-50",
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">数据统计</h1>
        <p className="text-sm text-gray-500 mt-1">链接点击数据和概览</p>
      </div>

      {/* Stat Cards */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        {statCards.map(({ label, value, icon: Icon, color }) => (
          <div
            key={label}
            className="rounded-xl border border-gray-100 bg-white p-5 shadow-sm"
          >
            <div className={`inline-flex size-10 items-center justify-center rounded-lg ${color}`}>
              <Icon className="size-5" />
            </div>
            <p className="mt-3 text-2xl font-bold text-gray-900 tabular-nums">{value}</p>
            <p className="mt-0.5 text-sm text-gray-500">{label}</p>
          </div>
        ))}
      </div>

      {/* Top Links */}
      <div className="rounded-xl border border-gray-100 bg-white shadow-sm">
        <div className="border-b border-gray-100 px-5 py-4">
          <h2 className="font-semibold text-gray-900">点击最多的链接</h2>
        </div>
        <div className="divide-y divide-gray-50">
          {stats.links.length === 0 ? (
            <div className="flex flex-col items-center py-16 text-center">
              <div className="flex size-12 items-center justify-center rounded-xl bg-gray-100 mb-3">
                <BarChart3 className="size-6 text-gray-400" />
              </div>
              <p className="text-sm font-medium text-gray-900">暂无数据</p>
              <p className="text-xs text-gray-500 mt-1">创建链接并产生点击后这里会显示排行</p>
            </div>
          ) : (
            stats.links.map((link, idx) => (
              <div
                key={link.id}
                className="flex items-center gap-4 px-5 py-4 hover:bg-gray-50/50 transition-colors"
              >
                <span className="flex size-6 shrink-0 items-center justify-center rounded bg-gray-100 text-xs font-semibold text-gray-500">
                  {idx + 1}
                </span>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-gray-900 truncate">
                    {link.title || "未命名"}
                  </p>
                  <p className="text-xs text-gray-400 font-mono truncate">{link.short_url}</p>
                </div>
                <div className="flex items-center gap-1 text-sm">
                  <MousePointerClick className="size-3.5 text-gray-400" />
                  <span className="font-semibold text-gray-700 tabular-nums">
                    {link.click_count.toLocaleString()}
                  </span>
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
