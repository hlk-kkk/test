/**
 * Copyright (c) 2026 relakkes@gmail.com
 *
 * This file is part of MediaCrawlerPro-ContentRemixAgent project.
 * Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/frontend/src/services/api.js
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
 * API 服务模块
 * 提供与后端通信的所有 API 接口
 */
import { createSSEStream } from './sse';
import { getCurrentLanguage } from '../i18n';
import { getAccessToken } from './auth';

/**
 * 创建带有语言头的 fetch 函数
 */
function fetchWithLanguage(url, options = {}) {
  const language = getCurrentLanguage();
  const token = getAccessToken();

  const headers = {
    'Accept-Language': language,
    ...(options.headers || {}),
  };

  // 如果已登录，添加认证头
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  return fetch(url, {
    ...options,
    headers,
  });
}

const API_BASE = '/api/v1/remix';

/**
 * 检测消息中是否包含支持的社媒链接
 * @param {string} message - 用户消息
 * @returns {string|null} - 检测到的 URL 或 null
 */
function detectUrl(message) {
  const urlPatterns = [
    /https?:\/\/[^\s]+/,
    /xiaohongshu\.com/,
    /xhslink\.com/,
    /douyin\.com/,
    /v\.douyin\.com/,
    /bilibili\.com/,
    /b23\.tv/,
    /kuaishou\.com/,
    /v\.kuaishou\.com/,
  ];

  for (const pattern of urlPatterns) {
    if (pattern.test(message)) {
      // 提取完整 URL
      const urlMatch = message.match(/https?:\/\/[^\s]+/);
      return urlMatch ? urlMatch[0] : null;
    }
  }
  return null;
}

// ========== SSE 流式 API ==========

/**
 * 发送分析请求 (POST /analyze)
 *
 * @param {string} url - 社媒链接
 * @param {object} options - { mode, instruction, sessionId }
 * @param {object} callbacks - { onChunk, onError, onComplete }
 * @returns {Promise<void>}
 */
export async function sendAnalyzeRequest(url, options = {}, callbacks = {}) {
  const { mode, instruction, sessionId, originalMessage, llmConfigName } = options;

  return createSSEStream(`${API_BASE}/analyze`, {
    url,
    mode,
    instruction,
    original_message: originalMessage || null,
    session_id: sessionId,
    llm_config_name: llmConfigName || null,
  }, callbacks);
}

/**
 * 发送对话请求 (POST /chat)
 *
 * @param {string} message - 用户消息
 * @param {string} sessionId - 会话 ID
 * @param {object} callbacks - { onChunk, onError, onComplete }
 * @returns {Promise<void>}
 */
export async function sendChatRequest(message, sessionId, callbacks = {}, options = {}) {
  return createSSEStream(`${API_BASE}/chat`, {
    message,
    session_id: sessionId,
    llm_config_name: options.llmConfigName || null,
  }, callbacks);
}

/**
 * 智能发送消息 - 自动检测是分析请求还是对话请求
 *
 * @param {string} message - 用户消息
 * @param {string} token - Auth token (unused for now)
 * @param {object} options - { conversationId, context, history, attachments }
 * @param {object} callbacks - { onChunk, onError, onComplete }
 * @returns {Promise<void>}
 */
export async function sendChatMessage(message, token, options = {}, callbacks = {}) {
  const { conversationId, mode: passedMode, llmConfigName } = options;

  // 检测消息中是否包含 URL
  const detectedUrl = detectUrl(message);

  if (detectedUrl && !conversationId) {
    // 首次分析请求 - 包含 URL 且没有会话 ID
    // 从消息中提取 instruction (去掉 URL 部分)
    let instruction = message.replace(detectedUrl, '').trim();
    // 去除常见的前缀如 "总结这个视频："、"深度拆解这个内容的创作技巧："
    instruction = instruction
      .replace(/^(提炼|深度拆解|提取|探索|总结|改写|仿写|拆解分析)(这个)?(内容|视频|爆款)的?[^：:]*[：:]/g, '')
      .trim();

    // 优先使用传入的 mode，否则尝试从消息前缀推断
    let mode = passedMode || null;
    if (!mode) {
      if (message.includes('总结') || message.includes('提炼')) {
        mode = 'summarize';
      } else if (message.includes('模板')) {
        mode = 'template';
      } else if (message.includes('风格')) {
        mode = 'style_explore';
      } else if (message.includes('拆解') || message.includes('分析')) {
        mode = 'analyze';
      }
    }

    return sendAnalyzeRequest(detectedUrl, {
      mode,
      instruction: instruction || null,
      sessionId: conversationId,
      originalMessage: message,
      llmConfigName,
    }, callbacks);
  } else {
    // 对话追问请求 - 需要 session_id
    return sendChatRequest(message, conversationId, callbacks, { llmConfigName });
  }
}

