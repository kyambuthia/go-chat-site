import { buildApiURL, getViteEnv, toAmountCents } from "./lib/runtimeConfig.js";

let token = typeof localStorage !== "undefined" ? localStorage.getItem("token") : null;
let authErrorHandler = null;

const getAuthHeaders = () => {
  const headers = {
    "Content-Type": "application/json",
  };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  return headers;
};

export const setToken = (newToken) => {
  token = newToken;
};

export const setAuthErrorHandler = (handler) => {
  authErrorHandler = handler;
};

const apiRequest = async (url, options = {}) => {
  const response = await fetch(buildApiURL(url, { apiBaseURL: getViteEnv("VITE_API_BASE_URL") }), {
    ...options,
    headers: {
      ...options.headers,
      ...getAuthHeaders(),
    },
  });

  if (!response.ok) {
    let errMsg = `Request failed with status ${response.status}`;
    try {
      const payload = await response.json();
      if (payload?.error) {
        errMsg = payload.error;
        if (payload.error === "invalid token" && typeof authErrorHandler === "function") {
          authErrorHandler(payload.error);
        }
      }
    } catch (_err) {
      // ignore non-json errors
    }
    throw new Error(errMsg);
  }

  if (response.status === 204 || response.headers.get("content-length") === "0") {
    return null;
  }

  return response.json();
};

export const registerUser = (username, password) =>
  apiRequest("/api/register", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });

export const loginUser = async (username, password) => {
  const data = await apiRequest("/api/login", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });
  if (data?.token) {
    setToken(data.token);
  }
  return data;
};

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
