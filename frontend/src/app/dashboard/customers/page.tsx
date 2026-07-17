"use client";

import useSWR from "swr";
import { Users, Monitor } from "lucide-react";
import { analyticsAPI } from "@/lib/api";
import { getToken } from "@/lib/auth";

export default function CustomersPage() {
  const token = getToken();

  const { data, isLoading } = useSWR(
    token ? "analytics-customers" : null,
    () => analyticsAPI.customers(token!)
  );

  const customers = data?.customers ?? [];

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">客户</h1>
        <p className="text-sm text-gray-500 mt-1">独立访客统计（按 IP）</p>
      </div>

      {/* Summary cards */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-3">
        <div className="rounded-xl border border-gray-100 bg-white p-5 shadow-sm">
          <div className="inline-flex size-10 items-center justify-center rounded-lg text-indigo-600 bg-indigo-50">
            <Users className="size-5" />
          </div>
          <p className="mt-3 text-2xl font-bold text-gray-900 tabular-nums">
            {customers.length.toLocaleString()}
          </p>
          <p className="mt-0.5 text-sm text-gray-500">独立 IP 数</p>
        </div>
        <div className="rounded-xl border border-gray-100 bg-white p-5 shadow-sm">
          <div className="inline-flex size-10 items-center justify-center rounded-lg text-emerald-600 bg-emerald-50">
            <Monitor className="size-5" />
          </div>
          <p className="mt-3 text-2xl font-bold text-gray-900 tabular-nums">
            {customers.reduce((sum: number, c: { click_count: number }) => sum + c.click_count, 0).toLocaleString()}
          </p>
          <p className="mt-0.5 text-sm text-gray-500">总点击次数</p>
        </div>
      </div>

      {/* Customer table */}
      <div className="bg-white rounded-xl border border-gray-100 shadow-sm overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-100 bg-gray-50/50">
                <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">IP 地址</th>
                <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">点击次数</th>
                <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">独立链接</th>
                <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">最近访问</th>
              </tr>
            </thead>
            <tbody>
              {isLoading ? (
                Array.from({ length: 5 }).map((_, i) => (
                  <tr key={i} className="border-b border-gray-50 animate-pulse">
                    {Array.from({ length: 4 }).map((_, j) => (
                      <td key={j} className="px-5 py-3"><div className="h-4 bg-gray-100 rounded w-20" /></td>
                    ))}
                  </tr>
                ))
              ) : customers.length === 0 ? (
                <tr>
                  <td colSpan={4} className="px-5 py-16 text-center">
                    <Users className="size-8 text-gray-300 mx-auto mb-2" />
                    <p className="text-sm text-gray-500">暂无访客数据</p>
                  </td>
                </tr>
              ) : (
                customers.map((c: {
                  ip: string;
                  click_count: number;
                  unique_links: number;
                  last_seen: string;
                }, i: number) => (
                  <tr key={c.ip} className="border-b border-gray-50 hover:bg-gray-50/50 transition">
                    <td className="px-5 py-3">
                      <code className="text-gray-700 font-mono text-xs">{c.ip}</code>
                    </td>
                    <td className="px-5 py-3">
                      <span className="text-gray-900 font-medium tabular-nums">{c.click_count}</span>
                    </td>
                    <td className="px-5 py-3 text-gray-500 tabular-nums">{c.unique_links}</td>
                    <td className="px-5 py-3 text-gray-500 text-xs">
                      {new Date(c.last_seen).toLocaleString("zh-CN")}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
