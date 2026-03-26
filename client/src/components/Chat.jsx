import { useState, useEffect, useMemo, useRef } from "react";
import {
  getContacts,
  getDeviceDirectory,
  getDevices,
  getInbox,
  getMessageThreads,
  getOutbox,
  markMessageDelivered,
  markMessageRead,
  markThreadRead,
  sendMoney,
  syncMessages,
} from "../api";
import {
  CheckIcon,
  PaperPlaneIcon,
  PersonIcon,
  MagnifyingGlassIcon,
  PlusIcon,
  Cross2Icon,
} from "@radix-ui/react-icons";
import { Avatar, AvatarFallback, AvatarImage } from "@radix-ui/react-avatar";
import {
  buildEncryptedEnvelope,
  buildOpaqueEnvelopeScaffold,
  decryptEncryptedEnvelope,
  selectPreferredActiveDevice,
  selectPreferredRecipientDevice,
} from "../lib/deviceIdentity.js";
import { getLocalDeviceBundle } from "../lib/deviceKeyStore.js";

const MICROAPP_PREFIX = "__microapp_v1__:";

function formatTime(isoString) {
  return new Date(isoString).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function formatMoney(amount) {
  const numeric = Number(amount);
  if (!Number.isFinite(numeric)) {
    return "$0.00";
  }
  return `$${numeric.toFixed(2)}`;
}

function dayLabel(isoString) {
  const date = new Date(isoString);
  const today = new Date();
  const yesterday = new Date();
  yesterday.setDate(today.getDate() - 1);

  const dateKey = date.toDateString();
  if (dateKey === today.toDateString()) {
    return "Today";
  }
  if (dateKey === yesterday.toDateString()) {
    return "Yesterday";
  }
  return date.toLocaleDateString([], { month: "short", day: "numeric", year: "numeric" });
}

function groupMessagesByDay(messages) {
  const groups = [];
  let currentLabel = "";

  for (const msg of messages) {
    const createdAt = msg.createdAt || new Date().toISOString();
    const label = dayLabel(createdAt);
    if (label !== currentLabel) {
      groups.push({ type: "day", label });
      currentLabel = label;
    }
    groups.push({ type: "message", message: { ...msg, createdAt } });
  }

  return groups;
}

function encodeMicroPayload(payload) {
  return `${MICROAPP_PREFIX}${JSON.stringify(payload)}`;
}

function decodeMicroPayload(body) {
  if (typeof body !== "string" || !body.startsWith(MICROAPP_PREFIX)) {
    return null;
  }
  try {
    return JSON.parse(body.slice(MICROAPP_PREFIX.length));
  } catch (_err) {
    return null;
  }
}

function canUseMessageBodyForDisplay(message) {
  return !message?.ciphertext || message.sent || message.decrypted;
}

function getDisplayedMessageBody(message, fallback = "No messages yet") {
  if (!message) {
    return fallback;
  }
  if (canUseMessageBodyForDisplay(message)) {
    return message.body || fallback;
  }
  return "Encrypted message";
}

function getPaymentRequestMeta(message) {
  if (message?.paymentRequestId && Number.isFinite(Number(message.paymentAmount))) {
    return {
      requestId: String(message.paymentRequestId),
      amount: Number(message.paymentAmount),
      status: message.paymentStatus || "pending",
      error: message.paymentError || "",
    };
  }

  if (!canUseMessageBodyForDisplay(message)) {
    return null;
  }

  const payload = decodeMicroPayload(message?.body);
  if (!payload || payload.kind !== "payment_request") {
    return null;
  }

  const amount = Number(payload.amount);
  if (!payload.requestId || !Number.isFinite(amount) || amount <= 0) {
    return null;
  }

  return {
    requestId: String(payload.requestId),
    amount,
    status: payload.status || "pending",
    error: "",
  };
}

function getPaymentUpdateMeta(message) {
  if (!canUseMessageBodyForDisplay(message)) {
    return null;
  }
  const payload = decodeMicroPayload(message?.body);
  if (!payload || payload.kind !== "payment_request_update" || !payload.requestId) {
    return null;
  }
  return {
    requestId: String(payload.requestId),
    status: payload.status || "pending",
  };
}

function addPaymentRequestMetadata(message) {
  const paymentRequest = getPaymentRequestMeta(message);
  if (!paymentRequest) {
    return message;
  }
  return {
    ...message,
    paymentRequestId: paymentRequest.requestId,
    paymentAmount: paymentRequest.amount,
    paymentStatus: message.paymentStatus || paymentRequest.status || "pending",
    paymentError: message.paymentError || "",
  };
}

function updateThreadPaymentRequest(messages, requestId, patch) {
  return messages.map((msg) => {
    const meta = getPaymentRequestMeta(msg);
    if (!meta || meta.requestId !== requestId) {
      return msg;
    }
    return {
      ...msg,
      ...patch,
      paymentRequestId: meta.requestId,
      paymentAmount: meta.amount,
      paymentStatus: patch.paymentStatus || msg.paymentStatus || meta.status,
      paymentError: patch.paymentError ?? msg.paymentError ?? "",
    };
  });
}

function dedupeMessages(messages) {
  const seen = new Set();
  const deduped = [];

  for (const message of messages) {
    const identity = message.clientID
      ? `client:${message.clientID}:${message.to || message.from || ""}:${message.sent ? "sent" : "received"}`
      : message.serverID
        ? `server:${message.serverID}`
        : `${message.sent ? "sent" : "received"}:${message.from || ""}:${message.body || ""}:${message.createdAt || ""}`;

    if (seen.has(identity)) {
      continue;
    }

    seen.add(identity);
    deduped.push(message);
  }

  return deduped;
}

function sortMessages(messages) {
  return [...messages].sort((left, right) => {
    const leftTime = Date.parse(left.createdAt || "");
    const rightTime = Date.parse(right.createdAt || "");

    if (Number.isFinite(leftTime) && Number.isFinite(rightTime) && leftTime !== rightTime) {
      return leftTime - rightTime;
    }

    const leftID = Number(left.serverID || left.id || 0);
    const rightID = Number(right.serverID || right.id || 0);
    return leftID - rightID;
  });
}

function normalizeEnvelopeFields(source) {
  return {
    ciphertext: source?.ciphertext || "",
    encryptionVersion: source?.encryption_version || source?.encryptionVersion || "",
    senderDeviceID: Number(source?.sender_device_id || source?.senderDeviceID || 0),
    recipientDeviceID: Number(source?.recipient_device_id || source?.recipientDeviceID || 0),
  };
}

function withEncryptedBodyFallback(message) {
  if (!message?.ciphertext || message?.body) {
    return message;
  }
  return {
    ...message,
    body: "Encrypted message unavailable on this device.",
    encryptedUnavailable: true,
  };
}

function buildThreadHistory(contact, inboxMessages, outboxMessages) {
  const received = inboxMessages
    .filter((message) => Number(message.from_user_id) === Number(contact.id))
    .map((message) => addPaymentRequestMetadata(withEncryptedBodyFallback({
      id: message.id,
      serverID: message.id,
      from: contact.username,
      body: message.body,
      sent: false,
      delivered: Boolean(message.delivered_at),
      read: Boolean(message.read_at),
      createdAt: message.created_at,
      ...normalizeEnvelopeFields(message),
    })));

  const sent = outboxMessages
    .filter((message) => Number(message.to_user_id) === Number(contact.id))
    .map((message) => addPaymentRequestMetadata(withEncryptedBodyFallback({
      id: message.id,
      serverID: message.id,
      clientID: message.client_message_id || undefined,
      from: "Me",
      to: contact.username,
      body: message.body,
      sent: true,
      delivered: Boolean(message.delivered_at),
      read: Boolean(message.read_at),
      createdAt: message.created_at,
      failed: Boolean(message.delivery_failed) && !message.delivered_at && !message.read_at,
      errorMessage: message.delivery_failed && !message.delivered_at && !message.read_at ? "Recipient was offline." : "",
      ...normalizeEnvelopeFields(message),
    })));

  return sortMessages(dedupeMessages([...received, ...sent]));
}

function normalizeStoredMessageForContact(message, contact) {
  const received = Number(message.from_user_id) === Number(contact.id);

  return addPaymentRequestMetadata(withEncryptedBodyFallback({
    id: message.id,
    serverID: message.id,
    clientID: !received && message.client_message_id ? message.client_message_id : undefined,
    from: received ? contact.username : "Me",
    to: received ? undefined : contact.username,
    body: message.body,
    sent: !received,
    delivered: Boolean(message.delivered_at),
    read: Boolean(message.read_at),
    createdAt: message.created_at,
    failed: !received && Boolean(message.delivery_failed) && !message.delivered_at && !message.read_at,
    errorMessage: !received && message.delivery_failed && !message.delivered_at && !message.read_at ? "Recipient was offline." : "",
    ...normalizeEnvelopeFields(message),
  }));
}

function buildThreadSummariesByUsername(summaries) {
  return Object.fromEntries((summaries || []).map((summary) => [summary.username, summary]));
}

function findContactByUsername(contacts, username) {
  return (contacts || []).find((contact) => contact.username === username) || null;
}

function buildLocalThreadSummary(contact, message, unreadCount = 0) {
  if (!contact) {
    return null;
  }

  return {
    user_id: contact.id,
    username: contact.username,
    display_name: contact.display_name || "",
    avatar_url: contact.avatar_url || "",
    unread_count: unreadCount,
    last_message: {
      id: message.serverID || message.id,
      from_user_id: message.sent ? null : contact.id,
      to_user_id: message.sent ? contact.id : null,
      body: getDisplayedMessageBody(message),
      created_at: message.createdAt,
    },
  };
}

function messageAffectsThreadSummary(message) {
  return !getPaymentUpdateMeta(message);
}

function buildThreadSummaryLastMessage(contact, message, fallbackUserID = null) {
  const counterpartyUserID = contact?.id ?? fallbackUserID ?? null;
  return {
    id: message.serverID || message.id,
    from_user_id: message.sent ? null : counterpartyUserID,
    to_user_id: message.sent ? counterpartyUserID : null,
    body: getDisplayedMessageBody(message),
    created_at: message.createdAt,
  };
}

function findLatestVisibleThreadMessage(messages) {
  for (let index = messages.length - 1; index >= 0; index -= 1) {
    if (messageAffectsThreadSummary(messages[index])) {
      return messages[index];
    }
  }
  return null;
}

function upsertLocalThreadSummary(prevSummary, contact, messages, unreadCount) {
  const latestVisible = Array.isArray(messages)
    ? findLatestVisibleThreadMessage(messages)
    : (messageAffectsThreadSummary(messages) ? messages : null);

  if (!latestVisible) {
    return prevSummary ? { ...prevSummary, unread_count: unreadCount } : null;
  }

  const base = prevSummary || buildLocalThreadSummary(contact, latestVisible, unreadCount);
  if (!base) {
    return null;
  }

  return {
    ...base,
    user_id: contact?.id ?? base.user_id,
    username: contact?.username ?? base.username,
    display_name: contact?.display_name ?? base.display_name,
    avatar_url: contact?.avatar_url ?? base.avatar_url,
    unread_count: unreadCount,
    last_message: buildThreadSummaryLastMessage(contact, latestVisible, base.user_id),
  };
}

function buildUnreadCountForThread(messages) {
  return (messages || []).filter((message) => !message.sent && !message.read && messageAffectsThreadSummary(message)).length;
}

function buildUnreadByUserFromThreadSummaries(summaries) {
  const unreadByUser = {};
  for (const summary of summaries || []) {
    unreadByUser[summary.username] = Number(summary.unread_count || 0);
  }
  return unreadByUser;
}

function getMaxThreadSummaryMessageID(summaries) {
  return (summaries || []).reduce((maxID, summary) => {
    const messageID = Number(summary?.last_message?.id || 0);
    return messageID > maxID ? messageID : maxID;
  }, 0);
}

function summarizePreviewBody(message) {
  const displayedBody = getDisplayedMessageBody(message);
  const payload = decodeMicroPayload(displayedBody);
  if (!payload) {
    return displayedBody || "No messages yet";
  }
  if (payload.kind === "payment_request" && Number.isFinite(Number(payload.amount))) {
    return `Requested ${formatMoney(payload.amount)}`;
  }
  if (payload.kind === "payment_request_update") {
    return payload.status === "paid" ? "Payment marked paid" : "Payment update";
  }
  return displayedBody || "No messages yet";
}

function formatThreadPreview(summary) {
  if (!summary?.last_message) {
    return "No messages yet";
  }
  const preview = summarizePreviewBody(summary.last_message);
  if (Number(summary.last_message.from_user_id) === Number(summary.user_id)) {
    return preview;
  }
  return `You: ${preview}`;
}

function ChatWindow({
  ws,
  selectedContact,
  messages,
  onSendMessage,
  onBack,
  isOnline,
  onSettlePaymentRequest,
  buildEnvelopeFields,
}) {
  const [newMessage, setNewMessage] = useState("");
  const [composerError, setComposerError] = useState("");
  const [microAppsOpen, setMicroAppsOpen] = useState(false);
  const [requestPaymentOpen, setRequestPaymentOpen] = useState(false);
  const [requestAmount, setRequestAmount] = useState("");
  const [requestError, setRequestError] = useState("");
  const [expandedRequestId, setExpandedRequestId] = useState(null);
  const [settlingRequestId, setSettlingRequestId] = useState(null);

  const timeline = useMemo(() => groupMessagesByDay(messages), [messages]);
  const canUseSocket = !!ws && ws.readyState === WebSocket.OPEN;
  const canSendText = !!newMessage.trim() && !!selectedContact;

  const closeMicroApps = () => {
    setMicroAppsOpen(false);
    setRequestPaymentOpen(false);
    setRequestError("");
  };

  const handleSendMessage = async () => {
    const trimmed = newMessage.trim();
    if (!trimmed || !selectedContact) {
      return;
    }

    if (!canUseSocket) {
      setComposerError("Connecting… try again in a moment.");
      return;
    }

    try {
      const message = {
        id: Date.now(),
        type: "direct_message",
        to: selectedContact.username,
        body: trimmed,
        createdAt: new Date().toISOString(),
        ...(await buildEnvelopeFields(trimmed)),
      };

      ws.send(JSON.stringify(message));
      onSendMessage(message);
      setComposerError("");
      setNewMessage("");
    } catch (err) {
      setComposerError(err?.message || "Failed to prepare encrypted message.");
    }
  };

  const handleSendPaymentRequest = async () => {
    setRequestError("");

    if (!selectedContact) {
      return;
    }
    if (!canUseSocket) {
      setRequestError("Chat is reconnecting. Try again in a moment.");
      return;
    }

    const amount = Number.parseFloat(requestAmount);
    if (!Number.isFinite(amount) || amount <= 0) {
      setRequestError("Enter a valid amount.");
      return;
    }

    const requestId = `payreq_${Date.now()}_${Math.floor(Math.random() * 1000)}`;
    const payload = {
      kind: "payment_request",
      requestId,
      amount: Number(amount.toFixed(2)),
    };
    const body = encodeMicroPayload(payload);

    try {
      const message = addPaymentRequestMetadata({
        id: Date.now() + 1,
        type: "direct_message",
        to: selectedContact.username,
        body,
        createdAt: new Date().toISOString(),
        paymentStatus: "pending",
        paymentError: "",
        ...(await buildEnvelopeFields(body)),
      });

      ws.send(JSON.stringify(message));
      onSendMessage(message);
      setRequestAmount("");
      closeMicroApps();
    } catch (err) {
      setRequestError(err?.message || "Failed to prepare encrypted payment request.");
    }
  };

  const handleSettleRequestClick = async (msg) => {
    const paymentRequest = getPaymentRequestMeta(msg);
    if (!paymentRequest || msg.sent || !msg.from) {
      return;
    }

    setSettlingRequestId(paymentRequest.requestId);
    try {
      await onSettlePaymentRequest(msg);
      setExpandedRequestId(null);
    } finally {
      setSettlingRequestId(null);
    }
  };

  if (!selectedContact) {
    return <div className="chat-window placeholder">Select a contact to start chatting.</div>;
  }

  return (
    <div className="chat-window">
      <div className="chat-header compact">
        <button onClick={onBack} className="back-button" aria-label="Back to contacts">←</button>
        <Avatar className={`avatar-placeholder ${isOnline ? "online" : ""}`}>
          <AvatarImage src={selectedContact.avatar_url || ""} alt={selectedContact.display_name || selectedContact.username} />
          <AvatarFallback><PersonIcon width="20" height="20" /></AvatarFallback>
        </Avatar>
        <div className="chat-header-meta">
          <h2>{selectedContact.display_name || selectedContact.username}</h2>
          <p>{isOnline ? "Online" : "Offline"}</p>
        </div>
      </div>

      <div className="messages compact">
        {messages.length === 0 && <p className="thread-empty">No messages yet. Start the conversation.</p>}
        {timeline.map((entry, index) => {
          if (entry.type === "day") {
            return <div key={`day-${entry.label}-${index}`} className="day-separator">{entry.label}</div>;
          }

          const msg = entry.message;
          const paymentRequest = getPaymentRequestMeta(msg);
          const paymentUpdate = getPaymentUpdateMeta(msg);
          const isPaymentRequest = !!paymentRequest;
          const isReceivedRequest = isPaymentRequest && !msg.sent;
          const isPendingRequest = isPaymentRequest && paymentRequest.status === "pending";
          const isExpanded = isPaymentRequest && expandedRequestId === paymentRequest.requestId;
          const isSettling = isPaymentRequest && settlingRequestId === paymentRequest.requestId;

          return (
            <div key={`${msg.id || index}-${msg.createdAt}`} className={`message ${msg.sent ? "sent" : "received"}`}>
              {isPaymentRequest ? (
                <>
                  {isReceivedRequest ? (
                    <button
                      type="button"
                      className={`message-body payment-request-bubble ${isPendingRequest ? "is-clickable" : ""}`}
                      onClick={() => {
                        if (!isPendingRequest) {
                          return;
                        }
                        setExpandedRequestId(isExpanded ? null : paymentRequest.requestId);
                      }}
                      title={isPendingRequest ? "Open payment request" : "Payment request"}
                    >
                      {formatMoney(paymentRequest.amount)}
                    </button>
                  ) : (
                    <div className="message-body payment-request-bubble">
                      {formatMoney(paymentRequest.amount)}
                    </div>
                  )}

                  <div className="payment-request-row">
                    <span className={`payment-request-status ${paymentRequest.status}`}>
                      {paymentRequest.status === "paid"
                        ? "Paid"
                        : paymentRequest.status === "processing"
                          ? "Processing…"
                          : msg.sent
                            ? "Requested"
                            : "Tap to pay"}
                    </span>
                    {paymentRequest.error && <span className="payment-request-error">{paymentRequest.error}</span>}
                  </div>

                  {isExpanded && isReceivedRequest && isPendingRequest && (
                    <div className="payment-request-actions">
                      <button type="button" onClick={() => handleSettleRequestClick(msg)} disabled={isSettling}>
                        {isSettling ? "Processing…" : `Pay ${formatMoney(paymentRequest.amount)}`}
                      </button>
                      <button type="button" className="danger" onClick={() => setExpandedRequestId(null)} disabled={isSettling}>
                        Close
                      </button>
                    </div>
                  )}
                </>
              ) : paymentUpdate ? (
                <div className="message-body">
                  {paymentUpdate.status === "paid" ? "Payment marked paid" : "Payment update"}
                </div>
              ) : (
                <div className="message-body">{getDisplayedMessageBody(msg, "Encrypted message")}</div>
              )}

              <div className="message-meta">
                <span className="message-time">{formatTime(msg.createdAt)}</span>
                {msg.sent && (
                  <>
                    {msg.read ? (
                      <span className="message-status-label">Read</span>
                    ) : msg.delivered ? (
                      <span className="message-status-icon" title="Delivered"><CheckIcon /></span>
                    ) : msg.failed ? (
                      <span className="message-status-label is-error" title={msg.errorMessage || "Not delivered"}>Not delivered</span>
                    ) : (
                      <span className="message-status-label">{msg.serverID ? "Pending" : "Sending…"}</span>
                    )}
                  </>
                )}
              </div>
              {msg.sent && msg.errorMessage && <p className="message-inline-error">{msg.errorMessage}</p>}
            </div>
          );
        })}
      </div>

      <div className="composer-shell">
        {(microAppsOpen || requestPaymentOpen) && (
          <div className="microapp-panel">
            {!requestPaymentOpen ? (
              <div className="microapp-grid">
                <button
                  type="button"
                  className="microapp-tile"
                  onClick={() => {
                    setRequestPaymentOpen(true);
                    setRequestError("");
                  }}
                >
                  <span className="microapp-tile-title">Request Payment</span>
                  <span className="microapp-tile-subtitle">Send amount bubble</span>
                </button>
              </div>
            ) : (
              <div className="request-payment-panel">
                <div className="request-payment-header">
                  <p>Request payment from @{selectedContact.username}</p>
                  <button
                    type="button"
                    className="microapp-close"
                    onClick={() => {
                      setRequestPaymentOpen(false);
                      setRequestError("");
                    }}
                    aria-label="Close payment request panel"
                  >
                    <Cross2Icon />
                  </button>
                </div>
                <div className="request-payment-controls">
                  <input
                    type="number"
                    min="0.01"
                    step="0.01"
                    inputMode="decimal"
                    placeholder="0.00"
                    value={requestAmount}
                    onChange={(e) => setRequestAmount(e.target.value)}
                    aria-label="Request payment amount"
                  />
                  <button type="button" onClick={handleSendPaymentRequest}>Send Request</button>
                </div>
                {requestError && <p className="composer-inline-error">{requestError}</p>}
              </div>
            )}
          </div>
        )}

        <div className="input-area compact">
          <button
            type="button"
            className={`composer-action-btn ${microAppsOpen ? "open" : ""}`}
            onClick={() => {
              setMicroAppsOpen((prev) => {
                const next = !prev;
                if (!next) {
                  setRequestPaymentOpen(false);
                }
                return next;
              });
              setRequestError("");
            }}
            aria-label="Open micro-apps"
          >
            <PlusIcon />
          </button>

          <input
            type="text"
            value={newMessage}
            onChange={(e) => {
              setNewMessage(e.target.value);
              if (composerError) {
                setComposerError("");
              }
            }}
            placeholder={canUseSocket ? "Type a message..." : "Connecting…"}
            onKeyUp={(e) => e.key === "Enter" && handleSendMessage()}
            className="composer-text-input"
          />

          <button
            onClick={handleSendMessage}
            className="send-button"
            disabled={!canSendText}
            title={canUseSocket ? "Send message" : "Chat is reconnecting"}
          >
            <PaperPlaneIcon className="send-button-icon" />
          </button>
        </div>

        {composerError && <p className="composer-inline-error">{composerError}</p>}
      </div>
    </div>
  );
}

export default function Chat({ ws, selectedContact, setSelectedContact, onlineUsers, lastWsMessage, syncToken }) {
  const [threads, setThreads] = useState({});
  const [unreadByUser, setUnreadByUser] = useState({});
  const [threadSummaries, setThreadSummaries] = useState({});
  const [contacts, setContacts] = useState([]);
  const [currentDeviceIdentity, setCurrentDeviceIdentity] = useState(null);
  const [selectedContactDirectory, setSelectedContactDirectory] = useState(null);
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const syncCursorRef = useRef(0);
  const appliedSyncTokenRef = useRef(0);
  const contactsHydratedRef = useRef(false);
  const summaryBootstrapAttemptsRef = useRef(new Set());
  const hydratedThreadsRef = useRef(new Set());
  const selectedContactRef = useRef(selectedContact);
  const threadsRef = useRef(threads);
  const threadSummariesRef = useRef(threadSummaries);
  const unreadByUserRef = useRef(unreadByUser);
  const deviceDirectoryCacheRef = useRef({});
  const decryptMessageForContactRef = useRef(null);

  const selectedUsername = selectedContact?.username;
  const recipientDeviceIdentity = selectPreferredRecipientDevice(selectedContactDirectory);
  const messages = selectedUsername ? (threads[selectedUsername] || []) : [];

  useEffect(() => {
    selectedContactRef.current = selectedContact;
  }, [selectedContact]);

  useEffect(() => {
    threadsRef.current = threads;
  }, [threads]);

  useEffect(() => {
    threadSummariesRef.current = threadSummaries;
  }, [threadSummaries]);

  useEffect(() => {
    unreadByUserRef.current = unreadByUser;
  }, [unreadByUser]);

  const cacheDeviceDirectory = (username, directory) => {
    if (!username) {
      return;
    }
    deviceDirectoryCacheRef.current = {
      ...deviceDirectoryCacheRef.current,
      [username]: directory,
    };
  };

  const getOrFetchDeviceDirectory = async (username) => {
    if (!username) {
      return null;
    }

    const cached = deviceDirectoryCacheRef.current[username];
    if (cached) {
      return cached;
    }

    const directory = await getDeviceDirectory(username);
    cacheDeviceDirectory(username, directory || null);
    return directory || null;
  };

  const decryptMessageForContact = async (message, contact) => {
    if (!message?.ciphertext || message.sent || !contact?.username) {
      return message;
    }

    const recipientDeviceID = Number(message.recipientDeviceID || 0);
    if (!Number.isInteger(recipientDeviceID) || recipientDeviceID <= 0) {
      return message;
    }

    const recipientPrivateBundle = getLocalDeviceBundle(recipientDeviceID);
    if (!recipientPrivateBundle) {
      return withEncryptedBodyFallback(message);
    }

    try {
      const directory = await getOrFetchDeviceDirectory(contact.username);
      const senderDevice = (directory?.devices || []).find(
        (device) => Number(device.id) === Number(message.senderDeviceID || 0),
      );
      const decrypted = await decryptEncryptedEnvelope({
        ciphertext: message.ciphertext,
        recipientPrivateBundle,
        senderIdentityKey: senderDevice?.identity_key || "",
      });
      return addPaymentRequestMetadata({
        ...message,
        body: decrypted.body,
        decrypted: true,
        signatureVerified: decrypted.signatureValid,
        encryptedUnavailable: false,
      });
    } catch (err) {
      console.error(`Failed to decrypt message for ${contact.username}:`, err);
      return withEncryptedBodyFallback({
        ...message,
        encryptedUnavailable: true,
      });
    }
  };

  decryptMessageForContactRef.current = decryptMessageForContact;

  useEffect(() => {
    let cancelled = false;

    const fetchCurrentDeviceIdentity = async () => {
      try {
        const deviceResponse = await getDevices();
        if (!cancelled) {
          setCurrentDeviceIdentity(selectPreferredActiveDevice(deviceResponse || []));
        }
      } catch (err) {
        if (!cancelled) {
          setCurrentDeviceIdentity(null);
          console.error("Failed to fetch local device identities:", err);
        }
      }
    };

    void fetchCurrentDeviceIdentity();

    return () => {
      cancelled = true;
    };
  }, [syncToken]);

  useEffect(() => {
    if (!selectedUsername) {
      setSelectedContactDirectory(null);
      return;
    }

    let cancelled = false;

    const fetchSelectedContactDirectory = async () => {
      try {
        const directoryResponse = await getDeviceDirectory(selectedUsername);
        if (!cancelled) {
          cacheDeviceDirectory(selectedUsername, directoryResponse || null);
          setSelectedContactDirectory(directoryResponse || null);
        }
      } catch (err) {
        if (!cancelled) {
          setSelectedContactDirectory(null);
          console.error(`Failed to fetch device directory for ${selectedUsername}:`, err);
        }
      }
    };

    void fetchSelectedContactDirectory();

    return () => {
      cancelled = true;
    };
  }, [selectedUsername]);

  useEffect(() => {
    const fetchContacts = async () => {
      try {
        if (!contactsHydratedRef.current) {
          setLoading(true);
        }
        setError(null);
        const contactsResponse = await getContacts();
        const nextContacts = contactsResponse || [];

        setContacts(nextContacts);
        deviceDirectoryCacheRef.current = Object.fromEntries(
          Object.entries(deviceDirectoryCacheRef.current).filter(([username]) =>
            nextContacts.some((contact) => contact.username === username),
          ),
        );
        if (selectedContactRef.current) {
          const refreshedSelectedContact = nextContacts.find((contact) => contact.username === selectedContactRef.current.username);
          if (refreshedSelectedContact) {
            setSelectedContact(refreshedSelectedContact);
          } else {
            setSelectedContact(null);
          }
        }

        if (!contactsHydratedRef.current) {
          const threadSummariesResponse = await getMessageThreads({ limit: 200 });
          const nextThreadSummaries = threadSummariesResponse || [];

          setThreadSummaries(buildThreadSummariesByUsername(nextThreadSummaries));
          setUnreadByUser(buildUnreadByUserFromThreadSummaries(nextThreadSummaries));
          syncCursorRef.current = getMaxThreadSummaryMessageID(nextThreadSummaries);
        }
        contactsHydratedRef.current = true;
      } catch (err) {
        console.error("Failed to fetch contacts:", err);
        setError(err.message);
        if (!contactsHydratedRef.current) {
          setContacts([]);
          setThreadSummaries({});
          setUnreadByUser({});
        }
      } finally {
        setLoading(false);
      }
    };
    fetchContacts();
  }, [setSelectedContact, syncToken]);

  useEffect(() => {
    if (loading || contacts.length === 0) {
      return;
    }

    const candidateContacts = contacts.filter((contact) =>
      !threadSummaries[contact.username] && !summaryBootstrapAttemptsRef.current.has(contact.username),
    );
    if (candidateContacts.length === 0) {
      return;
    }

    candidateContacts.forEach((contact) => {
      summaryBootstrapAttemptsRef.current.add(contact.username);
    });

    let cancelled = false;

    const bootstrapMissingSummaries = async () => {
      const localSummaryUpdates = {};
      const localUnreadUpdates = {};
      const remoteCandidates = [];

      candidateContacts.forEach((contact) => {
        const existingThread = threads[contact.username] || [];
        if (existingThread.length > 0) {
          const unreadCount = buildUnreadCountForThread(existingThread);
          const nextSummary = upsertLocalThreadSummary(null, contact, existingThread, unreadCount);
          if (nextSummary) {
            localSummaryUpdates[contact.username] = nextSummary;
            localUnreadUpdates[contact.username] = unreadCount;
          }
        } else {
          remoteCandidates.push(contact);
        }
      });

      if (!cancelled && Object.keys(localSummaryUpdates).length > 0) {
        setThreadSummaries((prev) => ({ ...prev, ...localSummaryUpdates }));
        setUnreadByUser((prev) => ({ ...prev, ...localUnreadUpdates }));
      }

      if (remoteCandidates.length === 0) {
        return;
      }

      try {
        const outboxMessages = await getOutbox({ limit: 500 });
        const bootstrappedThreads = await Promise.all(remoteCandidates.map(async (contact) => {
            const inboxMessages = await getInbox({ withUserID: contact.id, limit: 100 });
          const thread = await Promise.all(
            buildThreadHistory(contact, inboxMessages || [], outboxMessages || []).map((message) =>
              decryptMessageForContactRef.current(message, contact),
            ),
          );
          return {
            contact,
            thread: sortMessages(dedupeMessages(thread)),
          };
        }));

        if (cancelled) {
          return;
        }

        const nextThreads = {};
        const nextSummaries = {};
        const nextUnreadByUser = {};

        bootstrappedThreads.forEach(({ contact, thread }) => {
          hydratedThreadsRef.current.add(contact.username);
          if (thread.length === 0) {
            return;
          }

          nextThreads[contact.username] = thread;
          const unreadCount = buildUnreadCountForThread(thread);
          const nextSummary = upsertLocalThreadSummary(null, contact, thread, unreadCount);
          if (!nextSummary) {
            return;
          }
          nextSummaries[contact.username] = nextSummary;
          nextUnreadByUser[contact.username] = unreadCount;
          syncCursorRef.current = Math.max(syncCursorRef.current, Number(nextSummary.last_message?.id || 0));
        });

        if (Object.keys(nextThreads).length > 0) {
          setThreads((prev) => ({ ...prev, ...nextThreads }));
        }
        if (Object.keys(nextSummaries).length > 0) {
          setThreadSummaries((prev) => ({ ...prev, ...nextSummaries }));
          setUnreadByUser((prev) => ({ ...prev, ...nextUnreadByUser }));
        }
      } catch (err) {
        if (!cancelled) {
          remoteCandidates.forEach((contact) => {
            summaryBootstrapAttemptsRef.current.delete(contact.username);
          });
          console.error("Failed to bootstrap missing thread summaries:", err);
        }
      }
    };

    void bootstrapMissingSummaries();

    return () => {
      cancelled = true;
    };
  }, [contacts, loading, threadSummaries, threads]);

  useEffect(() => {
    if (loading) {
      return;
    }

    const contactsByUsername = new Map(contacts.map((contact) => [contact.username, contact]));
    const currentSelected = selectedContactRef.current;
    if (currentSelected && !contactsByUsername.has(currentSelected.username)) {
      setSelectedContact(null);
    }

    summaryBootstrapAttemptsRef.current.forEach((username) => {
      if (!contactsByUsername.has(username)) {
        summaryBootstrapAttemptsRef.current.delete(username);
      }
    });
    hydratedThreadsRef.current.forEach((username) => {
      if (!contactsByUsername.has(username)) {
        hydratedThreadsRef.current.delete(username);
      }
    });

    setThreadSummaries((prev) => {
      let changed = false;
      const next = {};

      Object.entries(prev).forEach(([username, summary]) => {
        const contact = contactsByUsername.get(username);
        if (!contact) {
          changed = true;
          return;
        }

        const nextSummary = {
          ...summary,
          user_id: contact.id,
          username: contact.username,
          display_name: contact.display_name || "",
          avatar_url: contact.avatar_url || "",
        };
        if (
          nextSummary.user_id !== summary.user_id
          || nextSummary.username !== summary.username
          || nextSummary.display_name !== summary.display_name
          || nextSummary.avatar_url !== summary.avatar_url
        ) {
          changed = true;
        }
        next[username] = nextSummary;
      });

      return changed ? next : prev;
    });

    setUnreadByUser((prev) => {
      let changed = false;
      const next = {};

      Object.entries(prev).forEach(([username, count]) => {
        if (!contactsByUsername.has(username)) {
          changed = true;
          return;
        }
        next[username] = count;
      });

      return changed ? next : prev;
    });
  }, [contacts, loading, setSelectedContact]);

  useEffect(() => {
    if (!syncToken || loading || appliedSyncTokenRef.current === syncToken) {
      return;
    }

    appliedSyncTokenRef.current = syncToken;
    let cancelled = false;

    const runSync = async () => {
      try {
        const contactsByID = new Map(contacts.map((contact) => [Number(contact.id), contact]));
        let nextAfterID = syncCursorRef.current;
        let hasMore = true;
        const deliveredMessageIDs = new Set();
        const readMessageIDs = new Set();
        const normalizedByUsername = {};

        while (!cancelled && hasMore) {
          const response = await syncMessages({ afterID: nextAfterID, limit: 200 });
          const pageMessages = response?.messages || [];
          const cursorNext = Number(response?.cursor?.next_after_id ?? nextAfterID);

          for (const message of pageMessages) {
            const contact = contactsByID.get(Number(message.from_user_id)) || contactsByID.get(Number(message.to_user_id));
            if (!contact) {
              continue;
            }

            const normalized = await decryptMessageForContactRef.current(
              normalizeStoredMessageForContact(message, contact),
              contact,
            );
            if (!normalizedByUsername[contact.username]) {
              normalizedByUsername[contact.username] = [];
            }
            normalizedByUsername[contact.username].push(normalized);

            const isIncoming = Number(message.from_user_id) === Number(contact.id);
            if (!isIncoming || !message.id) {
              continue;
            }

            if (selectedContact && Number(selectedContact.id) === Number(contact.id)) {
              if (!message.read_at) {
                readMessageIDs.add(message.id);
              }
            } else if (!message.delivered_at) {
              deliveredMessageIDs.add(message.id);
            }
          }

          if (pageMessages.length === 0 || !Number.isFinite(cursorNext) || cursorNext <= nextAfterID) {
            hasMore = false;
          } else {
            nextAfterID = cursorNext;
            hasMore = Boolean(response?.has_more);
          }
        }

        if (cancelled) {
          return;
        }

        if (Object.keys(normalizedByUsername).length > 0) {
          setThreads((prev) => {
            const next = { ...prev };
            Object.entries(normalizedByUsername).forEach(([username, syncedMessagesForUser]) => {
              next[username] = sortMessages(dedupeMessages([...(next[username] || []), ...syncedMessagesForUser]));
            });
            return next;
          });
        }

        const syncedLatestVisibleByUsername = {};
        const syncedUnreadVisibleCountsByUsername = {};
        Object.entries(normalizedByUsername).forEach(([username, syncedMessagesForUser]) => {
          const latestVisible = findLatestVisibleThreadMessage(syncedMessagesForUser);
          if (latestVisible) {
            syncedLatestVisibleByUsername[username] = latestVisible;
          }
          syncedUnreadVisibleCountsByUsername[username] = buildUnreadCountForThread(syncedMessagesForUser);
        });

        if (readMessageIDs.size > 0) {
          setThreads((prev) => {
            const next = { ...prev };
            Object.keys(next).forEach((username) => {
              next[username] = next[username].map((message) =>
                readMessageIDs.has(message.serverID)
                  ? { ...message, delivered: true, read: true, failed: false, errorMessage: "" }
                  : message
              );
            });
            return next;
          });

          if (selectedContact) {
            await markThreadRead(selectedContact.id);
          }
        }

        if (deliveredMessageIDs.size > 0) {
          setThreads((prev) => {
            const next = { ...prev };
            Object.keys(next).forEach((username) => {
              next[username] = next[username].map((message) =>
                deliveredMessageIDs.has(message.serverID)
                  ? { ...message, delivered: true, failed: false, errorMessage: "" }
                  : message
              );
            });
            return next;
          });

          await Promise.allSettled([...deliveredMessageIDs].map((messageID) => markMessageDelivered(messageID)));
        }

        syncCursorRef.current = nextAfterID;

        if (Object.keys(syncedLatestVisibleByUsername).length > 0 || (selectedContact && readMessageIDs.size > 0)) {
          setThreadSummaries((prev) => {
            const next = { ...prev };
            Object.entries(syncedLatestVisibleByUsername).forEach(([username, latestVisible]) => {
              const contact = findContactByUsername(contacts, username);
              const unreadCount = selectedContact?.username === username && readMessageIDs.size > 0
                ? 0
                : Number(next[username]?.unread_count || 0) + Number(syncedUnreadVisibleCountsByUsername[username] || 0);
              const nextSummary = upsertLocalThreadSummary(next[username], contact, latestVisible, unreadCount);
              if (nextSummary) {
                next[username] = nextSummary;
              }
            });

            if (selectedContact && readMessageIDs.size > 0 && next[selectedContact.username]) {
              next[selectedContact.username] = {
                ...next[selectedContact.username],
                unread_count: 0,
              };
            }

            return next;
          });

          setUnreadByUser((prev) => {
            const next = { ...prev };
            Object.entries(syncedUnreadVisibleCountsByUsername).forEach(([username, count]) => {
              if (count > 0) {
                next[username] = Number(next[username] || 0) + count;
              }
            });
            if (selectedContact && readMessageIDs.size > 0) {
              next[selectedContact.username] = 0;
            }
            return next;
          });
        }
      } catch (err) {
        if (!cancelled) {
          console.error("Failed to sync messages:", err);
        }
      }
    };

    void runSync();

    return () => {
      cancelled = true;
    };
  }, [contacts, loading, selectedContact, syncToken]);

  useEffect(() => {
    if (!selectedContact) {
      return;
    }

    let cancelled = false;

    const applySelectedThreadReadState = async (thread) => {
      const unreadServerIDs = thread
        .filter((message) => !message.sent && message.serverID && !message.read)
        .map((message) => message.serverID);

      if (unreadServerIDs.length === 0) {
        return;
      }

      setThreads((prev) => ({
        ...prev,
        [selectedContact.username]: (prev[selectedContact.username] || []).map((message) =>
          unreadServerIDs.includes(message.serverID)
            ? { ...message, read: true, delivered: true, failed: false, errorMessage: "" }
            : message
        ),
      }));
      await markThreadRead(selectedContact.id);
      if (!cancelled) {
        setThreadSummaries((prev) => {
          const nextSummary = upsertLocalThreadSummary(prev[selectedContact.username], selectedContact, thread, 0);
          if (!nextSummary) {
            return prev;
          }
          return {
            ...prev,
            [selectedContact.username]: nextSummary,
          };
        });
        setUnreadByUser((prev) => ({ ...prev, [selectedContact.username]: 0 }));
      }
    };

    const loadThreadHistory = async () => {
      try {
        if (hydratedThreadsRef.current.has(selectedContact.username)) {
          const existingThread = threads[selectedContact.username] || [];
          await applySelectedThreadReadState(existingThread);
          return;
        }

        const [inboxMessages, outboxMessages] = await Promise.all([
          getInbox({ withUserID: selectedContact.id, limit: 100 }),
          getOutbox({ limit: 200 }),
        ]);

        if (cancelled) {
          return;
        }

        const nextThread = sortMessages(dedupeMessages(await Promise.all(
          buildThreadHistory(selectedContact, inboxMessages || [], outboxMessages || []).map((message) =>
            decryptMessageForContactRef.current(message, selectedContact),
          ),
        )));
        hydratedThreadsRef.current.add(selectedContact.username);

        setThreads((prev) => ({
          ...prev,
          [selectedContact.username]: nextThread,
        }));

        await applySelectedThreadReadState(nextThread);
      } catch (err) {
        if (!cancelled) {
          console.error("Failed to load thread history:", err);
        }
      }
    };

    loadThreadHistory();

    return () => {
      cancelled = true;
    };
  }, [selectedContact, threads]);

  useEffect(() => {
    if (!lastWsMessage) {
      return;
    }

    const reconcileLocalThreadCollections = (nextThreads, usernames) => {
      const uniqueUsernames = [...new Set((usernames || []).filter(Boolean))];
      if (uniqueUsernames.length === 0) {
        return;
      }

      const nextSummaries = { ...threadSummariesRef.current };
      const nextUnreadByUser = { ...unreadByUserRef.current };

      uniqueUsernames.forEach((username) => {
        const thread = nextThreads[username] || [];
        const contact = findContactByUsername(contacts, username);
        const unreadCount = buildUnreadCountForThread(thread);
        const nextSummary = upsertLocalThreadSummary(nextSummaries[username] || null, contact, thread, unreadCount);

        if (nextSummary) {
          nextSummaries[username] = nextSummary;
          nextUnreadByUser[username] = unreadCount;
        } else {
          delete nextSummaries[username];
          delete nextUnreadByUser[username];
        }
      });

      threadSummariesRef.current = nextSummaries;
      unreadByUserRef.current = nextUnreadByUser;
      setThreadSummaries(nextSummaries);
      setUnreadByUser(nextUnreadByUser);
    };

    if (lastWsMessage.type === "message_ack") {
      const nextThreads = { ...threadsRef.current };
      const affectedUsernames = [];

      Object.keys(nextThreads).forEach((username) => {
        let changed = false;
        const updatedThread = nextThreads[username].map((msg) => {
          if (msg.clientID !== lastWsMessage.id && !(msg.id === lastWsMessage.id && !msg.serverID)) {
            return msg;
          }
          changed = true;
          return {
            ...msg,
            serverID: lastWsMessage.stored_message_id || msg.serverID,
            delivered: true,
            failed: false,
            errorMessage: "",
          };
        });
        if (changed) {
          nextThreads[username] = updatedThread;
          affectedUsernames.push(username);
        }
      });

      if (affectedUsernames.length > 0) {
        threadsRef.current = nextThreads;
        setThreads(nextThreads);
        reconcileLocalThreadCollections(nextThreads, affectedUsernames);
      }
      return;
    }

    if (lastWsMessage.type === "message_delivered" || lastWsMessage.type === "message_read") {
      const nextThreads = { ...threadsRef.current };
      const affectedUsernames = [];

      Object.keys(nextThreads).forEach((username) => {
        let changed = false;
        const updatedThread = nextThreads[username].map((msg) => {
          if (msg.serverID !== lastWsMessage.id) {
            return msg;
          }

          changed = true;
          if (lastWsMessage.type === "message_read") {
            return { ...msg, delivered: true, read: true, failed: false, errorMessage: "" };
          }

          return { ...msg, delivered: true, failed: false, errorMessage: "" };
        });

        if (changed) {
          nextThreads[username] = updatedThread;
          affectedUsernames.push(username);
        }
      });

      if (affectedUsernames.length > 0) {
        threadsRef.current = nextThreads;
        setThreads(nextThreads);
        reconcileLocalThreadCollections(nextThreads, affectedUsernames);
      }
      return;
    }

    if (lastWsMessage.type === "error") {
      const nextThreads = { ...threadsRef.current };
      const affectedUsernames = [];

      Object.keys(nextThreads).forEach((username) => {
        let changed = false;
        const updatedThread = nextThreads[username].map((msg) => {
          if (msg.clientID !== lastWsMessage.id && !(msg.id === lastWsMessage.id && !msg.serverID)) {
            return msg;
          }

          changed = true;
          return {
            ...msg,
            serverID: lastWsMessage.stored_message_id || msg.serverID,
            failed: true,
            delivered: false,
            errorMessage: lastWsMessage.body || "Delivery failed.",
          };
        });

        if (changed) {
          nextThreads[username] = updatedThread;
          affectedUsernames.push(username);
        }
      });

      if (affectedUsernames.length > 0) {
        threadsRef.current = nextThreads;
        setThreads(nextThreads);
        reconcileLocalThreadCollections(nextThreads, affectedUsernames);
      }

      if (selectedUsername && lastWsMessage.to === selectedUsername) {
        setError(null);
      }
      return;
    }

    if (lastWsMessage.type === "direct_message" && lastWsMessage.from) {
      void (async () => {
        const paymentUpdate = getPaymentUpdateMeta(lastWsMessage);
        const shouldAffectSummary = !paymentUpdate;
        const messageContact = findContactByUsername(contacts, lastWsMessage.from);
        const normalized = await decryptMessageForContactRef.current(addPaymentRequestMetadata(withEncryptedBodyFallback({
          ...lastWsMessage,
          id: lastWsMessage.id || Date.now(),
          serverID: lastWsMessage.id || null,
          createdAt: lastWsMessage.created_at || new Date().toISOString(),
          sent: false,
          delivered: false,
          read: false,
          ...normalizeEnvelopeFields(lastWsMessage),
        })), messageContact);

        setThreads((prev) => {
          const existing = prev[lastWsMessage.from] || [];
          const merged = sortMessages(dedupeMessages([...existing, normalized]));
          return {
            ...prev,
            [lastWsMessage.from]: paymentUpdate
              ? updateThreadPaymentRequest(merged, paymentUpdate.requestId, {
                paymentStatus: paymentUpdate.status,
                paymentError: "",
              })
              : merged,
          };
        });
        if (shouldAffectSummary) {
          setThreadSummaries((prev) => {
            const existing = prev[lastWsMessage.from] || buildLocalThreadSummary(
              messageContact,
              normalized,
              selectedUsername !== lastWsMessage.from ? 1 : 0,
            );
            if (!existing) {
              return prev;
            }
            return {
              ...prev,
              [lastWsMessage.from]: {
                ...existing,
                unread_count: selectedUsername !== lastWsMessage.from
                  ? Number(existing.unread_count || 0) + 1
                  : 0,
                last_message: {
                  id: normalized.serverID || normalized.id,
                  from_user_id: existing.user_id,
                  to_user_id: null,
                  body: getDisplayedMessageBody(normalized),
                  created_at: normalized.createdAt,
                },
              },
            };
          });
        }

        if (selectedUsername !== lastWsMessage.from) {
          if (shouldAffectSummary) {
            setUnreadByUser((prev) => ({
              ...prev,
              [lastWsMessage.from]: (prev[lastWsMessage.from] || 0) + 1,
            }));
          }

          if (lastWsMessage.id) {
            void markMessageDelivered(lastWsMessage.id).catch((err) => {
              console.error("Failed to mark message delivered:", err);
            });
          }
        } else if (lastWsMessage.id) {
          void markMessageRead(lastWsMessage.id).catch((err) => {
            console.error("Failed to mark message read:", err);
          });
        }
      })();
      return;
    }
  }, [contacts, lastWsMessage, selectedContact, selectedUsername]);

  useEffect(() => {
    if (!selectedUsername) {
      return;
    }
    setUnreadByUser((prev) => ({ ...prev, [selectedUsername]: 0 }));
  }, [selectedUsername]);

  const filteredContacts = useMemo(() => {
    const needle = query.trim().toLowerCase();
    const base = contacts.filter((contact) => {
      if (!needle) {
        return true;
      }
      const display = (contact.display_name || "").toLowerCase();
      return contact.username.toLowerCase().includes(needle) || display.includes(needle);
    });

    return base.sort((a, b) => {
      const aUnread = unreadByUser[a.username] || 0;
      const bUnread = unreadByUser[b.username] || 0;
      if (aUnread !== bUnread) {
        return bUnread - aUnread;
      }
      const aLastCreatedAt = Date.parse(threadSummaries[a.username]?.last_message?.created_at || "");
      const bLastCreatedAt = Date.parse(threadSummaries[b.username]?.last_message?.created_at || "");
      if (Number.isFinite(aLastCreatedAt) && Number.isFinite(bLastCreatedAt) && aLastCreatedAt !== bLastCreatedAt) {
        return bLastCreatedAt - aLastCreatedAt;
      }
      const aOnline = onlineUsers.includes(a.username) ? 1 : 0;
      const bOnline = onlineUsers.includes(b.username) ? 1 : 0;
      if (aOnline !== bOnline) {
        return bOnline - aOnline;
      }
      return a.username.localeCompare(b.username);
    });
  }, [contacts, onlineUsers, query, threadSummaries, unreadByUser]);

  const handleSelectContact = (contact) => {
    setSelectedContact(contact);
  };

  const buildEnvelopeFields = async (body) => {
    const senderDeviceID = Number(currentDeviceIdentity?.id);
    const recipientDeviceID = Number(recipientDeviceIdentity?.id);

    if (!Number.isInteger(senderDeviceID) || senderDeviceID <= 0) {
      return {};
    }
    if (!Number.isInteger(recipientDeviceID) || recipientDeviceID <= 0) {
      return {};
    }

    const senderPrivateBundle = getLocalDeviceBundle(senderDeviceID);

    try {
      if (senderPrivateBundle) {
        return await buildEncryptedEnvelope({
          body,
          senderDevice: currentDeviceIdentity,
          recipientDevice: recipientDeviceIdentity,
          senderPrivateBundle,
        });
      }
      return buildOpaqueEnvelopeScaffold({
        body,
        senderDeviceID,
        recipientDeviceID,
      });
    } catch (err) {
      console.error("Failed to build envelope metadata:", err);
      return {};
    }
  };

  const handleSendMessage = (message) => {
    if (!selectedUsername) {
      return;
    }

    const normalized = addPaymentRequestMetadata({
      ...message,
      clientID: message.id,
      from: "Me",
      sent: true,
      delivered: false,
      read: false,
      failed: false,
      errorMessage: "",
      createdAt: message.createdAt || new Date().toISOString(),
    });

    setThreads((prev) => ({
      ...prev,
      [selectedUsername]: sortMessages([...(prev[selectedUsername] || []), normalized]),
    }));
    if (messageAffectsThreadSummary(normalized)) {
      setThreadSummaries((prev) => ({
        ...prev,
        [selectedUsername]: buildLocalThreadSummary(selectedContact, normalized, 0),
      }));
    }
    setUnreadByUser((prev) => ({ ...prev, [selectedUsername]: 0 }));
  };

  const handleSettlePaymentRequest = async (message) => {
    const paymentRequest = getPaymentRequestMeta(message);
    if (!paymentRequest || message.sent || !message.from) {
      return;
    }

    const requestThread = message.from;
    setThreads((prev) => {
      const next = { ...prev };
      next[requestThread] = updateThreadPaymentRequest(next[requestThread] || [], paymentRequest.requestId, {
        paymentStatus: "processing",
        paymentError: "",
      });
      return next;
    });

    try {
      await sendMoney(message.from, paymentRequest.amount);

      setThreads((prev) => {
        const next = { ...prev };
        next[requestThread] = updateThreadPaymentRequest(next[requestThread] || [], paymentRequest.requestId, {
          paymentStatus: "paid",
          paymentError: "",
        });
        return next;
      });

      if (ws && ws.readyState === WebSocket.OPEN) {
        const updateBody = encodeMicroPayload({
          kind: "payment_request_update",
          requestId: paymentRequest.requestId,
          status: "paid",
        });
        const envelopeFields = await buildEnvelopeFields(updateBody);
        ws.send(JSON.stringify({
          id: Date.now(),
          type: "direct_message",
          to: message.from,
          body: updateBody,
          ...envelopeFields,
        }));
      }
    } catch (err) {
      setThreads((prev) => {
        const next = { ...prev };
        next[requestThread] = updateThreadPaymentRequest(next[requestThread] || [], paymentRequest.requestId, {
          paymentStatus: "pending",
          paymentError: err?.message || "Payment failed.",
        });
        return next;
      });
      throw err;
    }
  };

  if (loading) {
    return (
      <div className="chat-main">
        <h2>Chat</h2>
        <p>Loading contacts...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="chat-main">
        <h2>Chat</h2>
        <p>Error: {error}</p>
      </div>
    );
  }

  if (!selectedContact) {
    return (
      <div className="chat-main">
        <h2>Chat</h2>
        {contacts.length === 0 ? (
          <p className="empty-chat-message">Invite a friend from the Contacts tab to start chatting.</p>
        ) : (
          <div className="contacts-list">
            <div className="contacts-toolbar">
              <div className="search-wrap">
                <MagnifyingGlassIcon />
                <input
                  type="text"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  placeholder="Search contacts"
                  aria-label="Search contacts"
                />
              </div>
            </div>
            <ul>
              {filteredContacts.map((contact) => {
                const unread = unreadByUser[contact.username] || 0;
                const summary = threadSummaries[contact.username];
                return (
                  <li key={contact.id} onClick={() => handleSelectContact(contact)}>
                    <Avatar className={`avatar-placeholder ${onlineUsers.includes(contact.username) ? "online" : ""}`}>
                      <AvatarImage src={contact.avatar_url || ""} alt={contact.display_name || contact.username} />
                      <AvatarFallback><PersonIcon width="24" height="24" /></AvatarFallback>
                    </Avatar>
                    <div className="contact-meta">
                      <span>{contact.display_name || contact.username}</span>
                      <small>{onlineUsers.includes(contact.username) ? "Online" : "Offline"}</small>
                      {summary && <small className="contact-preview">{formatThreadPreview(summary)}</small>}
                    </div>
                    {unread > 0 && <span className="unread-pill">{unread}</span>}
                  </li>
                );
              })}
            </ul>
          </div>
        )}
      </div>
    );
  }

  return (
    <ChatWindow
      ws={ws}
      selectedContact={selectedContact}
      messages={messages}
      onSendMessage={handleSendMessage}
      onBack={() => setSelectedContact(null)}
      isOnline={onlineUsers.includes(selectedContact.username)}
      onSettlePaymentRequest={handleSettlePaymentRequest}
      buildEnvelopeFields={buildEnvelopeFields}
    />
  );
}
