import { buildApiURL, getViteEnv, toAmountCents } from "./lib/runtimeConfig.js";

const AUTH_STORAGE_KEY = "auth_session";
const LEGACY_TOKEN_STORAGE_KEY = "token";
const ACCESS_TOKEN_REFRESH_SKEW_MS = 30_000;

const emptyAuthState = () => ({
  accessToken: null,
  refreshToken: null,
  accessTokenExpiresAt: null,
  refreshTokenExpiresAt: null,
  session: null,
});

const canUseStorage = () => typeof localStorage !== "undefined";

const normalizeAuthState = (value = {}) => ({
  accessToken: value?.accessToken || value?.access_token || value?.token || null,
  refreshToken: value?.refreshToken || value?.refresh_token || null,
  accessTokenExpiresAt: value?.accessTokenExpiresAt || value?.access_token_expires_at || null,
  refreshTokenExpiresAt: value?.refreshTokenExpiresAt || value?.refresh_token_expires_at || null,
  session: value?.session || null,
});

const readStoredAuthState = () => {
  if (!canUseStorage()) {
    return emptyAuthState();
  }

  const raw = localStorage.getItem(AUTH_STORAGE_KEY);
  if (raw) {
    try {
      return normalizeAuthState(JSON.parse(raw));
    } catch (_err) {
      localStorage.removeItem(AUTH_STORAGE_KEY);
    }
  }

  const legacyToken = localStorage.getItem(LEGACY_TOKEN_STORAGE_KEY);
  if (!legacyToken) {
    return emptyAuthState();
  }

  return normalizeAuthState({ accessToken: legacyToken });
};

let authState = readStoredAuthState();
let authErrorHandler = null;
let refreshPromise = null;

const syncStoredAuthState = () => {
  if (!canUseStorage()) {
    return;
  }

  if (authState.accessToken || authState.refreshToken || authState.session) {
    localStorage.setItem(
      AUTH_STORAGE_KEY,
      JSON.stringify({
        accessToken: authState.accessToken,
        refreshToken: authState.refreshToken,
        accessTokenExpiresAt: authState.accessTokenExpiresAt,
        refreshTokenExpiresAt: authState.refreshTokenExpiresAt,
        session: authState.session,
      }),
    );
  } else {
    localStorage.removeItem(AUTH_STORAGE_KEY);
  }

  if (authState.accessToken) {
    localStorage.setItem(LEGACY_TOKEN_STORAGE_KEY, authState.accessToken);
  } else {
    localStorage.removeItem(LEGACY_TOKEN_STORAGE_KEY);
  }
};

const setAuthState = (nextState) => {
  authState = normalizeAuthState(nextState);
  syncStoredAuthState();
  return authState;
};

const applySessionPayload = (payload = {}) =>
  setAuthState({
    accessToken: payload.access_token || payload.token || null,
    refreshToken: payload.refresh_token || null,
    accessTokenExpiresAt: payload.access_token_expires_at || null,
    refreshTokenExpiresAt: payload.refresh_token_expires_at || null,
    session: payload.session || null,
  });

const isExpired = (timestamp, skewMs = 0) => {
  if (!timestamp) {
    return false;
  }
  const parsed = Date.parse(timestamp);
  if (!Number.isFinite(parsed)) {
    return false;
  }
  return parsed <= Date.now() + skewMs;
};

const createAuthFailure = (message) => {
  const error = new Error(message);
  error.isAuthFailure = true;
  return error;
};

const getBaseHeaders = (withAuth = true) => {
  const headers = {
    "Content-Type": "application/json",
  };
  if (withAuth && authState.accessToken) {
    headers.Authorization = `Bearer ${authState.accessToken}`;
  }
  return headers;
};

const sendAuthFailure = (message) => {
  if (typeof authErrorHandler === "function") {
    authErrorHandler(message);
  }
};

const parseErrorResponse = async (response) => {
  let payload = null;
  let errMsg = `Request failed with status ${response.status}`;

  try {
    payload = await response.json();
    if (payload?.error) {
      errMsg = payload.error;
    }
  } catch (_err) {
    // ignore non-json errors
  }

  return { payload, errMsg };
};

