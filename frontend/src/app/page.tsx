import Link from "next/link";
import { Link2, Shield, Smartphone, Zap, BarChart3, Globe } from "lucide-react";

const FEATURES = [
  { icon: Zap, title: "智能短链", desc: "一键缩短长链接，支持自定义短码" },
  { icon: Globe, title: "全平台兼容", desc: "自动识别微信、QQ、小红书等平台，引导在浏览器打开" },
  { icon: BarChart3, title: "点击追踪", desc: "记录每次点击，追踪来源平台、设备和地理位置" },
  { icon: Smartphone, title: "短信友好", desc: "支持手机号验证码登录，短信链接不被拦截" },
  { icon: Shield, title: "安全可靠", desc: "HTTPS 加密传输，JWT 认证，数据隔离保障链接安全" },
  { icon: Link2, title: "API 开放", desc: "提供 RESTful API，方便集成到你的应用和工作流" },
];

export default function HomePage() {
  return (
    <div className="min-h-screen bg-white">
      {/* Nav */}
      <header className="sticky top-0 z-50 border-b border-neutral-200 bg-white">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4 sm:px-6">
          <div className="flex items-center gap-2">
            <div className="flex size-8 items-center justify-center rounded-lg bg-indigo-600">
              <Link2 className="size-4 text-white" />
            </div>
            <span className="text-lg font-semibold text-neutral-900">Kada</span>
          </div>
          <div className="flex items-center gap-2">
            <Link
              href="/login"
              className="rounded-lg px-4 py-2 text-sm font-medium text-neutral-600 transition hover:bg-neutral-50 hover:text-neutral-900"
            >
              登录
            </Link>
            <Link
              href="/register"
              className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-indigo-700"
            >
              免费注册
            </Link>
          </div>
        </div>
      </header>

      {/* Hero */}
      <section className="mx-auto max-w-6xl px-4 py-24 sm:px-6 sm:py-32">
        <div className="mx-auto max-w-3xl text-center">
          <h1 className="text-4xl font-bold tracking-tight text-neutral-900 sm:text-5xl">
            你的链接<span className="text-indigo-600">，无处不在</span>
          </h1>
          <p className="mx-auto mt-6 max-w-2xl text-lg leading-relaxed text-neutral-500">
            智能短链接管理平台，缩短、分享并追踪你的每一个链接。
            完美兼容微信、QQ、小红书等国内主流平台。
          </p>
          <div className="mt-10 flex flex-col items-center justify-center gap-3 sm:flex-row">
            <Link
              href="/register"
              className="w-full rounded-lg bg-indigo-600 px-8 py-3 text-base font-medium text-white transition hover:bg-indigo-700 sm:w-auto"
            >
              免费开始使用
            </Link>
            <Link
              href="/login"
              className="w-full rounded-lg border border-neutral-200 bg-white px-8 py-3 text-base font-medium text-neutral-700 transition hover:border-neutral-300 hover:bg-neutral-50 sm:w-auto"
            >
              登录
            </Link>
          </div>
        </div>
      </section>

      {/* Features */}
      <section className="border-t border-neutral-100 bg-neutral-50">
        <div className="mx-auto max-w-6xl px-4 py-20 sm:px-6 sm:py-24">
          <div className="mb-14 text-center">
            <h2 className="text-3xl font-bold tracking-tight text-neutral-900">
              一个平台，搞定所有链接
            </h2>
            <p className="mt-3 text-base text-neutral-500">为国内环境量身打造的短链接解决方案</p>
          </div>
          <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
            {FEATURES.map(({ icon: Icon, title, desc }) => (
              <div
                key={title}
                className="rounded-xl border border-neutral-200 bg-white p-6 transition-colors hover:border-neutral-300"
              >
                <div className="mb-4 flex size-10 items-center justify-center rounded-lg bg-indigo-50 text-indigo-600">
                  <Icon className="size-5" />
                </div>
                <h3 className="mb-1.5 text-base font-semibold text-neutral-900">{title}</h3>
                <p className="text-sm leading-relaxed text-neutral-500">{desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="bg-indigo-600">
        <div className="mx-auto max-w-6xl px-4 py-16 text-center sm:px-6">
          <h2 className="text-2xl font-bold text-white sm:text-3xl">准备好管理你的链接了吗？</h2>
          <p className="mt-3 text-base text-indigo-100">免费注册，几秒钟内创建你的第一个短链接</p>
          <Link
            href="/register"
            className="mt-8 inline-flex rounded-lg bg-white px-8 py-3 text-base font-medium text-indigo-600 transition hover:bg-indigo-50"
          >
            立即开始
          </Link>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-neutral-200 bg-white">
        <div className="mx-auto flex max-w-6xl flex-col items-center justify-between gap-4 px-4 py-10 sm:flex-row sm:px-6">
          <div className="flex items-center gap-2">
            <div className="flex size-6 items-center justify-center rounded bg-indigo-600">
              <Link2 className="size-3.5 text-white" />
            </div>
            <span className="text-sm font-medium text-neutral-700">Kada</span>
          </div>
          <p className="text-xs text-neutral-400">
            &copy; {new Date().getFullYear()} Kada. 保留所有权利。
          </p>
        </div>
      </footer>
    </div>
  );
}
