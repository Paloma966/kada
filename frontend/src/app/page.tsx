import Link from "next/link";
import { Link2, Shield, Smartphone, Zap, BarChart3, Globe } from "lucide-react";

const FEATURES = [
  {
    icon: Zap,
    title: "智能短链",
    desc: "一键缩短长链接，自动生成简短易记的短链，支持自定义短码",
  },
  {
    icon: Globe,
    title: "全平台兼容",
    desc: "自动识别微信、QQ、小红书等平台，智能引导用户在浏览器中打开",
  },
  {
    icon: BarChart3,
    title: "点击追踪",
    desc: "实时记录每次点击，追踪来源平台、设备和地理位置",
  },
  {
    icon: Smartphone,
    title: "短信友好",
    desc: "支持手机号验证码登录，短信链接不被拦截，兼容国内运营商",
  },
  {
    icon: Shield,
    title: "安全可靠",
    desc: "HTTPS 加密传输，JWT 身份认证，数据隔离保障链接安全",
  },
  {
    icon: Link2,
    title: "API 开放",
    desc: "提供 RESTful API，方便集成到你自己的应用和工作流中",
  },
];

export default function HomePage() {
  return (
    <div className="min-h-screen">
      {/* Nav */}
      <header className="sticky top-0 z-50 border-b border-transparent bg-white/80 backdrop-blur-md">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4 sm:px-6">
          <div className="flex items-center gap-2">
            <div className="flex size-8 items-center justify-center rounded-lg bg-indigo-600">
              <Link2 className="size-4 text-white" />
            </div>
            <span className="text-xl font-bold text-gray-900">Kada</span>
          </div>
          <div className="flex items-center gap-3">
            <Link
              href="/login"
              className="rounded-lg px-4 py-2 text-sm font-medium text-gray-600 hover:bg-gray-100 transition"
            >
              登录
            </Link>
            <Link
              href="/register"
              className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 transition shadow-sm"
            >
              免费注册
            </Link>
          </div>
        </div>
      </header>

      {/* Hero */}
      <section className="relative overflow-hidden">
        <div className="absolute inset-0 bg-gradient-to-br from-indigo-500 via-purple-500 to-pink-500 opacity-[0.03]" />
        <div className="mx-auto max-w-6xl px-4 sm:px-6 py-24 sm:py-32">
          <div className="mx-auto max-w-3xl text-center">
            <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-indigo-200 bg-indigo-50 px-4 py-1.5">
              <span className="relative flex size-2">
                <span className="absolute inline-flex size-full animate-ping rounded-full bg-green-400 opacity-75" />
                <span className="relative inline-flex size-2 rounded-full bg-green-500" />
              </span>
              <span className="text-xs font-medium text-indigo-700">已服务 1,000+ 用户</span>
            </div>
            <h1 className="text-4xl font-extrabold tracking-tight text-gray-900 sm:text-5xl lg:text-6xl">
              你的链接
              <span className="bg-gradient-to-r from-indigo-600 to-purple-600 bg-clip-text text-transparent">
                {" "}无处不在
              </span>
            </h1>
            <p className="mt-6 text-lg leading-relaxed text-gray-500 sm:text-xl max-w-2xl mx-auto">
              智能短链接管理平台，缩短、分享并追踪你的每一个链接。
              完美兼容微信、QQ、小红书等国内主流平台。
            </p>
            <div className="mt-10 flex flex-col sm:flex-row items-center justify-center gap-4">
              <Link
                href="/register"
                className="w-full sm:w-auto rounded-xl bg-indigo-600 px-8 py-4 text-base font-semibold text-white shadow-lg hover:bg-indigo-700 transition hover:shadow-xl"
              >
                免费开始使用
              </Link>
              <Link
                href="/login"
                className="w-full sm:w-auto rounded-xl border border-gray-200 bg-white px-8 py-4 text-base font-medium text-gray-600 hover:bg-gray-50 hover:border-gray-300 transition"
              >
                已有账号？登录
              </Link>
            </div>
          </div>
        </div>
      </section>

      {/* Features Grid */}
      <section className="bg-white border-t border-gray-100">
        <div className="mx-auto max-w-6xl px-4 sm:px-6 py-20 sm:py-28">
          <div className="text-center mb-16">
            <h2 className="text-3xl font-bold text-gray-900 sm:text-4xl">
              一个平台，搞定所有链接
            </h2>
            <p className="mt-4 text-lg text-gray-500">
              为国内环境量身打造的短链接解决方案
            </p>
          </div>

          <div className="grid gap-8 sm:grid-cols-2 lg:grid-cols-3">
            {FEATURES.map(({ icon: Icon, title, desc }) => (
              <div
                key={title}
                className="group rounded-2xl border border-gray-100 bg-white p-8 transition-shadow hover:shadow-lg"
              >
                <div className="mb-4 flex size-12 items-center justify-center rounded-xl bg-indigo-50 text-indigo-600 group-hover:bg-indigo-100 transition-colors">
                  <Icon className="size-6" />
                </div>
                <h3 className="mb-2 text-lg font-semibold text-gray-900">{title}</h3>
                <p className="text-sm leading-relaxed text-gray-500">{desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="bg-gradient-to-br from-indigo-600 to-purple-600">
        <div className="mx-auto max-w-6xl px-4 sm:px-6 py-20 text-center">
          <h2 className="text-3xl font-bold text-white sm:text-4xl">
            准备好管理你的链接了吗？
          </h2>
          <p className="mt-4 text-lg text-indigo-100">
            免费注册，几秒钟内创建你的第一个短链接
          </p>
          <Link
            href="/register"
            className="mt-8 inline-flex rounded-xl bg-white px-10 py-4 text-base font-semibold text-indigo-600 shadow-lg hover:bg-indigo-50 transition"
          >
            立即开始
          </Link>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-gray-100 bg-white">
        <div className="mx-auto max-w-6xl px-4 sm:px-6 py-12 flex flex-col sm:flex-row items-center justify-between gap-4">
          <div className="flex items-center gap-2">
            <div className="flex size-6 items-center justify-center rounded bg-indigo-600">
              <Link2 className="size-3.5 text-white" />
            </div>
            <span className="text-sm font-medium text-gray-700">Kada</span>
          </div>
          <p className="text-xs text-gray-400">
            &copy; {new Date().getFullYear()} Kada. 保留所有权利。
          </p>
        </div>
      </footer>
    </div>
  );
}