const parseJSONResponse = async (response) => {
  if (response.status === 204 || response.headers.get("content-length") === "0") {
    return null;
  }

  return response.json();
};

const performRequest = (url, options = {}, withAuth = true) =>
  fetch(buildApiURL(url, { apiBaseURL: getViteEnv("VITE_API_BASE_URL") }), {
    ...options,
    headers: {
      ...getBaseHeaders(withAuth),
      ...options.headers,
    },
  });

export const setToken = (newToken) =>
  setAuthState({
    ...authState,
    accessToken: newToken || null,
  });

export const setAuthErrorHandler = (handler) => {
  authErrorHandler = handler;
};

export const getAccessToken = () => authState.accessToken;

export const hasStoredSession = () => Boolean(authState.accessToken || authState.refreshToken);

export const clearAuthState = () => {
  refreshPromise = null;
  setAuthState(emptyAuthState());
};

export const ensureAccessToken = async (deviceLabel = "") => {
  if (authState.accessToken && !isExpired(authState.accessTokenExpiresAt, ACCESS_TOKEN_REFRESH_SKEW_MS)) {
    return authState.accessToken;
  }

  if (!authState.refreshToken) {
    return authState.accessToken;
  }

  const data = await refreshSession(deviceLabel);
  return data.access_token || data.token || null;
};

export const refreshSession = async (deviceLabel = "") => {
  if (!authState.refreshToken) {
    throw createAuthFailure("refresh token unavailable");
  }
  if (isExpired(authState.refreshTokenExpiresAt)) {
    clearAuthState();
    throw createAuthFailure("session expired");
  }
  if (refreshPromise) {
    return refreshPromise;
  }

  refreshPromise = (async () => {
    const response = await performRequest(
      "/api/auth/refresh",
      {
        method: "POST",
        body: JSON.stringify({
          refresh_token: authState.refreshToken,
          device_label: deviceLabel,
        }),
      },
      false,
    );

    if (!response.ok) {
      const { errMsg } = await parseErrorResponse(response);
      if (response.status === 401) {
        clearAuthState();
        throw createAuthFailure(errMsg);
      }
      throw new Error(errMsg);
    }

    const payload = await parseJSONResponse(response);
    applySessionPayload(payload);
    return payload;
  })();

  try {
    return await refreshPromise;
  } finally {
    refreshPromise = null;
  }
};

const apiRequest = async (url, options = {}, requestOptions = {}) => {
  const { allowRefresh = true, withAuth = true } = requestOptions;

  const response = await performRequest(url, options, withAuth);

  if (!response.ok) {
    const { payload, errMsg } = await parseErrorResponse(response);
    const shouldRefresh =
      withAuth &&
      allowRefresh &&
      payload?.error === "invalid token" &&
      authState.refreshToken &&
      url !== "/api/auth/refresh";

    if (shouldRefresh) {
      try {
        await refreshSession();
      } catch (error) {
        if (error?.isAuthFailure) {
          sendAuthFailure(error.message);
        }
        throw error;
      }

      const retried = await performRequest(url, options, withAuth);
      if (!retried.ok) {
        const { payload: retryPayload, errMsg: retryErrMsg } = await parseErrorResponse(retried);
        if (retryPayload?.error === "invalid token") {
          sendAuthFailure(retryErrMsg);
        }
        throw new Error(retryErrMsg);
      }
      return parseJSONResponse(retried);
    }

    if (payload?.error === "invalid token") {
      sendAuthFailure(errMsg);
    }
    throw new Error(errMsg);
  }

  return parseJSONResponse(response);
};

export const registerUser = (username, password) =>
  apiRequest("/api/register", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  }, { withAuth: false });

export const loginUser = async (username, password, deviceLabel = "") => {
  const data = await apiRequest("/api/login", {
    method: "POST",
    body: JSON.stringify({ username, password, device_label: deviceLabel }),
  }, { withAuth: false, allowRefresh: false });
  if (data?.token || data?.access_token || data?.refresh_token) {
    applySessionPayload(data);
  }
  return data;
};

