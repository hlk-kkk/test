/**
 * 认证服务模块
 * 提供用户认证相关的 API 接口
 */

const AUTH_API_BASE = '/api/v1/auth';
const API_BASE_URL = import.meta.env.VITE_API_URL !== undefined ? import.meta.env.VITE_API_URL : "http://localhost:8019";

// ========== Token 存储 ==========

const TOKEN_KEY = 'pictactic_access_token';
const REFRESH_TOKEN_KEY = 'pictactic_refresh_token';
const USER_KEY = 'pictactic_user';

/**
 * 获取 Access Token
 */
export function getAccessToken() {
  return localStorage.getItem(TOKEN_KEY);
}

/**
 * 获取 Refresh Token
 */
export function getRefreshToken() {
  return localStorage.getItem(REFRESH_TOKEN_KEY);
}

/**
 * 保存 Token
 */
export function saveTokens(accessToken, refreshToken) {
  localStorage.setItem(TOKEN_KEY, accessToken);
  if (refreshToken) {
    localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
  }
}

/**
 * 清除 Token
 */
export function clearTokens() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
}

/**
 * 获取缓存的用户信息
 */
export function getCachedUser() {
  const userStr = localStorage.getItem(USER_KEY);
  if (userStr) {
    try {
      return JSON.parse(userStr);
    } catch {
      return null;
    }
  }
  return null;
}

/**
 * 保存用户信息
 */
export function saveUser(user) {
  localStorage.setItem(USER_KEY, JSON.stringify(user));
}

// ========== HTTP 请求 ==========

function getLang() {
  return localStorage.getItem("locale") || "zh";
}

/**
 * 创建带有语言和认证头的 fetch 函数
 */
