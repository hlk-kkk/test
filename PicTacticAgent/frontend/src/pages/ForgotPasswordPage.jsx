/**
 * 忘记密码页面
 */
import { useState } from "react";
import { Link } from "react-router-dom";
import { ArrowLeft, Loader2, MailCheck } from "lucide-react";
import { forgotPassword } from "@/services/auth";
import { useLocale } from "@/hooks/useLocale";
import { AuthShell } from "@/components/auth/AuthShell";

export default function ForgotPasswordPage() {
  const { t } = useLocale();

  const [email, setEmail] = useState("");
  const [loading, setLoading] = useState(false);
  const [sent, setSent] = useState(false);
  const [error, setError] = useState(null);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      await forgotPassword(email);
      setSent(true);
    } catch (err) {
      setError(err.message || t("auth.forgotPasswordFailed") || "发送失败，请稍后重试");
    } finally {
      setLoading(false);
    }
  };

  if (sent) {
    return (
      <AuthShell
        badge={t("auth.forgotPasswordTitle")}
        title={t("auth.emailSent")}
        subtitle={t("auth.checkEmail")}
        footer={
          <p className="text-center">
            <Link
              to="/auth/login"
              className="inline-flex items-center gap-2 text-sm font-semibold text-primary transition-colors hover:text-primary/80"
            >
              <ArrowLeft size={14} /> {t("auth.backToLogin")}
            </Link>
          </p>
        }
      >
        <div className="flex flex-col items-center gap-3 rounded-2xl border border-emerald-500/30 bg-emerald-500/10 p-6 text-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-emerald-500/20 text-emerald-400">
            <MailCheck className="h-6 w-6" />
          </div>
          <p className="text-sm leading-relaxed text-emerald-200">{t("auth.checkEmail")}</p>
        </div>
      </AuthShell>
    );
  }

  return (
    <AuthShell
      badge={t("auth.forgotPasswordTitle")}
      title={t("auth.forgotPasswordTitle") || "重置密码"}
      subtitle={t("auth.forgotPasswordDescription") || "输入您的邮箱，我们将发送密码重置链接。"}
      footer={
        <p className="text-center">
          <Link
            to="/auth/login"
            className="inline-flex items-center gap-2 text-sm font-semibold text-muted-foreground transition-colors hover:text-foreground"
          >
            <ArrowLeft size={14} /> {t("auth.backToLogin")}
          </Link>
        </p>
      }
    >
      {error && (
        <div className="rounded-2xl border border-destructive/35 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="space-y-1.5">
          <label htmlFor="forgot-email" className="block text-sm font-medium text-foreground/85">
            {t("auth.email")}
          </label>
          <input
            id="forgot-email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            className="brand-input"
            placeholder="you@example.com"
            autoComplete="email"
          />
        </div>

        <button
          type="submit"
          disabled={loading}
          className="brand-btn-primary w-full py-3.5 text-base font-semibold disabled:cursor-not-allowed disabled:opacity-70"
        >
          {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
          {t("auth.sendResetLink")}
        </button>
      </form>
    </AuthShell>
  );
}
