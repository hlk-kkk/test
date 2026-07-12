// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-CookieBridge project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-CookieBridge/blob/main/extension/background.js
// GitHub: https://github.com/NanmiCoder
// Licensed under NON-COMMERCIAL LEARNING LICENSE 1.1
//
// 声明：本代码仅供学习和研究目的使用。使用者应遵守以下原则：
// 1. 不得用于任何商业用途。
// 2. 使用时应遵守目标平台的使用条款和robots.txt规则。
// 3. 不得进行大规模爬取或对平台造成运营干扰。
// 4. 应合理控制请求频率，避免给目标平台带来不必要的负担。
// 5. 不得用于任何非法或不当的用途。
//
// 详细许可条款请参阅项目根目录下的LICENSE文件。
// 使用本代码即表示您同意遵守上述原则和LICENSE中的所有条款。

// MediaCrawlerPro CookieBridge - Background Service Worker (V2 Multi-Account)

const WS_URL = "ws://localhost:8274/ws";

const PLATFORM_DOMAINS = {
  xhs: [".xiaohongshu.com"],
  dy: [".douyin.com"],
  ks: [".kuaishou.com"],
  bili: [".bilibili.com"],
  wb: [".weibo.com", ".weibo.cn"],
  tieba: [".tieba.baidu.com"],
  zhihu: [".zhihu.com"]
};

// Unified login verification config per platform
// cookieCheck: quick cookie-based login indicator (from MediaCrawler login.py)
// url/method/body: API-based verification for accuracy (from MediaCrawlerPro client.py pong())
// isLoggedIn: function to check API response for login state
// extractNickname: function to extract nickname from API response
const LOGIN_VERIFY_APIS = {
  xhs: {
    cookieCheck: { name: "web_session" },
    url: "https://edith.xiaohongshu.com/api/sns/web/v2/user/me",
    method: "GET",
    isLoggedIn: (d) => !!(d?.data?.nickname),
    extractNickname: (d) => d?.data?.nickname || null,
  },
  dy: {
    // Douyin APIs require a_bogus signing which the extension cannot generate.
    // Check multiple possible login cookies:
    // - sessionid/sessionid_ss: real session tokens (httpOnly, only exist when logged in)
    // - sid_tt: another session identifier
    // - LOGIN_STATUS=="1": per MediaCrawler login.py (may not exist in newer versions)
    cookieCheck: [
      { name: "sessionid" },
      { name: "sessionid_ss" },
      { name: "sid_tt" },
      { name: "LOGIN_STATUS", value: "1" },
    ],
    url: null,
    isLoggedIn: null,
    extractNickname: null,
  },
  ks: {
    // MediaCrawler login.py checks passToken cookie
    cookieCheck: { name: "passToken" },
    // MediaCrawlerPro client.py pong() uses GraphQL visionProfileUserList
    url: "https://www.kuaishou.com/graphql",
    method: "POST",
    headers: { "Accept": "application/json", "Content-Type": "application/json" },
    body: JSON.stringify({
      operationName: "visionProfileUserList",
      variables: { ftype: 1 },
      query: "query visionProfileUserList($pcursor: String, $ftype: Int) { visionProfileUserList(pcursor: $pcursor, ftype: $ftype) { result fols { user_name headurl user_text isFollowing user_id __typename } hostName pcursor __typename } }"
    }),
    isLoggedIn: (d) => d?.data?.visionProfileUserList?.result === 1,
    extractNickname: (d) => d?.data?.visionProfileUserList?.hostName || null,
  },
  bili: {
    // MediaCrawler login.py checks SESSDATA or DedeUserID
    cookieCheck: [{ name: "SESSDATA" }, { name: "DedeUserID" }],
    url: "https://api.bilibili.com/x/web-interface/nav",
    method: "GET",
    isLoggedIn: (d) => d?.code === 0 && !!d?.data?.isLogin,
    extractNickname: (d) => d?.data?.uname || null,
  },
  wb: {
    // MediaCrawler login.py checks SSOLoginState cookie (on .weibo.cn)
    // H5 login sets MLOGIN=1 on .weibo.cn; desktop sets SUB on .weibo.com
    cookieCheck: [
      { name: "SSOLoginState" },
      { name: "MLOGIN", value: "1" },
      { name: "SUB" },
    ],
    url: "https://weibo.com/ajax/login/config",
    method: "GET",
    isLoggedIn: (d) => !!d?.data?.login,
    extractNickname: (d) => d?.data?.user?.screen_name || null,
  },
  tieba: {
    // MediaCrawler login.py checks STOKEN or PTOKEN
    cookieCheck: [{ name: "STOKEN" }, { name: "PTOKEN" }],
    url: "https://tieba.baidu.com/f/user/json_userinfo",
    method: "GET",
    isLoggedIn: (d) => !!(d?.data?.user_name_show),
    extractNickname: (d) => d?.data?.user_name_show || null,
  },
  zhihu: {
    // MediaCrawler login.py checks z_c0
    cookieCheck: { name: "z_c0" },
    url: "https://www.zhihu.com/api/v4/me",
    method: "GET",
    isLoggedIn: (d) => !!(d?.uid && d?.name),
    extractNickname: (d) => d?.name || null,
  },
};