// ========== 会话管理 API ==========

/**
 * 获取会话列表
 *
 * @param {number} limit - 返回数量限制
 * @param {number} offset - 偏移量
 * @returns {Promise<{sessions: Array, total: number}>}
 */
export async function fetchSessions(limit = 20, offset = 0) {
  const response = await fetchWithLanguage(`${API_BASE}/sessions?limit=${limit}&offset=${offset}`);
  if (!response.ok) {
    throw new Error(`Failed to fetch sessions: ${response.status}`);
  }
  return response.json();
}

/**
 * 删除会话
 *
 * @param {string} sessionId - 会话 ID
 * @returns {Promise<{success: boolean}>}
 */
export async function deleteSession(sessionId) {
  const response = await fetchWithLanguage(`${API_BASE}/session/${sessionId}`, {
    method: 'DELETE',
  });
  if (!response.ok) {
    throw new Error(`Failed to delete session: ${response.status}`);
  }
  return response.json();
}

/**
 * 获取单个会话状态
 *
 * @param {string} sessionId - 会话 ID
 * @returns {Promise<{session_id: string, content_info: object, transcript: string}|null>}
 */
export async function getSessionState(sessionId) {
  const response = await fetchWithLanguage(`${API_BASE}/session/${sessionId}`);
  if (!response.ok) {
    if (response.status === 404) {
      return null;
    }
    throw new Error(`Failed to get session: ${response.status}`);
  }
  return response.json();
}

/**
 * 获取会话消息列表
 *
 * @param {string} sessionId - 会话 ID
 * @param {number} limit - 返回数量限制
 * @param {number} offset - 偏移量
 * @returns {Promise<{session_id: string, messages: Array, total: number}|null>}
 */
export async function getSessionMessages(sessionId, limit = 100, offset = 0) {
  const response = await fetchWithLanguage(`${API_BASE}/session/${sessionId}/messages?limit=${limit}&offset=${offset}`);
  if (!response.ok) {
    if (response.status === 404) {
      return null;
    }
    throw new Error(`Failed to get session messages: ${response.status}`);
  }
  return response.json();
}

/**
 * 更新会话标题
 *
 * @param {string} sessionId - 会话 ID
 * @param {string} title - 新标题
 * @returns {Promise<object>}
 */
export async function updateSessionTitle(sessionId, title) {
  const response = await fetchWithLanguage(`${API_BASE}/session/${sessionId}`, {
    method: 'PATCH',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ title }),
  });
  if (!response.ok) {
    throw new Error(`Failed to update session: ${response.status}`);
  }
  return response.json();
}

/**
 * 获取可用的 remix 模式列表
 *
 * @returns {Promise<Array>}
 */
export async function fetchModes() {
  const response = await fetchWithLanguage(`${API_BASE}/modes`);
  if (!response.ok) {
    throw new Error('Failed to fetch modes');
  }
  return response.json();
}

// ========== Cookies 管理 API ==========

const COOKIES_API_BASE = '/api/v1/cookies';

/**
 * 获取所有平台 cookies 列表
 *
 * @returns {Promise<Array<{platform: string, cookies: string, remark: string, status: string, updated_at: string}>>}
 */
export async function fetchCookiesList() {
  const response = await fetchWithLanguage(COOKIES_API_BASE);
  if (!response.ok) {
    throw new Error(`Failed to fetch cookies: ${response.status}`);
  }
  const data = await response.json();
  // API returns { items: [...] }, extract the array
  return data.items || [];
}

