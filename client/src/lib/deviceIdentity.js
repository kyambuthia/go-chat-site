export const DEFAULT_DEVICE_ALGORITHM = "x3dh-ed25519-x25519-v1";
export const DEFAULT_ENVELOPE_VERSION = "x3dh-dr-v1";

function requireCrypto() {
  if (globalThis.crypto?.getRandomValues) {
    return globalThis.crypto;
  }
  throw new Error("Web Crypto is unavailable in this runtime.");
}

function randomBytes(length) {
  const bytes = new Uint8Array(length);
  requireCrypto().getRandomValues(bytes);
  return bytes;
}

function encodeBytesToBase64(bytes) {
  if (typeof Buffer !== "undefined") {
    return Buffer.from(bytes).toString("base64");
  }

  if (typeof btoa === "function") {
    let binary = "";
    for (const byte of bytes) {
      binary += String.fromCharCode(byte);
    }
    return btoa(binary);
  }

  throw new Error("Base64 encoding is unavailable in this runtime.");
}

function encodeStringToBase64(value) {
  return encodeBytesToBase64(new TextEncoder().encode(value));
}

function randomPositiveInt() {
  const bytes = randomBytes(4);
  const view = new DataView(bytes.buffer);
  return (view.getUint32(0, false) % 2147483646) + 1;
}

function randomBase64(length) {
  return encodeBytesToBase64(randomBytes(length));
}

export function selectPreferredActiveDevice(devices = []) {
  const activeDevices = devices.filter((device) => device?.state === "active");
  if (activeDevices.length === 0) {
    return null;
  }
  return activeDevices.find((device) => device.current_session) || activeDevices[0];
}

export function selectPreferredRecipientDevice(directory) {
  const devices = directory?.devices || [];
  return selectPreferredActiveDevice(devices);
}

export function generateDeviceIdentityBundle(options = {}) {
  const algorithm = options.algorithm?.trim() || DEFAULT_DEVICE_ALGORITHM;
  const prekeyCount = Number.isInteger(options.prekeyCount) && options.prekeyCount > 0 ? options.prekeyCount : 5;
  const signedPrekeyID = randomPositiveInt();
  const prekeys = Array.from({ length: prekeyCount }, (_, index) => ({
    prekey_id: signedPrekeyID + index + 1,
    public_key: randomBase64(32),
  }));

  return {
    algorithm,
    identity_key: randomBase64(32),
    signed_prekey_id: signedPrekeyID,
    signed_prekey: randomBase64(32),
    signed_prekey_signature: randomBase64(64),
    prekeys,
  };
}

export function formatPrekeysForTextarea(prekeys = []) {
  return prekeys.map((prekey) => `${prekey.prekey_id}:${prekey.public_key}`).join("\n");
}

// This is an opaque transport scaffold only. It preserves the future envelope
// shape without claiming to provide real cryptographic secrecy yet.
export function buildOpaqueEnvelopeScaffold({
  body,
  senderDeviceID,
  recipientDeviceID,
  encryptionVersion = DEFAULT_ENVELOPE_VERSION,
}) {
  if (!body) {
    throw new Error("body is required");
  }
  if (!Number.isInteger(senderDeviceID) || senderDeviceID <= 0) {
    throw new Error("senderDeviceID is required");
  }
  if (!Number.isInteger(recipientDeviceID) || recipientDeviceID <= 0) {
    throw new Error("recipientDeviceID is required");
  }

  return {
    ciphertext: encodeStringToBase64(JSON.stringify({
      kind: "opaque-envelope-scaffold",
      body,
      nonce: randomBase64(12),
    })),
    encryption_version: encryptionVersion,
    sender_device_id: senderDeviceID,
    recipient_device_id: recipientDeviceID,
  };
}
