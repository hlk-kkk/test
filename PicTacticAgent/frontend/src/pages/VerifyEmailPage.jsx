/**
 * 邮箱验证页面
 */
import { useState, useEffect } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import { AlertCircle, CheckCircle2, Loader2, Mail } from "lucide-react";
import { verifyEmail, resendVerification } from "@/services/auth";
import { useAuth } from "@/hooks/useAuth";
import { useLocale } from "@/hooks/useLocale";
import { AuthShell } from "@/components/auth/AuthShell";

export default function VerifyEmailPage() {
  const { t } = useLocale();
  const { user, updateUser, isAuthenticated } = useAuth();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();

  const [status, setStatus] = useState("idle");
  const [error, setError] = useState(null);
  const [resending, setResending] = useState(false);
  const [resent, setResent] = useState(false);

  const token = searchParams.get("token");

  useEffect(() => {
    const handleVerify = async () => {
      if (!token) return;

      setStatus("loading");
      try {
        await verifyEmail(token);
        setStatus("success");
        if (user) {
          updateUser({ email_verified: true, status: 1 });
        }
        setTimeout(() => {
          navigate("/app", { replace: true });
        }, 1800);
      } catch (err) {
        setStatus("error");
        setError(err.message || t("auth.verifyEmailFailed") || "验证失败");
      }
    };

    handleVerify();
  }, [token, navigate, t, user, updateUser]);

  const handleResend = async () => {
    setResending(true);
    setResent(false);
    setError(null);

    try {
      await resendVerification();
      setResent(true);
    } catch (err) {
      setError(err.message || t("auth.resendFailed") || "发送失败");
    } finally {
      setResending(false);
    }
  };

  if (token) {
    return (
      <AuthShell
        badge={t("auth.verifyYourEmail")}
        title={
          status === "success"
            ? t("auth.emailVerified")
            : status === "error"
              ? t("auth.verifyFailed")
              : t("auth.verifying")
        }
        subtitle={status === "success" ? t("auth.redirecting") : undefined}
      >
        {status === "loading" ? (
          <div className="rounded-2xl border border-border/70 bg-card/70 p-6 text-center">
            <Loader2 className="mx-auto h-8 w-8 animate-spin text-primary" />
            <p className="mt-3 text-sm text-muted-foreground">{t("auth.verifying")}</p>
          </div>
        ) : null}

        {status === "success" ? (
          <div className="rounded-2xl border border-emerald-500/35 bg-emerald-500/10 p-6 text-center">
            <CheckCircle2 className="mx-auto h-8 w-8 text-emerald-400" />
            <p className="mt-3 text-sm text-emerald-200">{t("auth.redirecting")}</p>
          </div>
        ) : null}

        {status === "error" ? (
          <div className="space-y-4">
            <div className="rounded-2xl border border-destructive/35 bg-destructive/10 p-5 text-center">
              <AlertCircle className="mx-auto h-8 w-8 text-destructive" />
              <p className="mt-3 text-sm text-destructive">{error}</p>
            </div>

            {isAuthenticated ? (
              <button
                onClick={handleResend}
                disabled={resending}
                className="brand-btn-secondary w-full justify-center py-3.5 text-sm font-semibold disabled:cursor-not-allowed disabled:opacity-70"
              >
                {resending ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
                {t("auth.resendVerification")}
              </button>
            ) : null}
          </div>
        ) : null}
      </AuthShell>
    );
  }

  return (
    <AuthShell
      badge={t("auth.verifyYourEmail")}
      title={t("auth.verifyYourEmail")}
      subtitle={t("auth.verificationSent")}
    >
      <div className="rounded-2xl border border-primary/30 bg-primary/10 p-5 text-center">
        <Mail className="mx-auto h-8 w-8 text-primary" />
        <p className="mt-3 text-sm leading-relaxed text-foreground/90">{t("auth.verificationSent")}</p>
      </div>

      {resent ? (
        <div className="rounded-2xl border border-emerald-500/35 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">
          {t("auth.verificationResent")}
        </div>
      ) : null}

      {error ? (
        <div className="rounded-2xl border border-destructive/35 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          {error}
        </div>
      ) : null}

      {isAuthenticated ? (
        <button
          onClick={handleResend}
          disabled={resending}
          className="brand-btn-primary w-full py-3.5 text-base font-semibold disabled:cursor-not-allowed disabled:opacity-70"
        >
          {resending ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
          {t("auth.resendVerification")}
        </button>
      ) : null}

      <p className="text-center text-xs text-muted-foreground">{t("auth.checkSpam")}</p>
    </AuthShell>
  );
}