/**
 * 获取指定平台 cookies 详情（含 cookies 明文）
 *
 * @param {string} platform - 平台标识 (xhs, dy, bili, ks)
 * @returns {Promise<{platform: string, cookies: string, remark: string, status: number, updated_at: string}>}
 */
export async function fetchCookiesDetail(platform) {
  const response = await fetchWithLanguage(`${COOKIES_API_BASE}/${platform}`);
  if (!response.ok) {
    throw new Error(`Failed to fetch cookies detail: ${response.status}`);
  }
  return response.json();
}

/**
 * 新增/更新平台 cookies
 *
 * @param {string} platform - 平台标识 (xhs, dy, bili, ks)
 * @param {string} cookies - cookies 内容
 * @param {string} remark - 备注
 * @returns {Promise<object>}
 */
export async function saveCookies(platform, cookies, remark = '') {
  const response = await fetchWithLanguage(`${COOKIES_API_BASE}/${platform}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ cookies, remark }),
  });
  if (!response.ok) {
    throw new Error(`Failed to save cookies: ${response.status}`);
  }
  return response.json();
}

/**
 * 删除平台 cookies
 *
 * @param {string} platform - 平台标识
 * @returns {Promise<object>}
 */
export async function deleteCookies(platform) {
  const response = await fetchWithLanguage(`${COOKIES_API_BASE}/${platform}`, {
    method: 'DELETE',
  });
  if (!response.ok) {
    throw new Error(`Failed to delete cookies: ${response.status}`);
  }
  return response.json();
}

// ========== LLM 配置管理 API ==========

const LLM_CONFIG_API_BASE = '/api/v1/llm-config';

/**
 * 检查 LLM 配置状态
 *
 * @returns {Promise<{configured: boolean, active_config: string|null}>}
 */
export async function checkLLMConfigStatus() {
  const response = await fetchWithLanguage(`${LLM_CONFIG_API_BASE}/status`);
  if (!response.ok) {
    throw new Error(`Failed to check LLM config status: ${response.status}`);
  }
  return response.json();
}

/**
 * 获取所有 LLM 配置列表
 *
 * @returns {Promise<{items: Array, active_config: string|null}>}
 */
export async function fetchLLMConfigList() {
  const response = await fetchWithLanguage(LLM_CONFIG_API_BASE);
  if (!response.ok) {
    throw new Error(`Failed to fetch LLM configs: ${response.status}`);
  }
  return response.json();
}

/**
 * 获取指定 LLM 配置
 *
 * @param {string} configName - 配置名称
 * @returns {Promise<object>}
 */
export async function fetchLLMConfig(configName) {
  const response = await fetchWithLanguage(`${LLM_CONFIG_API_BASE}/${encodeURIComponent(configName)}`);
  if (!response.ok) {
    throw new Error(`Failed to fetch LLM config: ${response.status}`);
  }
  return response.json();
}

/**
 * 创建新的 LLM 配置
 *
 * @param {object} config - 配置对象
 * @returns {Promise<object>}
 */
export async function createLLMConfig(config) {
  const response = await fetchWithLanguage(LLM_CONFIG_API_BASE, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(config),
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(error.detail || `Failed to create LLM config: ${response.status}`);
  }
  return response.json();
}

/**
 * 更新 LLM 配置
 *
 * @param {string} configName - 配置名称
 * @param {object} config - 配置更新对象
 * @returns {Promise<object>}
 */
export async function updateLLMConfig(configName, config) {
  const response = await fetchWithLanguage(`${LLM_CONFIG_API_BASE}/${encodeURIComponent(configName)}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(config),
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(error.detail || `Failed to update LLM config: ${response.status}`);
  }
  return response.json();
}

/**
 * 删除 LLM 配置
 *
 * @param {string} configName - 配置名称
 * @returns {Promise<object>}
 */
export async function deleteLLMConfig(configName) {
  const response = await fetchWithLanguage(`${LLM_CONFIG_API_BASE}/${encodeURIComponent(configName)}`, {
    method: 'DELETE',
  });
  if (!response.ok) {
    throw new Error(`Failed to delete LLM config: ${response.status}`);
  }
  return response.json();
}

/**
 * 激活指定 LLM 配置
 *
 * @param {string} configName - 配置名称
 * @returns {Promise<object>}
 */
