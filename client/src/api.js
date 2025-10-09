let token = null;

const getAuthHeaders = () => ({
  "Content-Type": "application/json",
  Authorization: `Bearer ${token}`,
});

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
    // You can handle errors more globally here
    console.error("API request failed:", response.status, response.statusText);
    throw new Error(`Request failed with status ${response.status}`);
  }

  // For 204 No Content, etc.
  if (response.status === 204 || response.headers.get("content-length") === "0") {
    return null; 
  }

  return response.json();
};

export const registerUser = (username, password) => {
  return apiRequest("/api/register", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });
};

export const loginUser = async (username, password) => {
  const data = await apiRequest("/api/login", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });
  if (data && data.token) {
    setToken(data.token);
  }
  return data;
};

export const getContacts = () => apiRequest("/api/contacts");



export const getInvites = () => apiRequest("/api/invites");

export const acceptInvite = (inviteID) => {
  return apiRequest("/api/invites/accept", {
    method: "POST",
    body: JSON.stringify({ invite_id: inviteID }),
  });
};

export const rejectInvite = (inviteID) => {
  return apiRequest("/api/invites/reject", {
    method: "POST",
    body: JSON.stringify({ invite_id: inviteID }),
  });
};
