import Link from "next/link";

export default function HomePage() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-gradient-to-br from-indigo-500 to-purple-600 px-4">
      <div className="text-center text-white">
        <h1 className="text-5xl font-bold tracking-tight">Kada</h1>
        <p className="mt-4 text-xl text-indigo-100">智能短链接管理平台</p>
        <p className="mt-2 text-sm text-indigo-200">兼容微信 · QQ · 小红书 · 短信</p>
        <div className="mt-8 flex gap-4 justify-center">
          <Link
            href="/login"
            className="rounded-lg bg-white px-8 py-3 font-medium text-indigo-600 shadow-lg hover:bg-indigo-50 transition"
          >
            登录
          </Link>
          <Link
            href="/register"
            className="rounded-lg border-2 border-white px-8 py-3 font-medium text-white hover:bg-white/10 transition"
          >
            注册
          </Link>
        </div>
      </div>
    </div>
  );
}
