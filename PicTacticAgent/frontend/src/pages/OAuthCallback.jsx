/**
 * OAuth 回调处理页面
 */
import { useEffect, useState, useRef } from 'react';
import { useParams, useSearchParams, useNavigate, Link } from 'react-router-dom';
import { oauthCallback } from '@/services/auth';
import { useLocale } from '@/hooks/useLocale';
import { useAuth } from '@/hooks/useAuth';
import { Loader2, AlertCircle, CheckCircle } from 'lucide-react';

export default function OAuthCallback() {
  const { provider } = useParams();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { t } = useLocale();
  const { updateUser, clearError, isAuthenticated, loading: authLoading } = useAuth();

  const [status, setStatus] = useState('loading'); // loading, success, error
  const [error, setError] = useState(null);
  const processed = useRef(false);

  useEffect(() => {
    // 等 AuthProvider 加载完成后再处理，否则 isAuthenticated 不可靠
    if (authLoading) return;

    if (processed.current) return;
    processed.current = true;

    const handleCallback = async () => {
      const code = searchParams.get('code');
      const state = searchParams.get('state');
      const errorParam = searchParams.get('error');
      const errorDescription = searchParams.get('error_description');

      // 已登录用户通过浏览器历史返回到此页面，直接跳转
      if (isAuthenticated) {
        navigate('/app', { replace: true });
        return;
      }

      // 处理 OAuth 错误
      if (errorParam) {
        setStatus('error');
        setError(errorDescription || errorParam);
        return;
      }

      if (!code) {
        setStatus('error');
        setError(t('auth.oauthNoCode') || '未收到授权码');
        return;
      }

      try {
        const data = await oauthCallback(provider, code, state);
        updateUser(data.user);
        clearError();
        setStatus('success');
        // 延迟跳转，让用户看到成功信息
        setTimeout(() => {
          navigate('/app', { replace: true });
        }, 1000);
      } catch (err) {
        setStatus('error');
        setError(err.message || t('auth.oauthFailed') || 'OAuth 登录失败');
      }
    };

    handleCallback();
  }, [authLoading, provider, searchParams, navigate, t, updateUser, clearError, isAuthenticated]);

  if (status === 'error') {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background p-4">
        <div className="w-full max-w-md text-center space-y-4">
          <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-2xl bg-red-500/10 border border-red-500/20">
            <AlertCircle className="h-8 w-8 text-red-400" />
          </div>
          <h2 className="text-xl font-bold text-foreground">
            {t('auth.oauthError') || '登录失败'}
          </h2>
          <p className="text-sm text-muted-foreground">{error}</p>
          <Link
            to="/auth/login"
            className="inline-block rounded-xl bg-secondary px-6 py-3 text-sm text-foreground hover:bg-secondary/80 transition-colors"
          >
            {t('auth.backToLogin') || '返回登录'}
          </Link>
        </div>
      </div>
    );
  }

  if (status === 'success') {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background p-4">
        <div className="w-full max-w-md text-center space-y-4">
          <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-2xl bg-emerald-500/10 border border-emerald-500/20">
            <CheckCircle className="h-8 w-8 text-emerald-400" />
          </div>
          <h2 className="text-xl font-bold text-foreground">
            {t('auth.oauthSuccess') || '登录成功！'}
          </h2>
          <p className="text-sm text-muted-foreground">
            {t('auth.redirecting') || '正在跳转...'}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background">
      <div className="text-center space-y-4">
        <Loader2 className="mx-auto h-8 w-8 animate-spin text-amber-500" />
        <p className="text-sm text-muted-foreground">
          {t('auth.oauthProcessing') || '正在处理登录...'}
        </p>
      </div>
    </div>
  );
}
