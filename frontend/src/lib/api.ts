const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

interface FetchOptions extends RequestInit {
  token?: string;
}

async function fetchAPI(path: string, options: FetchOptions = {}) {
  const { token, ...fetchOptions } = options;
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${API_URL}${path}`, {
    ...fetchOptions,
    headers,
  });

  const data = await res.json();

  if (!res.ok) {
    throw new Error(data.error || "Request failed");
  }

  return data;
}

// ========== Auth API ==========

export const authAPI = {
  sendSMSCode: (phone: string) =>
    fetchAPI("/api/auth/send-sms-code", {
      method: "POST",
      body: JSON.stringify({ phone }),
    }),

  loginByPhone: (phone: string, code: string) =>
    fetchAPI("/api/auth/login-by-phone", {
      method: "POST",
      body: JSON.stringify({ phone, code }),
    }),

  loginByEmail: (email: string, password: string) =>
    fetchAPI("/api/auth/login-by-email", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    }),

  registerByEmail: (email: string, password: string, name: string) =>
    fetchAPI("/api/auth/register-by-email", {
      method: "POST",
      body: JSON.stringify({ email, password, name }),
    }),

  getMe: (token: string) => fetchAPI("/api/me", { token }),

  updateMe: (token: string, data: { name?: string; email?: string }) =>
    fetchAPI("/api/me", {
      method: "PATCH",
      token,
      body: JSON.stringify(data),
    }),
};

// ========== Links API ==========

export const linksAPI = {
  list: (token: string, page = 1, pageSize = 20, search = "", folderId = 0, tagId = 0) => {
    const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) });
    if (search) params.set("search", search);
    if (folderId > 0) params.set("folder_id", String(folderId));
    if (tagId > 0) params.set("tag_id", String(tagId));
    return fetchAPI(`/api/links?${params.toString()}`, { token });
  },

  create: (token: string, data: {
    original_url: string;
    short_code?: string;
    title?: string;
    description?: string;
    domain?: string;
    password?: string;
    expires_at?: string;
    folder_id?: number;
    tag_ids?: number[];
    utm_source?: string;
    utm_medium?: string;
    utm_campaign?: string;
    utm_term?: string;
    utm_content?: string;
    ios_url?: string;
    android_url?: string;
  }) =>
    fetchAPI("/api/links", {
      method: "POST",
      token,
      body: JSON.stringify(data),
    }),

  get: (token: string, id: string) =>
    fetchAPI(`/api/links/${id}`, { token }),

  update: (token: string, id: number, data: Record<string, unknown>) =>
    fetchAPI(`/api/links/${id}`, {
      method: "PATCH",
      token,
      body: JSON.stringify(data),
    }),

  delete: (token: string, id: number) =>
    fetchAPI(`/api/links/${id}`, {
      method: "DELETE",
      token,
    }),
};

// ========== Analytics API ==========

export const analyticsAPI = {
  overview: (token: string) =>
    fetchAPI("/api/analytics/overview", { token }),

  platforms: (token: string) =>
    fetchAPI("/api/analytics/platforms", { token }),

  daily: (token: string) =>
    fetchAPI("/api/analytics/daily", { token }),

  events: (token: string, page = 1, pageSize = 20) =>
    fetchAPI(`/api/analytics/events?page=${page}&page_size=${pageSize}`, { token }),

  customers: (token: string) =>
    fetchAPI("/api/analytics/customers", { token }),
};

// ========== Folders API ==========

export const foldersAPI = {
  list: (token: string) =>
    fetchAPI("/api/folders", { token }),

  create: (token: string, name: string) =>
    fetchAPI("/api/folders", {
      method: "POST",
      token,
      body: JSON.stringify({ name }),
    }),

  update: (token: string, id: number, name: string) =>
    fetchAPI(`/api/folders/${id}`, {
      method: "PATCH",
      token,
      body: JSON.stringify({ name }),
    }),

  delete: (token: string, id: number) =>
    fetchAPI(`/api/folders/${id}`, {
      method: "DELETE",
      token,
    }),
};

// ========== Tags API ==========

export const tagsAPI = {
  list: (token: string) =>
    fetchAPI("/api/tags", { token }),

  create: (token: string, name: string, color?: string) =>
    fetchAPI("/api/tags", {
      method: "POST",
      token,
      body: JSON.stringify({ name, color }),
    }),

  delete: (token: string, id: number) =>
    fetchAPI(`/api/tags/${id}`, {
      method: "DELETE",
      token,
    }),

  addToLink: (token: string, linkId: number, tagId: number) =>
    fetchAPI(`/api/links/${linkId}/tags`, {
      method: "POST",
      token,
      body: JSON.stringify({ tag_id: tagId }),
    }),

  removeFromLink: (token: string, linkId: number, tagId: number) =>
    fetchAPI(`/api/links/${linkId}/tags/${tagId}`, {
      method: "DELETE",
      token,
    }),
};

// ========== Domains API ==========

export const domainsAPI = {
  list: (token: string) =>
    fetchAPI("/api/domains", { token }),

  create: (token: string, name: string) =>
    fetchAPI("/api/domains", {
      method: "POST",
      token,
      body: JSON.stringify({ name }),
    }),

  verify: (token: string, id: number) =>
    fetchAPI(`/api/domains/${id}/verify`, {
      method: "POST",
      token,
    }),

  delete: (token: string, id: number) =>
    fetchAPI(`/api/domains/${id}`, {
      method: "DELETE",
      token,
    }),
};

// ========== UTM Templates API ==========

export const utmAPI = {
  list: (token: string) =>
    fetchAPI("/api/utm-templates", { token }),

  create: (token: string, data: {
    name: string;
    utm_source?: string;
    utm_medium?: string;
    utm_campaign?: string;
    utm_term?: string;
    utm_content?: string;
  }) =>
    fetchAPI("/api/utm-templates", {
      method: "POST",
      token,
      body: JSON.stringify(data),
    }),

  delete: (token: string, id: number) =>
    fetchAPI(`/api/utm-templates/${id}`, {
      method: "DELETE",
      token,
    }),
};
