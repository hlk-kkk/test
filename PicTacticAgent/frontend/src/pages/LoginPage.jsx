/**
 * 登录页面
 */
import { useState } from "react";
import { Link } from "react-router-dom";
import { Loader2 } from "lucide-react";
import { useAuth } from "@/hooks/useAuth";
import { useLocale } from "@/hooks/useLocale";
import { getOAuthAuthorizeUrl } from "@/services/auth";
import { AuthShell } from "@/components/auth/AuthShell";
import { OAuthButtons } from "@/components/auth/OAuthButtons";

export default function LoginPage() {
  const { t } = useLocale();
  const { login, oauthProviders, error, clearError } = useAuth();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [oauthLoading, setOauthLoading] = useState(null);

  const handleSubmit = async (e) => {
    e.preventDefault();
    clearError();
    setLoading(true);

    try {
      await login(email, password);
    } finally {
      setLoading(false);
    }
  };

  const handleOAuthLogin = async (provider) => {
    setOauthLoading(provider);
    clearError();

    try {
      const data = await getOAuthAuthorizeUrl(provider);
      window.location.href = data.authorize_url;
    } catch (err) {
      console.error("OAuth error:", err);
      setOauthLoading(null);
    }
  };

  return (
    <AuthShell
      badge={t("landing.badge")}
      title={t("auth.loginTitle") || "欢迎回来"}
      subtitle={t("auth.loginSubtitle") || "登录你的账号"}
      footer={
        <p className="text-center text-sm text-muted-foreground">
          {t("auth.noAccount")} {" "}
          <Link to="/auth/register" className="font-semibold text-primary transition-colors hover:text-primary/80">
            {t("auth.register")}
          </Link>
        </p>
      }
    >
      <OAuthButtons
        oauthProviders={oauthProviders}
        oauthLoading={oauthLoading}
        onOAuthLogin={handleOAuthLogin}
      />

      {error && (
        <div className="rounded-2xl border border-destructive/35 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="space-y-1.5">
          <label htmlFor="login-email" className="block text-sm font-medium text-foreground/85">
            {t("auth.email")}
          </label>
          <input
            id="login-email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            className="brand-input"
            placeholder="you@example.com"
            autoComplete="email"
          />
        </div>

        <div className="space-y-1.5">
          <div className="flex items-center justify-between gap-3">
            <label htmlFor="login-password" className="block text-sm font-medium text-foreground/85">
              {t("auth.password")}
            </label>
            <Link
              to="/auth/forgot-password"
              className="text-xs font-semibold text-primary transition-colors hover:text-primary/80"
            >
              {t("auth.forgotPassword")}
            </Link>
          </div>
          <input
            id="login-password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            className="brand-input"
            placeholder="••••••••"
            autoComplete="current-password"
          />
        </div>

        <button
          type="submit"
          disabled={loading}
          className="brand-btn-primary w-full py-3.5 text-base font-semibold disabled:cursor-not-allowed disabled:opacity-70"
        >
          {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
          {t("auth.loginButton")}
        </button>
      </form>
    </AuthShell>
  );
}
