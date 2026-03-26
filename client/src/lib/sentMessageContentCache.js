const SENT_MESSAGE_CONTENT_STORAGE_KEY = "sent_message_content_cache";

function canUseStorage() {
  return typeof localStorage !== "undefined";
}

function readStore() {
  if (!canUseStorage()) {
    return {};
  }

  const raw = localStorage.getItem(SENT_MESSAGE_CONTENT_STORAGE_KEY);
  if (!raw) {
    return {};
  }

  try {
    const parsed = JSON.parse(raw);
    return parsed && typeof parsed === "object" ? parsed : {};
  } catch (_err) {
    localStorage.removeItem(SENT_MESSAGE_CONTENT_STORAGE_KEY);
    return {};
  }
}

function writeStore(nextStore) {
  if (!canUseStorage()) {
    return;
  }

  const entries = Object.entries(nextStore || {});
  if (entries.length === 0) {
    localStorage.removeItem(SENT_MESSAGE_CONTENT_STORAGE_KEY);
    return;
  }

  localStorage.setItem(SENT_MESSAGE_CONTENT_STORAGE_KEY, JSON.stringify(Object.fromEntries(entries.slice(-200))));
}

export function saveSentMessageContent({ clientID = 0, serverID = 0, to = "", contentKind = "text", body }) {
  if (!body) {
    return;
  }

  const normalizedClientID = Number(clientID);
  const normalizedServerID = Number(serverID);
  if ((!Number.isInteger(normalizedClientID) || normalizedClientID <= 0) && (!Number.isInteger(normalizedServerID) || normalizedServerID <= 0)) {
    return;
  }

  const store = readStore();
  const entry = {
    client_id: Number.isInteger(normalizedClientID) && normalizedClientID > 0 ? normalizedClientID : 0,
    server_id: Number.isInteger(normalizedServerID) && normalizedServerID > 0 ? normalizedServerID : 0,
    to,
    content_kind: contentKind || "text",
    body,
    saved_at: new Date().toISOString(),
  };

  if (entry.client_id > 0) {
    store[`client:${entry.client_id}`] = entry;
  }
  if (entry.server_id > 0) {
    store[`server:${entry.server_id}`] = entry;
  }

  writeStore(store);
}

export function linkStoredMessageID(clientID, serverID) {
  const normalizedClientID = Number(clientID);
  const normalizedServerID = Number(serverID);
  if (!Number.isInteger(normalizedClientID) || normalizedClientID <= 0 || !Number.isInteger(normalizedServerID) || normalizedServerID <= 0) {
    return;
  }

  const store = readStore();
  const entry = store[`client:${normalizedClientID}`];
  if (!entry) {
    return;
  }

  const updated = {
    ...entry,
    server_id: normalizedServerID,
  };
  store[`client:${normalizedClientID}`] = updated;
  store[`server:${normalizedServerID}`] = updated;
  writeStore(store);
}

export function resolveSentMessageContent({ clientID = 0, serverID = 0 }) {
  const store = readStore();
  const normalizedClientID = Number(clientID);
  if (Number.isInteger(normalizedClientID) && normalizedClientID > 0) {
    const entry = store[`client:${normalizedClientID}`];
    if (entry) {
      return entry;
    }
  }

  const normalizedServerID = Number(serverID);
  if (Number.isInteger(normalizedServerID) && normalizedServerID > 0) {
    const entry = store[`server:${normalizedServerID}`];
    if (entry) {
      return entry;
    }
  }

  return null;
}