// Login verification cache: { platform: { loggedIn, nickname, timestamp } }
const loginVerifyCache = {};
const LOGIN_VERIFY_CACHE_TTL = 120_000; // 2 min
const LOGIN_VERIFY_DEBOUNCE_MS = 3000;  // 3s debounce for cookie-change triggered API checks
const loginVerifyDebounce = {};

const ALL_PLATFORMS = Object.keys(PLATFORM_DOMAINS);

let ws = null;
let reconnectDelay = 1000;
let reconnectTimer = null;
let clientId = null;

// Generate or retrieve persistent client ID
async function getClientId() {
  const data = await chrome.storage.local.get(["clientId"]);
  if (data.clientId) return data.clientId;
  const newId = crypto.randomUUID();
  await chrome.storage.local.set({ clientId: newId });
  return newId;
}

// Build a name->value map of all cookies for a platform's domains
async function buildCookieMap(platform) {
  const domains = PLATFORM_DOMAINS[platform];
  const cookieMap = new Map();
  if (!domains) return cookieMap;
  for (const domain of domains) {
    const cookies = await chrome.cookies.getAll({ domain });
    for (const c of cookies) {
      if (!cookieMap.has(c.name)) cookieMap.set(c.name, c.value);
    }
  }
  return cookieMap;
}

// Check login state via cookie indicators (fast, from MediaCrawler login.py)
function checkLoginViaCookies(platform, cookieMap) {
  const config = LOGIN_VERIFY_APIS[platform];
  if (!config?.cookieCheck) return false;

  const checks = Array.isArray(config.cookieCheck) ? config.cookieCheck : [config.cookieCheck];
  // For arrays (e.g. bili checks SESSDATA OR DedeUserID), any match = logged in
  return checks.some(check => {
    const cookieValue = cookieMap.get(check.name);
    if (cookieValue === undefined || cookieValue === "") return false;
    // If a specific value is required (e.g. dy LOGIN_STATUS must be "1"), check it
    if (check.value !== undefined) return cookieValue === check.value;
    return true;
  });
}

// Get cookies for a specific platform
async function getCookiesForPlatform(platform) {
  const cookieMap = await buildCookieMap(platform);
  const pairs = [...cookieMap.entries()].map(([k, v]) => `${k}=${v}`);
  const cookieString = pairs.join("; ");

  // Use API verification cache if available, otherwise fall back to cookie check
  const cached = loginVerifyCache[platform];
  const hasLogin = (cached && (Date.now() - cached.timestamp) < LOGIN_VERIFY_CACHE_TTL)
    ? cached.loggedIn
    : checkLoginViaCookies(platform, cookieMap);

  return { cookieString, cookieCount: pairs.length, hasLogin };
}

