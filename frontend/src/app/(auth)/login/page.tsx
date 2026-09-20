"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { RefreshCw } from "lucide-react";
import { toast } from "sonner";
import useSWR from "swr";
import { authAPI } from "@/lib/api";
import { setToken, setUser } from "@/lib/auth";
import { AuthCard } from "@/components/auth/AuthCard";
import { FormField, inputBase, fieldState } from "@/components/auth/FormField";
import { LanguageSwitcher } from "@/components/LanguageSwitcher";
import { useT } from "@/lib/i18n";

/**
 * Sign-in: phone number + SMS code, and nothing else.
 *
 * The email/password tab and the separate sign-up page are gone. A phone number that has never been seen
 * is registered on the spot, so "sign in" and "sign up" are the same action - which is why the page says
 * so instead of hiding it behind a link to a second form.
 *
 * The graphical challenge is not decoration: the SMS endpoint refuses any request that does not carry a
 * solved one, because an unauthenticated endpoint that makes the server send paid messages is the single
 * most abusable thing in this app.
 */
export default function LoginPage() {
  const router = useRouter();
  const t = useT();
  const [loading, setLoading] = useState(false);

  const [phone, setPhone] = useState("");
  const [code, setCode] = useState("");
  const [codeSent, setCodeSent] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const [sending, setSending] = useState(false);

  // The challenge is fetched with SWR rather than from an effect: an effect that calls setState on mount
  // is precisely the cascading render the react-hooks rule warns about, and SWR already expresses "fetch
  // this once, and let me ask for it again".
  //
  // Revalidation is off on purpose. The challenge is one-time and tied to its id, so the background
  // refresh a window focus would trigger would replace an image the user is halfway through reading with
  // a different one, whose answer they cannot know.
  const {
    data: challenge,
    error: challengeError,
    mutate: refreshCaptcha,
  } = useSWR("auth-captcha", () => authAPI.captcha(), {
    revalidateOnFocus: false,
    revalidateOnReconnect: false,
    // One blip must not take the whole form down with it. Deploying restarts kada-api, and nginx can
    // answer the first request after that with a 502 from a pooled connection the old process owned.
    // With retries switched off, that single failure left the field in its error state until the user
    // clicked the image - which is indistinguishable from a login page that simply does not work. SWR
    // only calls onErrorRetry at all when shouldRetryOnError is left at its default, so that line is
    // gone rather than unused.
    //
    // Bounded on purpose: this is an unauthenticated endpoint, so three attempts spaced over ~3s is
    // enough to ride out a restart and not enough to hammer a service that is genuinely down.
    onErrorRetry: (_err, _key, _config, revalidate, { retryCount }) => {
      if (retryCount > 3) return;
      setTimeout(() => revalidate({ retryCount }), 500 * retryCount);
    },
    // No dedupe window: the refresh after a send attempt MUST reach the server. The server consumes the
    // challenge on every attempt, so a suppressed refresh would leave the user looking at an image that
    // can no longer be answered - and it would only happen when they clicked quickly, which is the worst
    // way to find that out.
    dedupingInterval: 0,
  });

  // The field can only say "could not load"; it cannot print a 502 or an unreachable API without turning
  // the login form into a debug console. The reason still has to exist somewhere, so it goes to the
  // browser console - which is exactly where the last investigation looked and found nothing at all.
  useEffect(() => {
    if (challengeError) {
      console.error("kada: captcha challenge failed", challengeError);
    }
  }, [challengeError]);

  const captchaId = challenge?.captcha_id ?? "";
  const captchaImage = challenge?.image ?? "";
  const [captcha, setCaptcha] = useState("");

  const [agreed, setAgreed] = useState(false);

  const [phoneError, setPhoneError] = useState("");
  const [captchaError, setCaptchaError] = useState("");
  const [codeError, setCodeError] = useState("");
  const [agreeError, setAgreeError] = useState("");

  const phoneValid = phone.length === 11;

  // The resend countdown. The timer is cleared on unmount so a page that is gone does not keep one.
  useEffect(() => {
    if (countdown <= 0) return;
    const timer = setTimeout(() => setCountdown((c) => c - 1), 1000);
    return () => clearTimeout(timer);
  }, [countdown]);

  const validate = () => {
    let ok = true;
    if (!phoneValid) {
      setPhoneError(t("请输入 11 位手机号"));
      ok = false;
    } else {
      setPhoneError("");
    }
    if (!agreed) {
      setAgreeError(t("请先阅读并同意隐私政策"));
      ok = false;
    } else {
      setAgreeError("");
    }
    return ok;
  };

  const handleSendCode = async () => {
    let ok = true;
    if (!phoneValid) {
      setPhoneError(t("请输入 11 位手机号"));
      ok = false;
    } else {
      setPhoneError("");
    }
    if (!captcha.trim()) {
      setCaptchaError(t("请输入图形验证码"));
      ok = false;
    } else {
      setCaptchaError("");
    }
    if (!agreed) {
      setAgreeError(t("请先阅读并同意隐私政策"));
      ok = false;
    } else {
      setAgreeError("");
    }
    if (!ok) return;

    setSending(true);
    try {
      await authAPI.sendSMSCode(phone, captchaId, captcha.trim());
      setCodeSent(true);
      toast.success(t("验证码已发送"));
      setCountdown(60);
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t("发送失败"));
    } finally {
      setSending(false);
      // The server burns the challenge whether the send succeeded or was refused, so a fresh one is the
      // only state worth leaving behind.
      refreshCaptcha();
    }
  };

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    let ok = validate();
    if (code.length !== 6) {
      setCodeError(t("请输入 6 位验证码"));
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
      toast.success(t("登录成功"));
      router.push("/dashboard");
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t("登录失败"));
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <div className="fixed right-4 top-4 z-50">
        <LanguageSwitcher onDark />
      </div>
      <AuthCard
        title={t("登录 Kada")}
        subtitle={t("智能短链接管理平台")}
        footer={<>{t("未注册的手机号将自动创建账号")}</>}
      >
        <form onSubmit={handleLogin} className="space-y-4" noValidate>
          <FormField id="phone" label={t("手机号")} error={phoneError}>
            <input
              id="phone"
              type="tel"
              value={phone}
              onChange={(e) => setPhone(e.target.value.replace(/\D/g, "").slice(0, 11))}
              className={`${inputBase} ${fieldState(!!phoneError)}`}
              placeholder={t("请输入 11 位手机号")}
              autoComplete="tel"
              required
            />
          </FormField>

          <FormField
            id="captcha"
            label={t("图形验证码")}
            error={captchaError || (challengeError ? t("图形验证码加载失败，请点击刷新") : "")}
          >
            <div className="flex gap-3">
              <input
                id="captcha"
                type="text"
                value={captcha}
                onChange={(e) => setCaptcha(e.target.value.slice(0, 6))}
                className={`${inputBase} ${fieldState(!!captchaError || !!challengeError)}`}
                placeholder={t("输入图中字符")}
                autoComplete="off"
                required
              />
              {/* The image is the refresh control: clicking the picture to get another one is what everyone
                  tries first, and it keeps the affordance the same size as the challenge itself. */}
              <button
                type="button"
                onClick={() => refreshCaptcha()}
                title={t("看不清？换一张")}
                aria-label={t("看不清？换一张")}
                className="relative h-[42px] w-[120px] shrink-0 overflow-hidden rounded-lg border border-white/20 bg-white/10 transition hover:bg-white/20"
              >
                {captchaImage ? (
                  // Painted as a background rather than with <img>: the challenge is an inline SVG data URI
                  // generated per request, and next/image would route a one-time secret through an
                  // optimising cache. A background image needs no lint suppression and no loader.
                  <span
                    className="absolute inset-0 bg-contain bg-center bg-no-repeat"
                    style={{ backgroundImage: `url("${captchaImage}")` }}
                  />
                ) : (
                  <RefreshCw className="size-4 animate-spin text-indigo-200" />
                )}
              </button>
            </div>
          </FormField>

          <FormField id="code" label={t("验证码")} error={codeError}>
            <div className="flex gap-3">
              <input
                id="code"
                type="text"
                inputMode="numeric"
                value={code}
                onChange={(e) => setCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
                className={`${inputBase} ${fieldState(!!codeError)}`}
                placeholder={t("6 位验证码")}
                autoComplete="one-time-code"
                required
              />
              <button
                type="button"
                onClick={handleSendCode}
                // No challenge, no send: without an id to send back the server would refuse the request
                // anyway, and a button that visibly does nothing is worse than a disabled one.
                disabled={countdown > 0 || !phoneValid || !captchaId || sending}
                className="shrink-0 rounded-lg bg-indigo-500/20 px-4 py-2.5 text-sm font-medium text-indigo-200 transition hover:bg-indigo-500/30 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {countdown > 0 ? `${countdown}s` : codeSent ? t("重新发送") : t("获取验证码")}
              </button>
            </div>
          </FormField>

          <div>
            <label className="flex items-start gap-2 text-xs text-neutral-300">
              <input
                type="checkbox"
                checked={agreed}
                onChange={(e) => {
                  setAgreed(e.target.checked);
                  if (e.target.checked) setAgreeError("");
                }}
                className="mt-0.5 size-3.5 shrink-0 cursor-pointer rounded border-white/30 bg-white/10 accent-indigo-500"
              />
              <span>
                {t("我已阅读并同意")}
                <Link href="/privacy" target="_blank" className="text-indigo-300 underline hover:text-indigo-200">
                  {t("《隐私政策》")}
                </Link>
              </span>
            </label>
            {agreeError && <p className="mt-1 text-xs text-red-300">{agreeError}</p>}
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-lg bg-indigo-600 py-2.5 text-sm font-medium text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {loading ? t("登录中...") : t("登录 / 注册")}
          </button>
        </form>
      </AuthCard>
    </>
  );
}
