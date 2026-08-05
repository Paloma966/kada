"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import { authAPI } from "@/lib/api";
import { setToken, setUser } from "@/lib/auth";
import { AuthCard } from "@/components/auth/AuthCard";
import { FormField, inputBase, fieldState } from "@/components/auth/FormField";
import { PasswordInput } from "@/components/auth/PasswordInput";

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export default function LoginPage() {
  const router = useRouter();
  const [tab, setTab] = useState<"phone" | "email">("phone");
  const [loading, setLoading] = useState(false);

  // 手机号登录
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState("");
  const [codeSent, setCodeSent] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const [phoneError, setPhoneError] = useState("");
  const [codeError, setCodeError] = useState("");

  // 邮箱登录
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [emailError, setEmailError] = useState("");
  const [passwordError, setPasswordError] = useState("");

  const phoneValid = phone.length === 11;

  const handleSendCode = async () => {
    if (phone.length !== 11) {
      setPhoneError("请输入 11 位手机号");
      return;
    }
    setPhoneError("");
    try {
      await authAPI.sendSMSCode(phone);
      setCodeSent(true);
      toast.success("验证码已发送");
      setCountdown(60);
      const timer = setInterval(() => {
        setCountdown((c) => {
          if (c <= 1) {
            clearInterval(timer);
            return 0;
          }
          return c - 1;
        });
      }, 1000);
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "发送失败");
    }
  };

  const handlePhoneLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    let ok = true;
    if (phone.length !== 11) {
      setPhoneError("请输入 11 位手机号");
      ok = false;
    } else {
      setPhoneError("");
    }
    if (code.length !== 6) {
      setCodeError("请输入 6 位验证码");
      ok = false;
    } else {
      setCodeError("");
    }
    if (!ok) return;

    setLoading(true);
    try {
      const data = await authAPI.loginByPhone(phone, code);
      setToken(data.token);
      setUser(data.user);
      toast.success("登录成功");
      router.push("/dashboard");
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "登录失败");
    } finally {
      setLoading(false);
    }
  };

  const handleEmailLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    let ok = true;
    if (!EMAIL_RE.test(email)) {
      setEmailError("请输入有效的邮箱地址");
      ok = false;
    } else {
      setEmailError("");
    }
    if (!password) {
      setPasswordError("请输入密码");
      ok = false;
    } else {
      setPasswordError("");
    }
    if (!ok) return;

    setLoading(true);
    try {
      const data = await authAPI.loginByEmail(email, password);
      setToken(data.token);
      setUser(data.user);
      toast.success("登录成功");
      router.push("/dashboard");
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "登录失败");
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthCard
      title="登录 Kada"
      subtitle="智能短链接管理平台"
      footer={
        <>
          还没有账号？{" "}
          <Link href="/register" className="font-medium text-indigo-600 hover:text-indigo-500">
            立即注册
          </Link>
        </>
      }
    >
      {/* 手机号 / 邮箱 分段切换 */}
      <div className="mb-6 grid grid-cols-2 gap-1 rounded-lg bg-neutral-100 p-1">
        <button
          type="button"
          onClick={() => setTab("phone")}
          className={`rounded-md py-2 text-sm font-medium transition ${
            tab === "phone"
              ? "bg-white text-neutral-900 shadow-sm"
              : "text-neutral-500 hover:text-neutral-700"
          }`}
        >
          手机号登录
        </button>
        <button
          type="button"
          onClick={() => setTab("email")}
          className={`rounded-md py-2 text-sm font-medium transition ${
            tab === "email"
              ? "bg-white text-neutral-900 shadow-sm"
              : "text-neutral-500 hover:text-neutral-700"
          }`}
        >
          邮箱登录
        </button>
      </div>

      {tab === "phone" ? (
        <form onSubmit={handlePhoneLogin} className="space-y-4" noValidate>
          <FormField id="phone" label="手机号" error={phoneError}>
            <input
              id="phone"
              type="tel"
              value={phone}
              onChange={(e) => setPhone(e.target.value.replace(/\D/g, "").slice(0, 11))}
              className={`${inputBase} ${fieldState(!!phoneError)}`}
              placeholder="请输入 11 位手机号"
              required
            />
          </FormField>
          <FormField id="code" label="验证码" error={codeError}>
            <div className="flex gap-3">
              <input
                id="code"
                type="text"
                value={code}
                onChange={(e) => setCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
                className={`${inputBase} ${fieldState(!!codeError)}`}
                placeholder="6 位验证码"
                required
              />
              <button
                type="button"
                onClick={handleSendCode}
                disabled={countdown > 0 || !phoneValid}
                className="shrink-0 rounded-lg bg-indigo-50 px-4 py-2.5 text-sm font-medium text-indigo-600 transition hover:bg-indigo-100 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {countdown > 0 ? `${countdown}s` : codeSent ? "重新发送" : "获取验证码"}
              </button>
            </div>
          </FormField>
          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-lg bg-indigo-600 py-2.5 text-sm font-medium text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {loading ? "登录中..." : "登录"}
          </button>
        </form>
      ) : (
        <form onSubmit={handleEmailLogin} className="space-y-4" noValidate>
          <FormField id="email" label="邮箱" error={emailError}>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className={`${inputBase} ${fieldState(!!emailError)}`}
              placeholder="your@email.com"
              required
            />
          </FormField>
          <FormField id="password" label="密码" error={passwordError}>
            <PasswordInput
              id="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={fieldState(!!passwordError)}
              placeholder="请输入密码"
              required
            />
          </FormField>
          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-lg bg-indigo-600 py-2.5 text-sm font-medium text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {loading ? "登录中..." : "登录"}
          </button>
        </form>
      )}
    </AuthCard>
  );
}
