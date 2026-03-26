const DEVICE_KEY_STORAGE_KEY = "device_private_bundles";

function canUseStorage() {
  return typeof localStorage !== "undefined";
}

function readStore() {
  if (!canUseStorage()) {
    return {};
  }

  const raw = localStorage.getItem(DEVICE_KEY_STORAGE_KEY);
  if (!raw) {
    return {};
  }

  try {
    const parsed = JSON.parse(raw);
    return parsed && typeof parsed === "object" ? parsed : {};
  } catch (_err) {
    localStorage.removeItem(DEVICE_KEY_STORAGE_KEY);
    return {};
  }
}

function writeStore(nextStore) {
  if (!canUseStorage()) {
    return;
  }

  const keys = Object.keys(nextStore || {});
  if (keys.length === 0) {
    localStorage.removeItem(DEVICE_KEY_STORAGE_KEY);
    return;
  }

  localStorage.setItem(DEVICE_KEY_STORAGE_KEY, JSON.stringify(nextStore));
}

export function saveLocalDeviceBundle({ userID = 0, deviceID, bundle }) {
  const normalizedDeviceID = Number(deviceID);
  if (!Number.isInteger(normalizedDeviceID) || normalizedDeviceID <= 0) {
    throw new Error("deviceID is required");
  }
  if (!bundle || typeof bundle !== "object") {
    throw new Error("bundle is required");
  }

  const store = readStore();
  store[String(normalizedDeviceID)] = {
    user_id: Number.isInteger(Number(userID)) ? Number(userID) : 0,
    device_id: normalizedDeviceID,
    bundle,
    saved_at: new Date().toISOString(),
  };
  writeStore(store);
}

export function getLocalDeviceBundle(deviceID) {
  const entry = readStore()[String(Number(deviceID))];
  return entry?.bundle || null;
}

export function hasLocalDeviceBundle(deviceID) {
  return !!getLocalDeviceBundle(deviceID);
}

export function removeLocalDeviceBundle(deviceID) {
  const normalizedDeviceID = String(Number(deviceID));
  const store = readStore();
  if (!Object.prototype.hasOwnProperty.call(store, normalizedDeviceID)) {
    return;
  }
  delete store[normalizedDeviceID];
  writeStore(store);
}

export function listLocalDeviceBundleIDs() {
  return Object.values(readStore())
    .map((entry) => Number(entry?.device_id))
    .filter((deviceID) => Number.isInteger(deviceID) && deviceID > 0);
}
