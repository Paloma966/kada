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
};

// ========== Links API ==========

export const linksAPI = {
  list: (token: string, page = 1, pageSize = 20) =>
    fetchAPI(`/api/links?page=${page}&page_size=${pageSize}`, { token }),

  create: (token: string, data: {
    original_url: string;
    title?: string;
    description?: string;
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
