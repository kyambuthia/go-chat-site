import assert from "node:assert/strict";
import test from "node:test";

import {
  DEFAULT_DEVICE_ALGORITHM,
  DEFAULT_ENVELOPE_VERSION,
  buildEncryptedEnvelope,
  buildOpaqueEnvelopeScaffold,
  decryptEncryptedEnvelope,
  formatPrekeysForTextarea,
  generateDeviceIdentityBundle,
  selectPreferredActiveDevice,
  selectPreferredRecipientDevice,
} from "./deviceIdentity.js";

test("generateDeviceIdentityBundle creates public and private device material", async () => {
  const bundle = await generateDeviceIdentityBundle({ prekeyCount: 3 });

  assert.equal(bundle.algorithm, DEFAULT_DEVICE_ALGORITHM);
  assert.equal(typeof bundle.identity_key, "string");
  assert.equal(typeof bundle.signed_prekey, "string");
  assert.equal(typeof bundle.signed_prekey_signature, "string");
  assert.equal(Number.isInteger(bundle.signed_prekey_id), true);
  assert.equal(bundle.prekeys.length, 3);
  assert.ok(bundle.prekeys.every((prekey) => Number.isInteger(prekey.prekey_id) && typeof prekey.public_key === "string"));
  assert.equal(typeof bundle.private_bundle.identity_private_key, "string");
  assert.equal(typeof bundle.private_bundle.signed_prekey_private_key, "string");
  assert.equal(bundle.private_bundle.prekeys.length, 3);
});

test("formatPrekeysForTextarea serializes one prekey per line", () => {
  const text = formatPrekeysForTextarea([
    { prekey_id: 1, public_key: "alpha" },
    { prekey_id: 2, public_key: "beta" },
  ]);

  assert.equal(text, "1:alpha\n2:beta");
});

test("selectPreferredActiveDevice prefers the current active device", () => {
  const device = selectPreferredActiveDevice([
    { id: 1, state: "active", current_session: false },
    { id: 2, state: "active", current_session: true },
  ]);

  assert.equal(device.id, 2);
});

test("selectPreferredRecipientDevice returns the first active directory device", () => {
  const device = selectPreferredRecipientDevice({
    devices: [
      { id: 10, state: "revoked" },
      { id: 11, state: "active" },
    ],
  });

  assert.equal(device.id, 11);
});

test("buildOpaqueEnvelopeScaffold returns optional envelope metadata", () => {
  const envelope = buildOpaqueEnvelopeScaffold({
    body: "hello",
    senderDeviceID: 7,
    recipientDeviceID: 9,
  });

  assert.equal(typeof envelope.ciphertext, "string");
  assert.equal(envelope.encryption_version, DEFAULT_ENVELOPE_VERSION);
  assert.equal(envelope.sender_device_id, 7);
  assert.equal(envelope.recipient_device_id, 9);
});

test("buildEncryptedEnvelope encrypts a message body that the recipient can decrypt", async () => {
  const senderBundle = await generateDeviceIdentityBundle({ prekeyCount: 1 });
  const recipientBundle = await generateDeviceIdentityBundle({ prekeyCount: 1 });

  const envelope = await buildEncryptedEnvelope({
    body: "hello from alice",
    senderDevice: { id: 7, identity_key: senderBundle.identity_key },
    recipientDevice: { id: 9, signed_prekey: recipientBundle.signed_prekey },
    senderPrivateBundle: senderBundle.private_bundle,
  });

  assert.equal(typeof envelope.ciphertext, "string");
  assert.equal(envelope.encryption_version, DEFAULT_ENVELOPE_VERSION);
  assert.equal(envelope.sender_device_id, 7);
  assert.equal(envelope.recipient_device_id, 9);

  const decrypted = await decryptEncryptedEnvelope({
    ciphertext: envelope.ciphertext,
    recipientPrivateBundle: recipientBundle.private_bundle,
    senderIdentityKey: senderBundle.identity_key,
  });

  assert.equal(decrypted.body, "hello from alice");
  assert.equal(decrypted.signatureValid, true);
  assert.equal(decrypted.envelope.sender_device_id, 7);
  assert.equal(decrypted.envelope.recipient_device_id, 9);
});