export async function activateLLMConfig(configName) {
  const response = await fetchWithLanguage(`${LLM_CONFIG_API_BASE}/${encodeURIComponent(configName)}/activate`, {
    method: 'POST',
  });
  if (!response.ok) {
    throw new Error(`Failed to activate LLM config: ${response.status}`);
  }
  return response.json();
}

/**
 * 获取 LLM 提供商预设模板
 *
 * @returns {Promise<{templates: object}>}
 */
export async function fetchLLMTemplates() {
  const response = await fetchWithLanguage(`${LLM_CONFIG_API_BASE}/templates/list`);
  if (!response.ok) {
    throw new Error(`Failed to fetch LLM templates: ${response.status}`);
  }
  return response.json();
}

// ========== Insight Mode 配置管理 API ==========

const INSIGHT_MODE_API_BASE = '/api/v1/insight-modes';

/**
 * 获取所有 Insight Mode 列表
 *
 * @param {boolean} activeOnly - 是否只返回启用的模式
 * @returns {Promise<{items: Array}>}
 */
export async function fetchInsightModeList(activeOnly = false) {
  const url = activeOnly
    ? `${INSIGHT_MODE_API_BASE}?active_only=true`
    : INSIGHT_MODE_API_BASE;
  const response = await fetchWithLanguage(url);
  if (!response.ok) {
    throw new Error(`Failed to fetch insight modes: ${response.status}`);
  }
  return response.json();
}

/**
 * 获取指定 Insight Mode 详情
 *
 * @param {string} modeKey - 模式标识
 * @returns {Promise<object>}
 */
export async function fetchInsightMode(modeKey) {
  const response = await fetchWithLanguage(`${INSIGHT_MODE_API_BASE}/${encodeURIComponent(modeKey)}`);
  if (!response.ok) {
    throw new Error(`Failed to fetch insight mode: ${response.status}`);
  }
  return response.json();
}

/**
 * 创建新的 Insight Mode
 *
 * @param {object} data - 模式数据
 * @returns {Promise<object>}
 */
export async function createInsightMode(data) {
  const response = await fetchWithLanguage(INSIGHT_MODE_API_BASE, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(error.detail || `Failed to create insight mode: ${response.status}`);
  }
  return response.json();
}

/**
 * 更新 Insight Mode
 *
 * @param {string} modeKey - 模式标识
 * @param {object} data - 更新数据
 * @returns {Promise<object>}
 */
export async function updateInsightMode(modeKey, data) {
  const response = await fetchWithLanguage(`${INSIGHT_MODE_API_BASE}/${encodeURIComponent(modeKey)}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(error.detail || `Failed to update insight mode: ${response.status}`);
  }
  return response.json();
}

/**
 * 删除 Insight Mode
 *
 * @param {string} modeKey - 模式标识
 * @returns {Promise<object>}
 */
export async function deleteInsightMode(modeKey) {
  const response = await fetchWithLanguage(`${INSIGHT_MODE_API_BASE}/${encodeURIComponent(modeKey)}`, {
    method: 'DELETE',
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(error.detail || `Failed to delete insight mode: ${response.status}`);
  }
  return response.json();
}

/**
 * 切换 Insight Mode 启用状态
 *
 * @param {string} modeKey - 模式标识
 * @returns {Promise<object>}
 */
export async function toggleInsightMode(modeKey) {
  const response = await fetchWithLanguage(`${INSIGHT_MODE_API_BASE}/${encodeURIComponent(modeKey)}/toggle`, {
    method: 'POST',
  });
  if (!response.ok) {
    throw new Error(`Failed to toggle insight mode: ${response.status}`);
  }
  return response.json();
}

/**
 * 更新 Insight Mode 排序
 *
 * @param {Array<string>} modeKeys - 按顺序排列的 mode_key 列表
 * @returns {Promise<object>}
 */
export async function reorderInsightModes(modeKeys) {
  const response = await fetchWithLanguage(`${INSIGHT_MODE_API_BASE}/reorder`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ mode_keys: modeKeys }),
  });
  if (!response.ok) {
    throw new Error(`Failed to reorder insight modes: ${response.status}`);
  }
  return response.json();
}
