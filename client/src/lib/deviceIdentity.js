export const DEFAULT_DEVICE_ALGORITHM = "x3dh-ed25519-x25519-v1";
export const DEFAULT_ENVELOPE_VERSION = "x3dh-dr-v1";

const SIGNING_ALGORITHM = { name: "ECDSA", namedCurve: "P-256" };
const AGREEMENT_ALGORITHM = { name: "ECDH", namedCurve: "P-256" };
const SIGNATURE_ALGORITHM = { name: "ECDSA", hash: "SHA-256" };
// This is a bootstrap encrypted transport for Phase 4. The ADR still targets
// full X3DH + Double Ratchet, but this gives the client real local key
// generation and opaque ciphertext production without changing the server
// contract yet.
const ENVELOPE_SCHEME = "p256-ecdh-aesgcm-v1";

function requireCrypto() {
  if (globalThis.crypto?.getRandomValues && globalThis.crypto?.subtle) {
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

function decodeBase64ToBytes(value) {
  if (!value) {
    return new Uint8Array();
  }

  if (typeof Buffer !== "undefined") {
    return new Uint8Array(Buffer.from(value, "base64"));
  }

  if (typeof atob === "function") {
    const decoded = atob(value);
    return Uint8Array.from(decoded, (character) => character.charCodeAt(0));
  }

  throw new Error("Base64 decoding is unavailable in this runtime.");
}

function encodeText(value) {
  return new TextEncoder().encode(value);
}

function decodeText(bytes) {
  return new TextDecoder().decode(bytes);
}

function randomPositiveInt() {
  const bytes = randomBytes(4);
  const view = new DataView(bytes.buffer);
  return (view.getUint32(0, false) % 2147483646) + 1;
}

async function exportKeyBase64(format, key) {
  const subtle = requireCrypto().subtle;
  const exported = await subtle.exportKey(format, key);
  return encodeBytesToBase64(new Uint8Array(exported));
}

async function importSigningPrivateKey(pkcs8Base64, usages = ["sign"]) {
  return requireCrypto().subtle.importKey(
    "pkcs8",
    decodeBase64ToBytes(pkcs8Base64),
    SIGNING_ALGORITHM,
    false,
    usages,
  );
}

async function importSigningPublicKey(spkiBase64, usages = ["verify"]) {
  return requireCrypto().subtle.importKey(
    "spki",
    decodeBase64ToBytes(spkiBase64),
    SIGNING_ALGORITHM,
    false,
    usages,
  );
}

async function importAgreementPrivateKey(pkcs8Base64, usages = ["deriveBits"]) {
  return requireCrypto().subtle.importKey(
    "pkcs8",
    decodeBase64ToBytes(pkcs8Base64),
    AGREEMENT_ALGORITHM,
    false,
    usages,
  );
}

async function importAgreementPublicKey(spkiBase64) {
  return requireCrypto().subtle.importKey(
    "spki",
    decodeBase64ToBytes(spkiBase64),
    AGREEMENT_ALGORITHM,
    false,
    [],
  );
}

function normalizeSignaturePayload(payload) {
  return JSON.stringify({
    scheme: payload.scheme,
    sender_device_id: payload.sender_device_id,
    recipient_device_id: payload.recipient_device_id,
    ephemeral_key: payload.ephemeral_key,
    iv: payload.iv,
    ciphertext: payload.ciphertext,
  });
}

async function signEnvelopePayload(payload, signingPrivateKeyBase64) {
  const signingKey = await importSigningPrivateKey(signingPrivateKeyBase64);
  const signature = await requireCrypto().subtle.sign(
    SIGNATURE_ALGORITHM,
    signingKey,
    encodeText(normalizeSignaturePayload(payload)),
  );
  return encodeBytesToBase64(new Uint8Array(signature));
}

async function verifyEnvelopePayload(payload, signatureBase64, signingPublicKeyBase64) {
  if (!signatureBase64 || !signingPublicKeyBase64) {
    return false;
  }

  const publicKey = await importSigningPublicKey(signingPublicKeyBase64);
  return requireCrypto().subtle.verify(
    SIGNATURE_ALGORITHM,
    publicKey,
    decodeBase64ToBytes(signatureBase64),
    encodeText(normalizeSignaturePayload(payload)),
  );
}

async function deriveEnvelopeAESKey(privateKey, publicKey) {
  const sharedBits = await requireCrypto().subtle.deriveBits(
    { name: AGREEMENT_ALGORITHM.name, public: publicKey },
    privateKey,
    256,
  );
  return requireCrypto().subtle.importKey(
    "raw",
    sharedBits,
    { name: "AES-GCM" },
    false,
    ["encrypt", "decrypt"],
  );
}

async function generatePrekeyPair(prekeyID) {
  const pair = await requireCrypto().subtle.generateKey(AGREEMENT_ALGORITHM, true, ["deriveBits"]);
  return {
    prekey_id: prekeyID,
    public_key: await exportKeyBase64("spki", pair.publicKey),
    private_key: await exportKeyBase64("pkcs8", pair.privateKey),
  };
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

export async function generateDeviceIdentityBundle(options = {}) {
  const algorithm = options.algorithm?.trim() || DEFAULT_DEVICE_ALGORITHM;
  const prekeyCount = Number.isInteger(options.prekeyCount) && options.prekeyCount > 0 ? options.prekeyCount : 5;
  const subtle = requireCrypto().subtle;

  const identityPair = await subtle.generateKey(SIGNING_ALGORITHM, true, ["sign", "verify"]);
  const signedPrekeyPair = await subtle.generateKey(AGREEMENT_ALGORITHM, true, ["deriveBits"]);
  const signedPrekeyID = randomPositiveInt();
  const signedPrekeyPublic = await exportKeyBase64("spki", signedPrekeyPair.publicKey);
  const signedPrekeySignature = await subtle.sign(
    SIGNATURE_ALGORITHM,
    identityPair.privateKey,
    decodeBase64ToBytes(signedPrekeyPublic),
  );

  const prekeys = await Promise.all(
    Array.from({ length: prekeyCount }, (_, index) => generatePrekeyPair(signedPrekeyID + index + 1)),
  );

  return {
    algorithm,
    identity_key: await exportKeyBase64("spki", identityPair.publicKey),
    signed_prekey_id: signedPrekeyID,
    signed_prekey: signedPrekeyPublic,
    signed_prekey_signature: encodeBytesToBase64(new Uint8Array(signedPrekeySignature)),
    prekeys: prekeys.map(({ prekey_id, public_key }) => ({ prekey_id, public_key })),
    private_bundle: {
      version: 1,
      algorithm: ENVELOPE_SCHEME,
      identity_private_key: await exportKeyBase64("pkcs8", identityPair.privateKey),
      signed_prekey_private_key: await exportKeyBase64("pkcs8", signedPrekeyPair.privateKey),
      prekeys: prekeys.map(({ prekey_id, private_key }) => ({ prekey_id, private_key })),
    },
  };
}

export function formatPrekeysForTextarea(prekeys = []) {
  return prekeys.map((prekey) => `${prekey.prekey_id}:${prekey.public_key}`).join("\n");
}

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
    ciphertext: encodeBytesToBase64(encodeText(JSON.stringify({
      kind: "opaque-envelope-scaffold",
      body,
      nonce: encodeBytesToBase64(randomBytes(12)),
    }))),
    encryption_version: encryptionVersion,
    sender_device_id: senderDeviceID,
    recipient_device_id: recipientDeviceID,
  };
}

export async function buildEncryptedEnvelope({
  body,
  senderDevice,
  recipientDevice,
  senderPrivateBundle,
  encryptionVersion = DEFAULT_ENVELOPE_VERSION,
}) {
  if (!body) {
    throw new Error("body is required");
  }

  const senderDeviceID = Number(senderDevice?.id);
  const recipientDeviceID = Number(recipientDevice?.id);
  if (!Number.isInteger(senderDeviceID) || senderDeviceID <= 0) {
    throw new Error("sender device id is required");
  }
  if (!Number.isInteger(recipientDeviceID) || recipientDeviceID <= 0) {
    throw new Error("recipient device id is required");
  }
  if (!recipientDevice?.signed_prekey) {
    throw new Error("recipient signed prekey is required");
  }
  if (!senderPrivateBundle?.identity_private_key) {
    throw new Error("sender private bundle is required");
  }

  const subtle = requireCrypto().subtle;
  const ephemeralPair = await subtle.generateKey(AGREEMENT_ALGORITHM, true, ["deriveBits"]);
  const recipientPublicKey = await importAgreementPublicKey(recipientDevice.signed_prekey);
  const contentKey = await deriveEnvelopeAESKey(ephemeralPair.privateKey, recipientPublicKey);
  const iv = randomBytes(12);
  const encrypted = await subtle.encrypt(
    { name: "AES-GCM", iv },
    contentKey,
    encodeText(body),
  );

  const payload = {
    scheme: ENVELOPE_SCHEME,
    sender_device_id: senderDeviceID,
    recipient_device_id: recipientDeviceID,
    ephemeral_key: await exportKeyBase64("spki", ephemeralPair.publicKey),
    iv: encodeBytesToBase64(iv),
    ciphertext: encodeBytesToBase64(new Uint8Array(encrypted)),
  };
  const signature = await signEnvelopePayload(payload, senderPrivateBundle.identity_private_key);

  return {
    ciphertext: encodeBytesToBase64(encodeText(JSON.stringify({
      ...payload,
      signature,
    }))),
    encryption_version: encryptionVersion,
    sender_device_id: senderDeviceID,
    recipient_device_id: recipientDeviceID,
  };
}

export async function decryptEncryptedEnvelope({
  ciphertext,
  recipientPrivateBundle,
  senderIdentityKey,
}) {
  if (!ciphertext) {
    throw new Error("ciphertext is required");
  }
  if (!recipientPrivateBundle?.signed_prekey_private_key) {
    throw new Error("recipient private bundle is required");
  }

  const envelope = JSON.parse(decodeText(decodeBase64ToBytes(ciphertext)));
  const { signature = "", ...payload } = envelope;
  const recipientPrivateKey = await importAgreementPrivateKey(recipientPrivateBundle.signed_prekey_private_key);
  const ephemeralPublicKey = await importAgreementPublicKey(payload.ephemeral_key);
  const contentKey = await deriveEnvelopeAESKey(recipientPrivateKey, ephemeralPublicKey);
  const decrypted = await requireCrypto().subtle.decrypt(
    { name: "AES-GCM", iv: decodeBase64ToBytes(payload.iv) },
    contentKey,
    decodeBase64ToBytes(payload.ciphertext),
  );

  let signatureValid = false;
  if (senderIdentityKey) {
    signatureValid = await verifyEnvelopePayload(payload, signature, senderIdentityKey);
  }

  return {
    body: decodeText(new Uint8Array(decrypted)),
    envelope: payload,
    signatureValid,
  };
}