// Verify login state via API call for a single platform
// Returns { loggedIn: boolean, nickname: string|null }
async function verifyLogin(platform) {
  const config = LOGIN_VERIFY_APIS[platform];
  if (!config) return { loggedIn: false, nickname: null };

  // Check cache first
  const cached = loginVerifyCache[platform];
  if (cached && (Date.now() - cached.timestamp) < LOGIN_VERIFY_CACHE_TTL) {
    return { loggedIn: cached.loggedIn, nickname: cached.nickname };
  }

  // No API endpoint (e.g. Douyin) - fall back to cookie check
  if (!config.url) {
    const { hasLogin } = await getCookiesForPlatform(platform);
    const result = { loggedIn: hasLogin, nickname: null };
    loginVerifyCache[platform] = { ...result, timestamp: Date.now() };
    return result;
  }

  // Build cookie map for both API call and cookie fallback
  const cookieMap = await buildCookieMap(platform);
  const cookieString = [...cookieMap.entries()].map(([k, v]) => `${k}=${v}`).join("; ");

  // Call the platform's API to verify login
  try {
    const headers = { ...(config.headers || { "Accept": "application/json" }) };
    if (cookieString) headers["Cookie"] = cookieString;

    const fetchOptions = {
      method: config.method || "GET",
      credentials: "include",
      headers,
    };
    if (config.body) fetchOptions.body = config.body;

    const response = await fetch(config.url, fetchOptions);
    let loggedIn = false;
    let nickname = null;
    if (response.ok) {
      const data = await response.json();
      loggedIn = !!config.isLoggedIn(data);
      nickname = loggedIn ? (config.extractNickname(data) || null) : null;
    }

    // If API says not logged in, fall back to cookie check
    // (e.g. weibo H5 login vs desktop API, or API temporarily broken)
    if (!loggedIn) {
      loggedIn = checkLoginViaCookies(platform, cookieMap);
    }

    const result = { loggedIn, nickname };
    loginVerifyCache[platform] = { ...result, timestamp: Date.now() };
    return result;
  } catch {
    // Network error - try cookie check, then cache
    const cookieLoggedIn = checkLoginViaCookies(platform, cookieMap);
    if (cookieLoggedIn) {
      const result = { loggedIn: true, nickname: null };
      loginVerifyCache[platform] = { ...result, timestamp: Date.now() };
      return result;
    }
    const currentCache = loginVerifyCache[platform];
    if (currentCache) {
      loginVerifyCache[platform] = { ...currentCache, timestamp: Date.now() };
      return { loggedIn: currentCache.loggedIn, nickname: currentCache.nickname };
    }
    const result = { loggedIn: false, nickname: null };
    loginVerifyCache[platform] = { ...result, timestamp: Date.now() };
    return result;
  }
}

// Verify all platforms in parallel, returns { platforms, nicknames }
async function verifyAllLogins() {
  const loginStatus = {};
  const nicknames = {};

  const entries = await Promise.allSettled(
    ALL_PLATFORMS.map(async (p) => {
      const result = await verifyLogin(p);
      return [p, result];
    })
  );
  for (const entry of entries) {
    if (entry.status === "fulfilled") {
      const [platform, result] = entry.value;
      loginStatus[platform] = result.loggedIn;
      if (result.nickname) nicknames[platform] = result.nickname;
    }
  }
  return { loginStatus, nicknames };
}

// Get status for all platforms (with API verification)
async function getAllPlatformStatus() {
  const { loginStatus, nicknames } = await verifyAllLogins();

  const platforms = {};
  const cookieInfos = await Promise.all(
    ALL_PLATFORMS.map(async (p) => [p, await getCookiesForPlatform(p)])
  );
  for (const [platform, info] of cookieInfos) {
    platforms[platform] = {
      has_cookies: info.cookieCount > 0,
      cookie_count: info.cookieCount,
      has_login: loginStatus[platform] || false,
    };
  }
  return { platforms, nicknames };
}

// Save state to chrome.storage.local for popup
async function saveState(connected, platforms, nicknames) {
  const stateData = {
    wsConnected: connected,
    platforms: platforms || {},
    lastUpdate: Date.now(),
    clientId: clientId
  };
  if (nicknames !== undefined) {
    stateData.nicknames = nicknames;
  }
  await chrome.storage.local.set(stateData);
}

// Send a JSON message over WebSocket (auto-attaches client_id)
function wsSend(msg) {
  if (ws && ws.readyState === WebSocket.OPEN) {
    msg.client_id = clientId;
    ws.send(JSON.stringify(msg));
    return true;
  }
  return false;
}

// Handle incoming messages from the server
async function handleMessage(data) {
  let msg;
  try {
    msg = JSON.parse(data);
  } catch {
    return;
  }

  switch (msg.type) {
    case "get_cookies": {
      const { cookieString, hasLogin } = await getCookiesForPlatform(msg.platform);
      wsSend({
        type: "cookies_response",
        request_id: msg.request_id,
        platform: msg.platform,
        cookies: cookieString,
        valid: hasLogin
      });
      break;
    }

    case "get_all_cookies": {
      for (const platform of ALL_PLATFORMS) {
        const { cookieString, hasLogin } = await getCookiesForPlatform(platform);
        wsSend({
          type: "cookies_response",
          request_id: msg.request_id,
          platform: platform,
          cookies: cookieString,
          valid: hasLogin
        });
      }
      wsSend({
        type: "cookies_response_complete",
        request_id: msg.request_id
      });
      break;
    }

    case "heartbeat_ack":
      break;

    default:
      break;
  }
}

