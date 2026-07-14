"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { ChevronDown, ChevronRight, Globe, Shield, Clock, Smartphone, Tags } from "lucide-react";
import useSWR from "swr";
import { linksAPI, domainsAPI, utmAPI } from "@/lib/api";
import { getToken } from "@/lib/auth";

export default function CreateLinkPage() {
  const router = useRouter();
  const token = getToken();

  // Basic fields
  const [originalUrl, setOriginalUrl] = useState("");
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");

  // Advanced sections
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [showUTM, setShowUTM] = useState(false);
  const [showDeeplink, setShowDeeplink] = useState(false);

  // Domain
  const [domain, setDomain] = useState("");

  // Password + expiration
  const [password, setPassword] = useState("");
  const [expiresAt, setExpiresAt] = useState("");

  // UTM
  const [utmSource, setUtmSource] = useState("");
  const [utmMedium, setUtmMedium] = useState("");
  const [utmCampaign, setUtmCampaign] = useState("");
  const [utmTerm, setUtmTerm] = useState("");
  const [utmContent, setUtmContent] = useState("");

  // Deep links
  const [iosUrl, setIosUrl] = useState("");
  const [androidUrl, setAndroidUrl] = useState("");

  const [loading, setLoading] = useState(false);

  // Fetch domains and UTM templates
  const { data: domainData } = useSWR(token ? "domains" : null, () => domainsAPI.list(token!));
  const { data: utmData } = useSWR(token ? "utm-templates" : null, () => utmAPI.list(token!));

  const verifiedDomains = (domainData?.domains ?? []).filter((d: { verified: boolean }) => d.verified);
  const utmTemplates = utmData?.templates ?? [];

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const applyUTMTemplate = (t: any) => {
    setUtmSource(t.utm_source || "");
    setUtmMedium(t.utm_medium || "");
    setUtmCampaign(t.utm_campaign || "");
    setUtmTerm(t.utm_term || "");
    setUtmContent(t.utm_content || "");
    setShowUTM(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!token) { router.push("/login"); return; }
    setLoading(true);

    try {
      await linksAPI.create(token, {
        original_url: originalUrl,
        title: title || undefined,
        description: description || undefined,
        domain: domain || undefined,
        password: password || undefined,
        expires_at: expiresAt || undefined,
        utm_source: utmSource || undefined,
        utm_medium: utmMedium || undefined,
        utm_campaign: utmCampaign || undefined,
        utm_term: utmTerm || undefined,
        utm_content: utmContent || undefined,
        ios_url: iosUrl || undefined,
        android_url: androidUrl || undefined,
      });
      toast.success("短链接创建成功！");
      router.push("/dashboard");
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "创建失败");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold text-gray-900 mb-6">创建短链接</h1>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Basic Info */}
        <div className="bg-white rounded-xl border border-gray-100 shadow-sm p-6 space-y-4">
          <h2 className="font-semibold text-gray-900 text-sm uppercase tracking-wider">基本信息</h2>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              目标 URL <span className="text-red-500">*</span>
            </label>
            <input
              type="url"
              value={originalUrl}
              onChange={(e) => setOriginalUrl(e.target.value)}
              className="w-full rounded-lg border border-gray-200 px-4 py-2.5 text-sm focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 transition"
              placeholder="https://example.com/your-long-url"
              required
            />
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">标题</label>
              <input
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                className="w-full rounded-lg border border-gray-200 px-4 py-2.5 text-sm focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 transition"
                placeholder="我的推广链接"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                <span className="inline-flex items-center gap-1.5">
                  <Globe className="size-3.5 text-gray-400" />
                  自定义域名
                </span>
              </label>
              <select
                value={domain}
                onChange={(e) => setDomain(e.target.value)}
                className="w-full rounded-lg border border-gray-200 px-4 py-2.5 text-sm focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 transition bg-white"
              >
                <option value="">kada.link（默认）</option>
                {verifiedDomains.map((d: { id: number; name: string }) => (
                  <option key={d.id} value={d.name}>{d.name}</option>
                ))}
              </select>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">描述</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="w-full rounded-lg border border-gray-200 px-4 py-2.5 text-sm focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 transition"
              placeholder="简短描述（可选）"
              rows={2}
            />
          </div>
        </div>

        {/* Advanced Options */}
        <div className="bg-white rounded-xl border border-gray-100 shadow-sm overflow-hidden">
          <button
            type="button"
            onClick={() => setShowAdvanced(!showAdvanced)}
            className="w-full flex items-center justify-between p-4 text-sm font-medium text-gray-700 hover:bg-gray-50 transition"
          >
            <span className="flex items-center gap-2">
              <Shield className="size-4 text-gray-400" />
              高级选项
            </span>
            {showAdvanced ? <ChevronDown className="size-4" /> : <ChevronRight className="size-4" />}
          </button>

          {showAdvanced && (
            <div className="border-t border-gray-100 p-6 space-y-4">
              <div className="grid gap-4 sm:grid-cols-2">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    密码保护
                  </label>
                  <input
                    type="text"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className="w-full rounded-lg border border-gray-200 px-4 py-2.5 text-sm focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 transition"
                    placeholder="留空则不加密"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    <span className="inline-flex items-center gap-1.5">
                      <Clock className="size-3.5 text-gray-400" />
                      过期时间
                    </span>
                  </label>
                  <input
                    type="datetime-local"
                    value={expiresAt}
                    onChange={(e) => setExpiresAt(e.target.value)}
                    className="w-full rounded-lg border border-gray-200 px-4 py-2.5 text-sm focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 transition"
                  />
                </div>
              </div>
            </div>
          )}
        </div>

        {/* UTM Parameters */}
        <div className="bg-white rounded-xl border border-gray-100 shadow-sm overflow-hidden">
          <button
            type="button"
            onClick={() => setShowUTM(!showUTM)}
            className="w-full flex items-center justify-between p-4 text-sm font-medium text-gray-700 hover:bg-gray-50 transition"
          >
            <span className="flex items-center gap-2">
              <Tags className="size-4 text-gray-400" />
              UTM 参数
            </span>
            {showUTM ? <ChevronDown className="size-4" /> : <ChevronRight className="size-4" />}
          </button>

          {showUTM && (
            <div className="border-t border-gray-100 p-6 space-y-4">
              {/* UTM Template quick-fill */}
              {utmTemplates.length > 0 && (
                <div className="flex flex-wrap items-center gap-2 pb-3 border-b border-gray-50">
                  <span className="text-xs text-gray-500">快速填充：</span>
                  {utmTemplates.map((t: { id: number; name: string }) => (
                    <button
                      key={t.id}
                      type="button"
                      onClick={() => applyUTMTemplate(t)}
                      className="text-xs px-2.5 py-1 rounded-full bg-violet-50 text-violet-700 hover:bg-violet-100 transition"
                    >
                      {t.name}
                    </button>
                  ))}
                </div>
              )}

              <div className="grid gap-4 sm:grid-cols-2">
                <div>
                  <label className="block text-xs font-medium text-gray-500 mb-1">来源 (utm_source)</label>
                  <input type="text" value={utmSource} onChange={(e) => setUtmSource(e.target.value)}
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm font-mono focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 transition"
                    placeholder="wechat" />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-500 mb-1">媒介 (utm_medium)</label>
                  <input type="text" value={utmMedium} onChange={(e) => setUtmMedium(e.target.value)}
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm font-mono focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 transition"
                    placeholder="social" />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-500 mb-1">活动 (utm_campaign)</label>
                  <input type="text" value={utmCampaign} onChange={(e) => setUtmCampaign(e.target.value)}
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm font-mono focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 transition"
                    placeholder="spring-sale" />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-500 mb-1">关键词 (utm_term)</label>
                  <input type="text" value={utmTerm} onChange={(e) => setUtmTerm(e.target.value)}
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm font-mono focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 transition"
                    placeholder="shoes" />
                </div>
                <div className="sm:col-span-2">
                  <label className="block text-xs font-medium text-gray-500 mb-1">内容 (utm_content)</label>
                  <input type="text" value={utmContent} onChange={(e) => setUtmContent(e.target.value)}
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm font-mono focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 transition"
                    placeholder="banner-top" />
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Deep Links */}
        <div className="bg-white rounded-xl border border-gray-100 shadow-sm overflow-hidden">
          <button
            type="button"
            onClick={() => setShowDeeplink(!showDeeplink)}
            className="w-full flex items-center justify-between p-4 text-sm font-medium text-gray-700 hover:bg-gray-50 transition"
          >
            <span className="flex items-center gap-2">
              <Smartphone className="size-4 text-gray-400" />
              深度链接（App 跳转）
            </span>
            {showDeeplink ? <ChevronDown className="size-4" /> : <ChevronRight className="size-4" />}
          </button>

          {showDeeplink && (
            <div className="border-t border-gray-100 p-6 space-y-4">
              <div>
                <label className="block text-xs font-medium text-gray-500 mb-1">iOS URL Scheme</label>
                <input type="text" value={iosUrl} onChange={(e) => setIosUrl(e.target.value)}
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm font-mono focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 transition"
                  placeholder="myapp://product/123" />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-500 mb-1">Android URL Scheme</label>
                <input type="text" value={androidUrl} onChange={(e) => setAndroidUrl(e.target.value)}
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm font-mono focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 transition"
                  placeholder="myapp://product/123" />
              </div>
            </div>
          )}
        </div>

        {/* Submit */}
        <div className="flex items-center gap-3">
          <button
            type="submit"
            disabled={loading || !originalUrl}
            className="flex-1 rounded-lg bg-indigo-600 py-3 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-40 disabled:cursor-not-allowed transition shadow-sm"
          >
            {loading ? "创建中..." : "创建短链接"}
          </button>
          <button
            type="button"
            onClick={() => router.back()}
            className="rounded-lg border border-gray-200 px-6 py-3 text-sm font-medium text-gray-600 hover:bg-gray-50 transition"
          >
            取消
          </button>
        </div>
      </form>
    </div>
  );
}
