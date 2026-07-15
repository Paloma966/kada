"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { Globe, Clock, Shield, Smartphone, Tags } from "lucide-react";
import useSWR from "swr";
import { linksAPI, domainsAPI, utmAPI } from "@/lib/api";
import { getToken } from "@/lib/auth";

export default function CreateLinkPage() {
  const router = useRouter();
  const token = getToken();

  const [originalUrl, setOriginalUrl] = useState("");
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [domain, setDomain] = useState("");
  const [password, setPassword] = useState("");
  const [expiresAt, setExpiresAt] = useState("");
  const [utmTemplateId, setUtmTemplateId] = useState<number | null>(null);
  const [iosUrl, setIosUrl] = useState("");
  const [androidUrl, setAndroidUrl] = useState("");
  const [loading, setLoading] = useState(false);

  const { data: domainData } = useSWR(token ? "domains" : null, () => domainsAPI.list(token!));
  const { data: utmData } = useSWR(token ? "utm-templates" : null, () => utmAPI.list(token!));

  const verifiedDomains = (domainData?.domains ?? []).filter((d: { verified: boolean }) => d.verified);
  const utmTemplates = utmData?.templates ?? [];
  const selectedTemplate = utmTemplates.find((t: { id: number }) => t.id === utmTemplateId);

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
        utm_source: selectedTemplate?.utm_source || undefined,
        utm_medium: selectedTemplate?.utm_medium || undefined,
        utm_campaign: selectedTemplate?.utm_campaign || undefined,
        utm_term: selectedTemplate?.utm_term || undefined,
        utm_content: selectedTemplate?.utm_content || undefined,
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
    <div className="max-w-4xl mx-auto">
      <h1 className="text-2xl font-bold text-gray-900 mb-6">创建短链接</h1>

      <form onSubmit={handleSubmit}>
        <div className="bg-white rounded-2xl border border-gray-100 shadow-sm overflow-hidden">
          <div className="grid lg:grid-cols-5 divide-y lg:divide-y-0 lg:divide-x divide-gray-100">

            {/* ===== Left: Main ===== */}
            <div className="lg:col-span-3 p-6 sm:p-8 space-y-5">
              <h2 className="text-sm font-semibold text-gray-900 uppercase tracking-wider">基本信息</h2>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">
                  目标 URL <span className="text-red-500">*</span>
                </label>
                <input
                  type="url"
                  value={originalUrl}
                  onChange={(e) => setOriginalUrl(e.target.value)}
                  className="w-full rounded-xl border border-gray-200 px-4 py-3 text-sm focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 transition"
                  placeholder="https://example.com/your-long-url"
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">标题</label>
                <input
                  type="text"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  className="w-full rounded-xl border border-gray-200 px-4 py-3 text-sm focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 transition"
                  placeholder="我的推广链接"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">描述</label>
                <textarea
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  className="w-full rounded-xl border border-gray-200 px-4 py-3 text-sm focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 transition resize-none"
                  placeholder="简短描述（可选）"
                  rows={3}
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">
                  <span className="inline-flex items-center gap-1.5">
                    <Smartphone className="size-3.5 text-gray-400" />
                    App 深度链接
                  </span>
                </label>
                <div className="grid grid-cols-2 gap-3">
                  <input type="text" value={iosUrl} onChange={(e) => setIosUrl(e.target.value)}
                    className="w-full rounded-xl border border-gray-200 px-4 py-3 text-sm font-mono focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 transition"
                    placeholder="iOS URL" />
                  <input type="text" value={androidUrl} onChange={(e) => setAndroidUrl(e.target.value)}
                    className="w-full rounded-xl border border-gray-200 px-4 py-3 text-sm font-mono focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 transition"
                    placeholder="Android URL" />
                </div>
              </div>
            </div>

            {/* ===== Right: Advanced ===== */}
            <div className="lg:col-span-2 p-6 sm:p-8 space-y-6 bg-gray-50/50">
              <h2 className="text-sm font-semibold text-gray-900 uppercase tracking-wider">高级选项</h2>

              {/* Domain */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">
                  <span className="inline-flex items-center gap-1.5">
                    <Globe className="size-3.5 text-gray-400" />
                    域名
                  </span>
                </label>
                <select
                  value={domain}
                  onChange={(e) => setDomain(e.target.value)}
                  className="w-full rounded-xl border border-gray-200 px-4 py-3 text-sm focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 transition bg-white"
                >
                  <option value="">kada.click（默认）</option>
                  {verifiedDomains.map((d: { id: number; name: string }) => (
                    <option key={d.id} value={d.name}>{d.name}</option>
                  ))}
                </select>
              </div>

              {/* Password */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">
                  <span className="inline-flex items-center gap-1.5">
                    <Shield className="size-3.5 text-gray-400" />
                    密码保护
                  </span>
                </label>
                <input
                  type="text"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full rounded-xl border border-gray-200 px-4 py-3 text-sm focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 transition"
                  placeholder="留空则不加密"
                />
              </div>

              {/* Expiration */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">
                  <span className="inline-flex items-center gap-1.5">
                    <Clock className="size-3.5 text-gray-400" />
                    过期时间
                  </span>
                </label>
                <input
                  type="datetime-local"
                  value={expiresAt}
                  onChange={(e) => setExpiresAt(e.target.value)}
                  className="w-full rounded-xl border border-gray-200 px-4 py-3 text-sm focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 transition"
                />
              </div>

              {/* UTM Template */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">
                  <span className="inline-flex items-center gap-1.5">
                    <Tags className="size-3.5 text-gray-400" />
                    UTM 模板
                  </span>
                </label>
                {utmTemplates.length > 0 ? (
                  <div className="space-y-2">
                    <select
                      value={utmTemplateId ?? ""}
                      onChange={(e) => setUtmTemplateId(e.target.value ? Number(e.target.value) : null)}
                      className="w-full rounded-xl border border-gray-200 px-4 py-3 text-sm focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 transition bg-white"
                    >
                      <option value="">不使用模板</option>
                      {utmTemplates.map((t: { id: number; name: string }) => (
                        <option key={t.id} value={t.id}>{t.name}</option>
                      ))}
                    </select>
                    {selectedTemplate && (
                      <div className="flex flex-wrap gap-1.5">
                        {[
                          ["source", selectedTemplate.utm_source],
                          ["medium", selectedTemplate.utm_medium],
                          ["campaign", selectedTemplate.utm_campaign],
                          ["term", selectedTemplate.utm_term],
                          ["content", selectedTemplate.utm_content],
                        ].filter(([, v]) => v).map(([k, v]) => (
                          <span key={k} className="text-[10px] bg-violet-50 text-violet-600 px-2 py-0.5 rounded-full font-mono">
                            {k}={v}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                ) : (
                  <p className="text-xs text-gray-400 py-2">
                    还没有 UTM 模板，去<a href="/dashboard/utm" className="text-indigo-600 hover:underline ml-1">UTM 模板</a>创建
                  </p>
                )}
              </div>
            </div>
          </div>

          {/* Footer */}
          <div className="border-t border-gray-100 px-6 py-4 flex items-center justify-between bg-white">
            <button
              type="button"
              onClick={() => router.back()}
              className="text-sm text-gray-500 hover:text-gray-700 transition"
            >
              取消
            </button>
            <button
              type="submit"
              disabled={loading || !originalUrl}
              className="rounded-xl bg-indigo-600 px-8 py-2.5 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-40 disabled:cursor-not-allowed transition shadow-sm"
            >
              {loading ? "创建中..." : "创建短链接"}
            </button>
          </div>
        </div>
      </form>
    </div>
  );
}