async function fetchAuth(url, options = {}) {
  const language = getLang();
  const token = getAccessToken();

  const headers = {
    'Accept-Language': language,
    'Content-Type': 'application/json',
    ...(options.headers || {}),
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE_URL}${url}`, {
    ...options,
    headers,
  });

  return response;
}

/**
 * 处理 API 响应
 */
async function handleResponse(response) {
  const data = await response.json();

  if (!response.ok) {
    const error = new Error(data.detail || '请求失败');
    error.status = response.status;
    error.data = data;
    throw error;
  }

  return data;
}

// ========== 认证 API ==========

/**
 * 用户注册
 * @param {string} email - 邮箱
 * @param {string} password - 密码
 * @param {string} displayName - 显示名称（可选）
 */
export async function register(email, password, displayName = null) {
  const response = await fetchAuth(`${AUTH_API_BASE}/register`, {
    method: 'POST',
    body: JSON.stringify({
      email,
      password,
      display_name: displayName,
    }),
  });

  const data = await handleResponse(response);

  // 保存 Token 和用户信息
  saveTokens(data.access_token, data.refresh_token);
  saveUser(data.user);

  return data;
}

/**
 * 用户登录
 * @param {string} email - 邮箱
 * @param {string} password - 密码
 */
export async function login(email, password) {
  const response = await fetchAuth(`${AUTH_API_BASE}/login`, {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });

  const data = await handleResponse(response);

  // 保存 Token 和用户信息
  saveTokens(data.access_token, data.refresh_token);
  saveUser(data.user);

  return data;
}

/**
 * 用户登出
 */
export async function logout() {
  const refreshToken = getRefreshToken();

  if (refreshToken) {
    try {
      await fetchAuth(`${AUTH_API_BASE}/logout`, {
        method: 'POST',
        body: JSON.stringify({ refresh_token: refreshToken }),
      });
    } catch (e) {
      // 忽略登出错误
      console.warn('Logout request failed:', e);
    }
  }

  // 清除本地存储
  clearTokens();
}

/**
 * 刷新 Token（带并发锁）
 * 多个请求同时触发 401 时，只有第一个会真正调用后端 refresh，其他复用同一个 Promise
 */
let _refreshPromise = null;

export async function refreshTokens() {
  if (_refreshPromise) {
    return _refreshPromise;
  }

  _refreshPromise = _doRefreshTokens();
  try {
    return await _refreshPromise;
  } finally {
    _refreshPromise = null;
  }
}

async function _doRefreshTokens() {
  const refreshToken = getRefreshToken();

  if (!refreshToken) {
    throw new Error('No refresh token');
  }

  const response = await fetch(`${API_BASE_URL}${AUTH_API_BASE}/refresh`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Accept-Language': getLang(),
    },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });

  const data = await handleResponse(response);

  // 保存新 Token
  saveTokens(data.access_token, data.refresh_token);
  saveUser(data.user);

  return data;
}

/**
 * 获取当前用户信息
 */
export async function getCurrentUser() {
  const response = await fetchAuth(`${AUTH_API_BASE}/me`);
  const data = await handleResponse(response);
  saveUser(data);
  return data;
}

// ========== OAuth API ==========

/**
 * 获取可用的 OAuth 提供商
 */
export async function getOAuthProviders() {
  const response = await fetchAuth(`${AUTH_API_BASE}/oauth/providers`);
  return handleResponse(response);
}

/**
 * 获取 OAuth 授权 URL
 * @param {string} provider - 提供商 (github, google)
 */
export async function getOAuthAuthorizeUrl(provider) {
  const response = await fetchAuth(`${AUTH_API_BASE}/oauth/${provider}/authorize`);
  return handleResponse(response);
}

/**
 * OAuth 回调处理
 * @param {string} provider - 提供商 (github, google)
 * @param {string} code - 授权码
 * @param {string} state - state 参数
 */
export async function oauthCallback(provider, code, state = null) {
  const response = await fetchAuth(`${AUTH_API_BASE}/oauth/${provider}/callback`, {
    method: 'POST',
    body: JSON.stringify({ code, state }),
  });

  const data = await handleResponse(response);

  // 保存 Token 和用户信息
  saveTokens(data.access_token, data.refresh_token);
  saveUser(data.user);

  return data;
}

// ========== 邮箱验证 API ==========

/**
 * 验证邮箱
 * @param {string} token - 验证令牌
 */
export async function verifyEmail(token) {
  const response = await fetchAuth(`${AUTH_API_BASE}/verify-email`, {
    method: 'POST',
    body: JSON.stringify({ token }),
  });
  return handleResponse(response);
}

/**
 * 重新发送验证邮件
 */
export async function resendVerification() {
  const response = await fetchAuth(`${AUTH_API_BASE}/resend-verification`, {
    method: 'POST',
  });
  return handleResponse(response);
}

// ========== 密码重置 API ==========

/**
 * 请求重置密码
 * @param {string} email - 邮箱
 */
export async function forgotPassword(email) {
  const response = await fetchAuth(`${AUTH_API_BASE}/forgot-password`, {
    method: 'POST',
    body: JSON.stringify({ email }),
  });
  return handleResponse(response);
}

/**
 * 重置密码
 * @param {string} token - 重置令牌
 * @param {string} newPassword - 新密码
 */
export async function resetPassword(token, newPassword) {
  const response = await fetchAuth(`${AUTH_API_BASE}/reset-password`, {
    method: 'POST',
    body: JSON.stringify({
      token,
      new_password: newPassword,
    }),
  });
  return handleResponse(response);
}

// ========== 跨标签页 Token 同步 ==========

/**
 * 解析 JWT payload 获取过期时间（不验证签名，仅解码）
 * @returns {number|null} 过期时间戳（毫秒），解析失败返回 null
 */
export function getTokenExpiry(token) {
  try {
    const base64Url = token.split('.')[1];
    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
    const payload = JSON.parse(atob(base64));
    return typeof payload.exp === 'number' && isFinite(payload.exp)
      ? payload.exp * 1000
      : null;
  } catch {
    return null;
  }
}

// 监听其他标签页的 storage 变化（防止 HMR 重复注册）
if (typeof window !== 'undefined' && !window.__pictactic_storage_listener__) {
  window.__pictactic_storage_listener__ = true;
  window.addEventListener('storage', (e) => {
    if (e.key === TOKEN_KEY && !e.newValue) {
      // 其他标签页清除了 token（登出），跳转登录页
      window.location.href = '/auth/login';
    }
    // 其他标签页刷新了 token，当前标签页下次请求会自动读取新值（来自 localStorage）
  });
}

// ========== 带认证的 Fetch ==========

/**
 * 带认证的 fetch 函数
 * 自动添加 Authorization 头，支持 Token 自动刷新
 */
export async function fetchWithAuth(url, options = {}) {
  const language = getLang();
  let token = getAccessToken();

  const makeRequest = async (accessToken) => {
    const headers = {
      'Accept-Language': language,
      ...(options.headers || {}),
    };

    if (accessToken) {
      headers['Authorization'] = `Bearer ${accessToken}`;
    }

    return fetch(url, {
      ...options,
      headers,
    });
  };

  let response = await makeRequest(token);

  // 如果 401，尝试刷新 Token
  if (response.status === 401 && getRefreshToken()) {
    try {
      const refreshData = await refreshTokens();
      token = refreshData.access_token;
      response = await makeRequest(token);
    } catch (e) {
      // 刷新失败，清除 Token
      clearTokens();
      throw new Error('Session expired');
    }
  }

  return response;
}
