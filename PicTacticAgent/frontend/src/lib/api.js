import { getAccessToken, refreshTokens, clearTokens, getRefreshToken } from '@/services/auth';

const API_BASE_URL = import.meta.env.VITE_API_URL !== undefined ? import.meta.env.VITE_API_URL : "http://localhost:8019";

function getLang() {
  return localStorage.getItem("locale") || "zh";
}

class ApiClient {
  constructor(baseUrl = API_BASE_URL) {
    this.baseUrl = baseUrl;
  }

  _headers(extra = {}) {
    const token = getAccessToken();
    // Exclude Authorization from extra to avoid old token overwriting refreshed token
    const { Authorization, ...restExtra } = extra;
    return {
      "Accept-Language": getLang(),
      ...(token ? { "Authorization": `Bearer ${token}` } : {}),
      ...restExtra,
    };
  }

  async _fetchWithAuth(url, options = {}) {
    let response = await fetch(url, options);

    // 如果 401，尝试刷新 Token
    if (response.status === 401 && getRefreshToken()) {
      try {
        await refreshTokens();
        // 用新 token 重试
        const newHeaders = this._headers(options.headers || {});
        response = await fetch(url, { ...options, headers: newHeaders });
      } catch {
        // 刷新失败，清除 Token
        clearTokens();
        // 重定向到登录页
        window.location.href = '/auth/login';
        throw new Error('Session expired');
      }
    }

    return response;
  }

  async getProviderCapabilities() {
    const response = await this._fetchWithAuth(`${this.baseUrl}/api/v1/generation/provider/capabilities`, {
      headers: this._headers(),
    });
    if (!response.ok) {
      return { name: null, supported_modes: ["text_to_image", "image_to_image"], supports_batch: true };
    }
    return response.json();
  }

  async createTask(request) {
    const response = await this._fetchWithAuth(`${this.baseUrl}/api/v1/generation/`, {
      method: "POST",
      headers: this._headers({ "Content-Type": "application/json" }),
      body: JSON.stringify(request),
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || "Failed to create task");
    }
    return response.json();
  }

  async getTaskStatus(taskId) {
    const response = await this._fetchWithAuth(`${this.baseUrl}/api/v1/generation/${taskId}`, {
      headers: this._headers(),
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || "Failed to get task status");
    }
    return response.json();
  }

  async getTaskResult(taskId) {
    const response = await this._fetchWithAuth(`${this.baseUrl}/api/v1/generation/${taskId}/result`, {
      headers: this._headers(),
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || "Failed to get task result");
    }
    return response.json();
  }

