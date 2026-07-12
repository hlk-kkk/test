/**
 * Copyright (c) 2026 relakkes@gmail.com
 *
 * This file is part of MediaCrawlerPro-ContentRemixAgent project.
 * Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/frontend/src/components/Auth/OAuthCallback.jsx
 * GitHub: https://github.com/NanmiCoder
 * Licensed under NON-COMMERCIAL LEARNING LICENSE 1.1
 *
 * 声明：本代码仅供学习和研究目的使用。使用者应遵守以下原则：
 * 1. 不得用于任何商业用途。
 * 2. 使用时应遵守目标平台的使用条款和robots.txt规则。
 * 3. 不得进行大规模爬取或对平台造成运营干扰。
 * 4. 应合理控制请求频率，避免给目标平台带来不必要的负担。
 * 5. 不得用于任何非法或不当的用途。
 *
 * 详细许可条款请参阅项目根目录下的LICENSE文件。
 * 使用本代码即表示您同意遵守上述原则和LICENSE中的所有条款。
 */

/**
 * OAuth 回调处理页面
 */
import React, { useEffect, useRef, useState } from 'react';
import { useParams, useSearchParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { oauthCallback } from '../../services/auth';
import { useAuth } from '../../hooks/useAuth';

export default function OAuthCallback() {
  const { t } = useTranslation();
  const { provider } = useParams();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { updateUser } = useAuth();

  const [error, setError] = useState(null);
  const [processing, setProcessing] = useState(true);
  const calledRef = useRef(false);

  useEffect(() => {
    // 防止 React StrictMode 双重调用（OAuth code 只能使用一次）
    if (calledRef.current) return;
    calledRef.current = true;

    const handleCallback = async () => {
      const code = searchParams.get('code');
      const state = searchParams.get('state');
      const errorParam = searchParams.get('error');

      if (errorParam) {
        const errorDescription = searchParams.get('error_description') || errorParam;
        setError(errorDescription);
        setProcessing(false);
        return;
      }

      if (!code) {
        setError(t('auth.oauthNoCode'));
        setProcessing(false);
        return;
      }

      try {
        const data = await oauthCallback(provider, code, state);
        updateUser(data.user);
        navigate('/', { replace: true });
      } catch (err) {
        setError(err.message);
        setProcessing(false);
      }
    };

    handleCallback();
  }, [provider, searchParams, navigate, t]);

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background-dark px-4">
        <div className="w-full max-w-md text-center">
          {/* 错误图标 */}
          <div className="w-16 h-16 mx-auto mb-6 bg-red-500/10 rounded-full flex items-center justify-center">
            <span className="material-symbols-outlined text-4xl text-red-400">error</span>
          </div>

          <h2 className="text-xl font-semibold text-text-primary mb-2">
            {t('auth.oauthFailed')}
          </h2>

          <p className="text-text-secondary mb-6">{error}</p>

          <button
            onClick={() => navigate('/auth/login', { replace: true })}
            className="px-6 py-2.5 bg-primary hover:bg-primary/90 text-white font-medium rounded-lg transition-colors"
          >
            {t('auth.backToLogin')}
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-background-dark px-4">
      <div className="w-full max-w-md text-center">
        {/* 加载动画 */}
        <div className="w-16 h-16 mx-auto mb-6 relative">
          <div className="absolute inset-0 border-4 border-slate-border rounded-full" />
          <div className="absolute inset-0 border-4 border-primary border-t-transparent rounded-full animate-spin" />
        </div>

        <h2 className="text-xl font-semibold text-text-primary mb-2">
          {t('auth.oauthProcessing')}
        </h2>

        <p className="text-text-secondary">
          {t('auth.oauthPleaseWait')}
        </p>
      </div>
    </div>
  );
}