export const logoutSession = () =>
  apiRequest("/api/logout", {
    method: "POST",
  });

export const getSessions = () => apiRequest("/api/sessions");

export const revokeSession = (sessionID) =>
  apiRequest("/api/sessions", {
    method: "DELETE",
    body: JSON.stringify({ session_id: sessionID }),
  });

export const getDevices = () => apiRequest("/api/devices");

export const registerDeviceIdentity = (payload) =>
  apiRequest("/api/devices", {
    method: "POST",
    body: JSON.stringify(payload),
  });

export const publishDevicePrekeys = (deviceID, prekeys) =>
  apiRequest("/api/messaging/prekeys", {
    method: "POST",
    body: JSON.stringify({ device_id: deviceID, prekeys }),
  });

export const revokeDeviceIdentity = (deviceID) =>
  apiRequest("/api/devices", {
    method: "DELETE",
    body: JSON.stringify({ device_id: deviceID }),
  });

export const getContacts = () => apiRequest("/api/contacts");

export const addContact = (username) =>
  apiRequest("/api/contacts", {
    method: "POST",
    body: JSON.stringify({ username }),
  });

export const removeContact = (contactID) =>
  apiRequest("/api/contacts", {
    method: "DELETE",
    body: JSON.stringify({ contact_id: contactID }),
  });

export const sendInvite = (username) =>
  apiRequest("/api/invites/send", {
    method: "POST",
    body: JSON.stringify({ username }),
  });

export const getInvites = () => apiRequest("/api/invites");

export const acceptInvite = (inviteID) =>
  apiRequest("/api/invites/accept", {
    method: "POST",
    body: JSON.stringify({ invite_id: inviteID }),
  });

export const rejectInvite = (inviteID) =>
  apiRequest("/api/invites/reject", {
    method: "POST",
    body: JSON.stringify({ invite_id: inviteID }),
  });

export const getMe = () => apiRequest("/api/me");

export const updateMe = (profile) =>
  apiRequest("/api/me", {
    method: "PATCH",
    body: JSON.stringify(profile),
  });

export const getWallet = () => apiRequest("/api/wallet");

export const getWalletTransfers = (params = {}) =>
  apiRequest(`/api/wallet/transfers${buildQueryString(params)}`);

const buildQueryString = (params = {}) => {
  const query = new URLSearchParams();

  if (params.limit) {
    query.set("limit", String(params.limit));
  }
  if (params.beforeID) {
    query.set("before_id", String(params.beforeID));
  }
  if (params.afterID) {
    query.set("after_id", String(params.afterID));
  }
  if (params.withUserID) {
    query.set("with_user_id", String(params.withUserID));
  }
  if (params.unreadOnly) {
    query.set("unread_only", "true");
  }

  const encoded = query.toString();
  return encoded ? `?${encoded}` : "";
};

export const getInbox = (params = {}) => apiRequest(`/api/messages/inbox${buildQueryString(params)}`);

export const getOutbox = (params = {}) => apiRequest(`/api/messages/outbox${buildQueryString(params)}`);

export const getMessageThreads = (params = {}) => apiRequest(`/api/messaging/threads${buildQueryString(params)}`);

export const syncMessages = (params = {}) => apiRequest(`/api/messaging/sync${buildQueryString(params)}`);

export const markMessageRead = (messageID) =>
  apiRequest("/api/messages/read", {
    method: "POST",
    body: JSON.stringify({ message_id: messageID }),
  });

export const markThreadRead = (withUserID) =>
  apiRequest("/api/messaging/read-thread", {
    method: "POST",
    body: JSON.stringify({ with_user_id: withUserID }),
  });

export const markMessageDelivered = (messageID) =>
  apiRequest("/api/messages/delivered", {
    method: "POST",
    body: JSON.stringify({ message_id: messageID }),
  });

export const sendMoney = (username, amount) =>
  apiRequest("/api/wallet/send", {
    method: "POST",
    body: JSON.stringify({ username, amount_cents: toAmountCents(amount) }),
  });