  async editImage(taskId, imageId, editRequest) {
    const response = await this._fetchWithAuth(
      `${this.baseUrl}/api/v1/generation/${taskId}/images/${imageId}/edit`,
      {
        method: "POST",
        headers: this._headers({ "Content-Type": "application/json" }),
        body: JSON.stringify(editRequest),
      }
    );
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || "Failed to edit image");
    }
    return response.json();
  }

  async cancelTask(taskId) {
    const response = await this._fetchWithAuth(`${this.baseUrl}/api/v1/generation/${taskId}/cancel`, {
      method: "POST",
      headers: this._headers(),
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || "Failed to cancel task");
    }
    return response.json();
  }

  async listTasks(limit = 20) {
    const response = await this._fetchWithAuth(`${this.baseUrl}/api/v1/generation/?limit=${limit}`, {
      headers: this._headers(),
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || "Failed to list tasks");
    }
    return response.json();
  }

  async listTaskHistory(limit = 50, offset = 0) {
    const response = await this._fetchWithAuth(
      `${this.baseUrl}/api/v1/generation/history/list?limit=${limit}&offset=${offset}`,
      { headers: this._headers() }
    );
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || "Failed to list task history");
    }
    return response.json();
  }

  // ========== Provider Settings ==========

  async getProviderSettings() {
    const response = await this._fetchWithAuth(`${this.baseUrl}/api/v1/settings/provider`, {
      headers: this._headers(),
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || "Failed to get provider settings");
    }
    return response.json();
  }

  async updateProviderSettings(data) {
    const response = await this._fetchWithAuth(`${this.baseUrl}/api/v1/settings/provider`, {
      method: "PUT",
      headers: this._headers({ "Content-Type": "application/json" }),
      body: JSON.stringify(data),
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || "Failed to update provider settings");
    }
    return response.json();
  }

  getWebSocketUrl(taskId) {
    const lang = getLang();
    const token = getAccessToken();
    const tokenParam = token ? `&token=${encodeURIComponent(token)}` : "";
    if (this.baseUrl) {
      const wsUrl = this.baseUrl.replace(/^http/, "ws");
      return `${wsUrl}/ws/progress/${taskId}?lang=${lang}${tokenParam}`;
    }
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${wsProtocol}//${window.location.host}/ws/progress/${taskId}?lang=${lang}${tokenParam}`;
  }

  // ========== Task Management ==========

  async deleteTask(taskId) {
    const response = await this._fetchWithAuth(
      `${this.baseUrl}/api/v1/generation/${taskId}`,
      { method: "DELETE", headers: this._headers() }
    );
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || "Failed to delete task");
    }
    return response.json();
  }

  // ========== Video Generation ==========

  async createVideoTask(request) {
    const response = await this._fetchWithAuth(`${this.baseUrl}/api/v1/video/`, {
      method: "POST",
      headers: this._headers({ "Content-Type": "application/json" }),
      body: JSON.stringify(request),
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || "Failed to create video task");
    }
    return response.json();
  }

  async getVideoTaskStatus(taskId) {
    const response = await this._fetchWithAuth(`${this.baseUrl}/api/v1/video/${taskId}`, {
      headers: this._headers(),
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || "Failed to get video task status");
    }
    return response.json();
  }

  async getVideoTaskResult(taskId) {
    const response = await this._fetchWithAuth(`${this.baseUrl}/api/v1/video/${taskId}/result`, {
      headers: this._headers(),
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || "Failed to get video task result");
    }
    return response.json();
  }

  async cancelVideoTask(taskId) {
    const response = await this._fetchWithAuth(`${this.baseUrl}/api/v1/video/${taskId}/cancel`, {
      method: "POST",
      headers: this._headers(),
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || "Failed to cancel video task");
    }
    return response.json();
  }

  async listVideoProviders() {
    const response = await this._fetchWithAuth(`${this.baseUrl}/api/v1/video/providers/list`, {
      headers: this._headers(),
    });
    if (!response.ok) {
      return [];
    }
    return response.json();
  }

  async listVideoTasks(limit = 20) {
    const response = await this._fetchWithAuth(`${this.baseUrl}/api/v1/video/?limit=${limit}`, {
      headers: this._headers(),
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || "Failed to list video tasks");
    }
    return response.json();
  }

  getVideoWebSocketUrl(taskId) {
    const lang = getLang();
    const token = getAccessToken();
    const tokenParam = token ? `&token=${encodeURIComponent(token)}` : "";
    if (this.baseUrl) {
      const wsUrl = this.baseUrl.replace(/^http/, "ws");
      return `${wsUrl}/ws/video/${taskId}?lang=${lang}${tokenParam}`;
    }
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${wsProtocol}//${window.location.host}/ws/video/${taskId}?lang=${lang}${tokenParam}`;
  }

  // ========== User Profile ==========

  async updateProfile(data) {
    const response = await this._fetchWithAuth(
      `${this.baseUrl}/api/v1/auth/profile`,
      {
        method: "PUT",
        headers: this._headers({ "Content-Type": "application/json" }),
        body: JSON.stringify(data),
      }
    );
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || "Failed to update profile");
    }
    return response.json();
  }

  // ========== Templates ==========

  async createTemplate(data) {
    const response = await this._fetchWithAuth(`${this.baseUrl}/api/v1/templates/`, {
      method: "POST",
      headers: this._headers({ "Content-Type": "application/json" }),
      body: JSON.stringify(data),
    });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || "Failed to create template");
    }
    return response.json();
  }

  async listTemplates({ search, category, page = 1, pageSize = 20 } = {}) {
    const params = new URLSearchParams();
    if (search) params.set("search", search);
    if (category) params.set("category", category);
    params.set("page", String(page));
    params.set("page_size", String(pageSize));

    const response = await this._fetchWithAuth(
      `${this.baseUrl}/api/v1/templates/?${params}`,
      { headers: this._headers() }
    );
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || "Failed to list templates");
    }
    return response.json();
  }

  async getTemplate(templateId) {
    const response = await this._fetchWithAuth(
      `${this.baseUrl}/api/v1/templates/${templateId}`,
      { headers: this._headers() }
    );
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || "Failed to get template");
    }
    return response.json();
  }

  async updateTemplate(templateId, data) {
    const response = await this._fetchWithAuth(
      `${this.baseUrl}/api/v1/templates/${templateId}`,
      {
        method: "PUT",
        headers: this._headers({ "Content-Type": "application/json" }),
        body: JSON.stringify(data),
      }
    );
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || "Failed to update template");
    }
    return response.json();
  }

  async deleteTemplate(templateId) {
    const response = await this._fetchWithAuth(
      `${this.baseUrl}/api/v1/templates/${templateId}`,
      { method: "DELETE", headers: this._headers() }
    );
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || "Failed to delete template");
    }
    return response.json();
  }

  async getTemplateCategories() {
    const response = await this._fetchWithAuth(
      `${this.baseUrl}/api/v1/templates/categories`,
      { headers: this._headers() }
    );
    if (!response.ok) {
      return [];
    }
    return response.json();
  }

  async generateFromTemplate(templateId, data = {}) {
    const response = await this._fetchWithAuth(
      `${this.baseUrl}/api/v1/templates/${templateId}/generate`,
      {
        method: "POST",
        headers: this._headers({ "Content-Type": "application/json" }),
        body: JSON.stringify(data),
      }
    );
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || "Failed to generate from template");
    }
    return response.json();
  }
}

export const apiClient = new ApiClient();
