// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-CookieBridge project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-CookieBridge/blob/main/extension/popup.js
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

const PLATFORM_LABELS = {
  xhs: "XiaoHongShu",
  dy: "Douyin",
  ks: "Kuaishou",
  bili: "Bilibili",
  wb: "Weibo",
  tieba: "Tieba",
  zhihu: "Zhihu"
};

const ALL_PLATFORMS = ["xhs", "dy", "ks", "bili", "wb", "tieba", "zhihu"];

const CHECK_SVG = '<svg viewBox="0 0 10 10" fill="none"><path d="M8.5 3L4.2 7.3 1.5 4.6" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>';
const WARN_SVG = '<svg viewBox="0 0 10 10" fill="none"><path d="M5 1.5L9 8.5H1L5 1.5z" stroke="currentColor" stroke-width="1" stroke-linejoin="round"/><path d="M5 4.5v1.5" stroke="currentColor" stroke-width="1" stroke-linecap="round"/><circle cx="5" cy="7.2" r="0.5" fill="currentColor"/></svg>';

function render(connected, platforms, nicknames, clientIdValue) {
  const dot = document.getElementById("statusDot");
  const text = document.getElementById("statusText");

  if (connected) {
    dot.className = "status-dot connected";
    text.textContent = "Online";
  } else {
    dot.className = "status-dot disconnected";
    text.textContent = "Offline";
  }

  // Display client ID
  const clientIdEl = document.getElementById("clientId");
  if (clientIdValue) {
    clientIdEl.textContent = clientIdValue.substring(0, 8) + "-" + clientIdValue.substring(8, 12) + "...";
    clientIdEl.title = clientIdValue;
  } else {
    clientIdEl.textContent = "--";
  }

  const list = document.getElementById("platformList");
  list.innerHTML = "";

  const nicks = nicknames || {};

  for (const p of ALL_PLATFORMS) {
    const info = (platforms && platforms[p]) || { has_cookies: false, cookie_count: 0, has_login: false };

    const row = document.createElement("div");
    row.className = "platform-row " + (info.has_login ? "logged-in" : "logged-out");

    // Left section
    const left = document.createElement("div");
    left.className = "platform-left";

    // Platform icon
    const icon = document.createElement("img");
    icon.className = "platform-icon";
    icon.src = "icons/platforms/" + p + ".png";
    icon.alt = PLATFORM_LABELS[p];

    // Name + nickname
    const infoDiv = document.createElement("div");
    infoDiv.className = "platform-info";

    const name = document.createElement("div");
    name.className = "platform-name";
    name.textContent = PLATFORM_LABELS[p];

    infoDiv.appendChild(name);

    if (nicks[p]) {
      const nick = document.createElement("div");
      nick.className = "nickname";
      nick.textContent = "@" + nicks[p];
      nick.title = nicks[p];
      infoDiv.appendChild(nick);
    }

    left.appendChild(icon);
    left.appendChild(infoDiv);

    // Right section
    const right = document.createElement("div");
    right.className = "platform-right";

    const badge = document.createElement("span");
    if (info.has_login) {
      badge.className = "login-badge yes";
      badge.innerHTML = CHECK_SVG + "\u5df2\u767b\u5f55";
    } else {
      badge.className = "login-badge no";
      badge.innerHTML = WARN_SVG + "\u672a\u767b\u5f55";
    }

    right.appendChild(badge);

    row.appendChild(left);
    row.appendChild(right);
    list.appendChild(row);
  }
}

// Toast notification
let toastTimer = null;
function showToast(message, type = "success") {
  const toast = document.getElementById("toast");
  toast.textContent = message;
  toast.className = "toast " + type;
  // Force reflow then show
  void toast.offsetWidth;
  toast.classList.add("visible");
  if (toastTimer) clearTimeout(toastTimer);
  toastTimer = setTimeout(() => {
    toast.classList.remove("visible");
  }, 2000);
}

// Load state from storage
function loadState() {
  chrome.storage.local.get(["wsConnected", "platforms", "nicknames", "clientId"], (data) => {
    render(data.wsConnected || false, data.platforms || {}, data.nicknames || {}, data.clientId || "");
  });
}

// Sync button
document.getElementById("syncBtn").addEventListener("click", () => {
  const btn = document.getElementById("syncBtn");
  const btnText = document.getElementById("syncBtnText");
  btnText.textContent = "Syncing...";
  btn.disabled = true;
  btn.classList.add("syncing");

  chrome.runtime.sendMessage({ type: "sync" }, (response) => {
    btnText.textContent = "Sync All";
    btn.disabled = false;
    btn.classList.remove("syncing");
    if (response) {
      render(response.connected, response.platforms, response.nicknames, response.clientId);
      if (response.connected) {
        const platforms = response.platforms || {};
        const loggedIn = Object.values(platforms).filter(p => p.has_login).length;
        showToast(`Synced! ${loggedIn} platform${loggedIn !== 1 ? "s" : ""} logged in`, "success");
      } else {
        showToast("Server offline, sync failed", "error");
      }
    } else {
      showToast("Sync failed", "error");
    }
  });
});

// Listen for storage changes
chrome.storage.onChanged.addListener((changes) => {
  if (changes.wsConnected || changes.platforms || changes.nicknames || changes.clientId) {
    loadState();
  }
});

// Initial load
loadState();
