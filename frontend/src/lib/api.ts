// The contract for the API base URL:
//
//   undefined -> development fallback (the Go API on :8080)
//   ""        -> same origin, which is how production runs: nginx serves the frontend and proxies
//                /api/ to the Go backend, so requests must stay relative
//   anything  -> that explicit base URL
//
// `??` (not `||`) is what makes the empty string meaningful, so the production build has to set the
// variable explicitly - see the NEXT_PUBLIC_API_URL env in .github/workflows/ci.yml.
const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

interface FetchOptions extends RequestInit {
  token?: string;
}

/**
 * Reads a response that is expected to be JSON.
 *
 * Calling `res.json()` on an HTML body throws `Unexpected token '<', "<!DOCTYPE "... is not valid JSON`,
 * which says nothing about what actually happened: the request probably never reached the Go API and was
 * answered by Next.js or a proxy instead. Checking the content type first turns that into an error that
 * names the URL, the status and the content type, so the cause is visible without opening DevTools.
 */
async function readJSON(res: Response, url: string) {
  const contentType = res.headers.get("content-type") ?? "(none)";

  if (!contentType.includes("application/json")) {
    const body = (await res.text()).slice(0, 200).replace(/\s+/g, " ");
    throw new Error(
      `${url} returned ${res.status} ${res.statusText} with content-type ${contentType} instead of JSON. ` +
        `Check that NEXT_PUBLIC_API_URL points at the API and that it is running. Body starts with: ${body}`
    );
  }

  return res.json();
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

  const url = `${API_URL}${path}`;

  let res: Response;
  try {
    res = await fetch(url, { ...fetchOptions, headers });
  } catch (cause) {
    // A network-level failure (API not running, wrong host, CORS preflight rejected) never produces a
    // Response at all, so it needs its own message rather than surfacing as a bare "Failed to fetch".
    throw new Error(`cannot reach the API at ${url}: ${cause instanceof Error ? cause.message : cause}`);
  }

  const data = await readJSON(res, url);

  if (!res.ok) {
    throw new Error(data.error || `request failed with status ${res.status}`);
  }

  return data;
}

/** Performs a request that returns CSV rather than JSON, used by the link export. */
export async function fetchCSV(path: string, token: string): Promise<string> {
  const url = `${API_URL}${path}`;

  let res: Response;
  try {
    res = await fetch(url, { headers: { Authorization: `Bearer ${token}` } });
  } catch (cause) {
    throw new Error(`cannot reach the API at ${url}: ${cause instanceof Error ? cause.message : cause}`);
  }

  if (!res.ok) {
    throw new Error(`export failed with status ${res.status}`);
  }

  return res.text();
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
  list: (token: string, page = 1, pageSize = 20, search = "", folderId = 0, tagId = 0, workspaceId = 0, sort = "created_desc") => {
    const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) });
    if (search) params.set("search", search);
    if (folderId > 0) params.set("folder_id", String(folderId));
    if (tagId > 0) params.set("tag_id", String(tagId));
    if (workspaceId > 0) params.set("workspace_id", String(workspaceId));
    if (sort) params.set("sort", sort);
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
    workspace_id?: number;
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

  batchDelete: (token: string, ids: number[]) =>
    fetchAPI("/api/links/batch-delete", {
      method: "POST",
      token,
      body: JSON.stringify({ ids }),
    }),

  batchTag: (token: string, ids: number[], tagId: number) =>
    fetchAPI("/api/links/batch-tag", {
      method: "POST",
      token,
      body: JSON.stringify({ ids, tag_id: tagId }),
    }),

  preview: (token: string, url: string): Promise<{
    preview: { title: string; description: string; image_url: string; favicon_url: string };
  }> =>
    fetchAPI("/api/links/preview", {
      method: "POST",
      token,
      body: JSON.stringify({ url }),
    }),

  /** Downloads the user's links as CSV. */
  export: (token: string) => fetchCSV("/api/links/export", token),
};

// ========== Analytics API ==========

export const analyticsAPI = {
  overview: (token: string) =>
    fetchAPI("/api/analytics/overview", { token }),

  platforms: (token: string) =>
    fetchAPI("/api/analytics/platforms", { token }),

  /** Platform breakdown for a single link, used by the link detail page. */
  platformsForLink: (token: string, linkId: number): Promise<{ platforms: { platform: string; count: number }[] }> =>
    fetchAPI(`/api/analytics/platforms?link_id=${linkId}`, { token }),

  daily: (token: string) =>
    fetchAPI("/api/analytics/daily", { token }),

  /** Daily clicks for a single link, used by the link detail page. */
  dailyForLink: (token: string, linkId: number): Promise<{ daily: { date: string; count: number }[] }> =>
    fetchAPI(`/api/analytics/daily?link_id=${linkId}`, { token }),

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

// ========== API Tokens ==========

export const tokensAPI = {
  list: (token: string) =>
    fetchAPI("/api/api-tokens", { token }),

  create: (token: string, name: string) =>
    fetchAPI("/api/api-tokens", {
      method: "POST",
      token,
      body: JSON.stringify({ name }),
    }),

  delete: (token: string, id: number) =>
    fetchAPI(`/api/api-tokens/${id}`, {
      method: "DELETE",
      token,
    }),
};

// ========== Workspaces API ==========

export const workspacesAPI = {
  list: (token: string) =>
    fetchAPI("/api/workspaces", { token }),

  create: (token: string, name: string, slug: string) =>
    fetchAPI("/api/workspaces", {
      method: "POST",
      token,
      body: JSON.stringify({ name, slug }),
    }),

  update: (token: string, id: number, data: { name?: string; slug?: string }) =>
    fetchAPI(`/api/workspaces/${id}`, {
      method: "PATCH",
      token,
      body: JSON.stringify(data),
    }),

  delete: (token: string, id: number) =>
    fetchAPI(`/api/workspaces/${id}`, {
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
