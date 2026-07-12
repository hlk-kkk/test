/**
 * 重置密码页面
 */
import { useState, useEffect } from "react";
import { Link, useSearchParams, useNavigate } from "react-router-dom";
import { AlertCircle, CheckCircle2, Loader2 } from "lucide-react";
import { resetPassword } from "@/services/auth";
import { useLocale } from "@/hooks/useLocale";
import { AuthShell } from "@/components/auth/AuthShell";

export default function ResetPasswordPage() {
  const { t } = useLocale();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();

  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState(null);

  const token = searchParams.get("token");

  useEffect(() => {
    if (!token) {
      setError(t("auth.invalidResetLink") || "无效的重置链接");
    }
  }, [token, t]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError(null);

    if (password !== confirmPassword) {
      setError(t("auth.passwordMismatch") || "两次输入的密码不一致");
      return;
    }

    if (password.length < 6) {
      setError(t("auth.passwordTooShort") || "密码长度至少为 6 位");
      return;
    }

    setLoading(true);

    try {
      await resetPassword(token, password);
      setSuccess(true);
      setTimeout(() => {
        navigate("/auth/login", { replace: true });
      }, 2000);
    } catch (err) {
      setError(err.message || t("auth.resetPasswordFailed") || "重置密码失败");
    } finally {
      setLoading(false);
    }
  };

  if (!token) {
    return (
      <AuthShell
        badge={t("auth.invalidLink")}
        title={t("auth.invalidLink")}
        subtitle={t("auth.invalidResetLinkDescription")}
      >
        <div className="rounded-2xl border border-destructive/35 bg-destructive/10 p-5 text-center">
          <AlertCircle className="mx-auto h-8 w-8 text-destructive" />
          <p className="mt-3 text-sm text-destructive">{error}</p>
        </div>

        <Link to="/auth/forgot-password" className="brand-btn-secondary w-full justify-center py-3.5 text-sm font-semibold">
          {t("auth.requestNewLink")}
        </Link>
      </AuthShell>
    );
  }

  if (success) {
    return (
      <AuthShell
        badge={t("auth.resetPassword")}
        title={t("auth.passwordResetSuccess")}
        subtitle={t("auth.redirectToLogin")}
      >
        <div className="rounded-2xl border border-emerald-500/35 bg-emerald-500/10 p-6 text-center">
          <CheckCircle2 className="mx-auto h-8 w-8 text-emerald-400" />
          <p className="mt-3 text-sm text-emerald-200">{t("auth.redirectToLogin")}</p>
        </div>
      </AuthShell>
    );
  }

  return (
    <AuthShell
      badge={t("auth.resetPassword")}
      title={t("auth.resetPassword") || "重置密码"}
      subtitle={t("auth.resetPasswordSubtitle") || "设置新密码"}
    >
      {error && (
        <div className="rounded-2xl border border-destructive/35 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="space-y-1.5">
          <label htmlFor="new-password" className="block text-sm font-medium text-foreground/85">
            {t("auth.newPassword")}
          </label>
          <input
            id="new-password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            className="brand-input"
            placeholder={t("auth.passwordPlaceholder")}
            autoComplete="new-password"
          />
        </div>

        <div className="space-y-1.5">
          <label htmlFor="confirm-new-password" className="block text-sm font-medium text-foreground/85">
            {t("auth.confirmNewPassword")}
          </label>
          <input
            id="confirm-new-password"
            type="password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            required
            className="brand-input"
            placeholder={t("auth.confirmPasswordPlaceholder")}
            autoComplete="new-password"
          />
        </div>

        <button
          type="submit"
          disabled={loading}
          className="brand-btn-primary w-full py-3.5 text-base font-semibold disabled:cursor-not-allowed disabled:opacity-70"
        >
          {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
          {t("auth.setNewPassword")}
        </button>
      </form>
    </AuthShell>
  );
}