// Connect to WebSocket server
function connect() {
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
    return;
  }

  try {
    ws = new WebSocket(WS_URL);
  } catch {
    scheduleReconnect();
    return;
  }

  ws.onopen = async () => {
    reconnectDelay = 1000;

    const { platforms, nicknames } = await getAllPlatformStatus();

    const registerPayload = {
      type: "register",
      nicknames: nicknames,
      platforms: {}
    };
    for (const [p, info] of Object.entries(platforms)) {
      registerPayload.platforms[p] = {
        has_cookies: info.has_cookies,
        cookie_count: info.cookie_count
      };
    }
    wsSend(registerPayload);

    await saveState(true, platforms, nicknames);
  };

  ws.onmessage = (event) => {
    handleMessage(event.data);
  };

  ws.onclose = () => {
    ws = null;
    saveState(false, null);
    scheduleReconnect();
  };

  ws.onerror = () => {
    // onclose will fire after onerror
  };
}

function scheduleReconnect() {
  if (reconnectTimer) clearTimeout(reconnectTimer);
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null;
    connect();
  }, reconnectDelay);
  reconnectDelay = Math.min(reconnectDelay * 2, 30000);
}

// Keepalive alarm
chrome.alarms.create("keepalive", { periodInMinutes: 0.4 });

chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === "keepalive") {
    if (ws && ws.readyState === WebSocket.OPEN) {
      wsSend({ type: "heartbeat" });
    } else {
      connect();
    }
  }
});

// Detect which platform a cookie domain belongs to
function domainToPlatform(domain) {
  for (const [platform, domains] of Object.entries(PLATFORM_DOMAINS)) {
    for (const d of domains) {
      if (domain === d || domain.endsWith(d)) {
        return platform;
      }
    }
  }
  return null;
}

// Proactively send cookie updates when cookies change
const cookieUpdateDebounce = {};

chrome.cookies.onChanged.addListener((changeInfo) => {
  const domain = changeInfo.cookie.domain;
  const platform = domainToPlatform(domain);
  if (!platform) return;

  // Fast path: push cookie updates to server (500ms debounce)
  if (cookieUpdateDebounce[platform]) {
    clearTimeout(cookieUpdateDebounce[platform]);
  }
  cookieUpdateDebounce[platform] = setTimeout(async () => {
    delete cookieUpdateDebounce[platform];

    const { cookieString, cookieCount, hasLogin } = await getCookiesForPlatform(platform);

    wsSend({
      type: "cookies_update",
      platform: platform,
      cookies: cookieString,
      timestamp: Math.floor(Date.now() / 1000)
    });

    const stored = await chrome.storage.local.get(["platforms"]);
    const platforms = stored.platforms || {};
    platforms[platform] = {
      has_cookies: cookieCount > 0,
      cookie_count: cookieCount,
      has_login: hasLogin
    };
    await chrome.storage.local.set({ platforms, lastUpdate: Date.now() });
  }, 500);

  // Slow path: API-based login verification (3s debounce, avoids hammering during login/logout)
  if (loginVerifyDebounce[platform]) {
    clearTimeout(loginVerifyDebounce[platform]);
  }
  loginVerifyDebounce[platform] = setTimeout(async () => {
    delete loginVerifyDebounce[platform];
    // Clear cache to force re-verify
    delete loginVerifyCache[platform];
    const result = await verifyLogin(platform);

    // Update stored state with accurate login status
    const stored = await chrome.storage.local.get(["platforms", "nicknames"]);
    const platforms = stored.platforms || {};
    if (platforms[platform]) {
      platforms[platform].has_login = result.loggedIn;
    }
    const nicknames = stored.nicknames || {};
    if (result.nickname) {
      nicknames[platform] = result.nickname;
    } else {
      delete nicknames[platform];
    }
    await chrome.storage.local.set({ platforms, nicknames, lastUpdate: Date.now() });
  }, LOGIN_VERIFY_DEBOUNCE_MS);
});

// Listen for messages from popup
chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (msg.type === "refresh" || msg.type === "sync") {
    (async () => {
      // Clear all verification caches to force re-check
      for (const p of ALL_PLATFORMS) delete loginVerifyCache[p];

      const { platforms, nicknames } = await getAllPlatformStatus();
      const connected = ws && ws.readyState === WebSocket.OPEN;

      // Re-send register to server with updated info
      if (connected) {
        const registerPayload = {
          type: "register",
          nicknames: nicknames,
          platforms: {}
        };
        for (const [p, info] of Object.entries(platforms)) {
          registerPayload.platforms[p] = {
            has_cookies: info.has_cookies,
            cookie_count: info.cookie_count
          };
        }
        wsSend(registerPayload);
      }

      await saveState(connected, platforms, nicknames);
      sendResponse({ connected, platforms, nicknames, clientId });
    })();
    return true; // async response
  }
  if (msg.type === "reconnect") {
    connect();
    sendResponse({ ok: true });
    return false;
  }
});

// Initialize client ID and start connection
(async () => {
  clientId = await getClientId();
  connect();
})();
