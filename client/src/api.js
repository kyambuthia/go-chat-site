let token = localStorage.getItem("token");

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

const apiRequest = async (url, options = {}) => {
  const response = await fetch(url, {
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

export const getWallet = () => apiRequest("/api/wallet");

export const sendMoney = (username, amount) =>
  apiRequest("/api/wallet/send", {
    method: "POST",
    body: JSON.stringify({ username, amount }),
  });
