import { useState, useEffect, useMemo, useRef } from "react";
import { getContacts, getInbox, getMessageThreads, getOutbox, markMessageDelivered, markMessageRead, sendMoney, syncMessages } from "../api";
import {
  CheckIcon,
  PaperPlaneIcon,
  PersonIcon,
  MagnifyingGlassIcon,
  PlusIcon,
  Cross2Icon,
} from "@radix-ui/react-icons";
import { Avatar, AvatarFallback, AvatarImage } from "@radix-ui/react-avatar";

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

function getPaymentRequestMeta(message) {
  if (message?.paymentRequestId && Number.isFinite(Number(message.paymentAmount))) {
    return {
      requestId: String(message.paymentRequestId),
      amount: Number(message.paymentAmount),
      status: message.paymentStatus || "pending",
      error: message.paymentError || "",
    };
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

function buildThreadHistory(contact, inboxMessages, outboxMessages) {
  const received = inboxMessages
    .filter((message) => Number(message.from_user_id) === Number(contact.id))
    .map((message) => addPaymentRequestMetadata({
      id: message.id,
      serverID: message.id,
      from: contact.username,
      body: message.body,
      sent: false,
      delivered: Boolean(message.delivered_at),
      read: Boolean(message.read_at),
      createdAt: message.created_at,
    }));

  const sent = outboxMessages
    .filter((message) => Number(message.to_user_id) === Number(contact.id))
    .map((message) => addPaymentRequestMetadata({
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
      failed: !message.delivered_at,
      errorMessage: "",
    }));

  return sortMessages(dedupeMessages([...received, ...sent]));
}

function normalizeStoredMessageForContact(message, contact) {
  const received = Number(message.from_user_id) === Number(contact.id);

  return addPaymentRequestMetadata({
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
    failed: !received && !message.delivered_at,
    errorMessage: "",
  });
}

function buildThreadSummariesByUsername(summaries) {
  return Object.fromEntries((summaries || []).map((summary) => [summary.username, summary]));
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

function summarizePreviewBody(body) {
  const payload = decodeMicroPayload(body);
  if (!payload) {
    return body || "No messages yet";
  }
  if (payload.kind === "payment_request" && Number.isFinite(Number(payload.amount))) {
    return `Requested ${formatMoney(payload.amount)}`;
  }
  if (payload.kind === "payment_request_update") {
    return payload.status === "paid" ? "Payment marked paid" : "Payment update";
  }
  return body || "No messages yet";
}

function formatThreadPreview(summary) {
  if (!summary?.last_message) {
    return "No messages yet";
  }
  const preview = summarizePreviewBody(summary.last_message.body);
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

  const handleSendMessage = () => {
    const trimmed = newMessage.trim();
    if (!trimmed || !selectedContact) {
      return;
    }

    if (!canUseSocket) {
      setComposerError("Connecting… try again in a moment.");
      return;
    }

    const message = {
      id: Date.now(),
      type: "direct_message",
      to: selectedContact.username,
      body: trimmed,
      createdAt: new Date().toISOString(),
    };

    ws.send(JSON.stringify(message));
    onSendMessage(message);
    setComposerError("");
    setNewMessage("");
  };

  const handleSendPaymentRequest = () => {
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

    const message = addPaymentRequestMetadata({
      id: Date.now() + 1,
      type: "direct_message",
      to: selectedContact.username,
      body: encodeMicroPayload(payload),
      createdAt: new Date().toISOString(),
      paymentStatus: "pending",
      paymentError: "",
    });

    ws.send(JSON.stringify(message));
    onSendMessage(message);
    setRequestAmount("");
    closeMicroApps();
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
              ) : (
                <div className="message-body">{msg.body}</div>
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
                      <span className="message-status-label">Sending…</span>
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
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const syncCursorRef = useRef(0);
  const appliedSyncTokenRef = useRef(0);

  const selectedUsername = selectedContact?.username;
  const messages = selectedUsername ? (threads[selectedUsername] || []) : [];

  useEffect(() => {
    const fetchContacts = async () => {
      try {
        setLoading(true);
        setError(null);
        const [contactsResponse, threadSummariesResponse] = await Promise.all([
          getContacts(),
          getMessageThreads({ limit: 200 }),
        ]);
        const nextContacts = contactsResponse || [];
        const nextThreadSummaries = threadSummariesResponse || [];

        setContacts(nextContacts);
        setThreadSummaries(buildThreadSummariesByUsername(nextThreadSummaries));
        setUnreadByUser(buildUnreadByUserFromThreadSummaries(nextThreadSummaries));
        syncCursorRef.current = getMaxThreadSummaryMessageID(nextThreadSummaries);
      } catch (err) {
        console.error("Failed to fetch contacts:", err);
        setError(err.message);
        setContacts([]);
        setThreadSummaries({});
        setUnreadByUser({});
      } finally {
        setLoading(false);
      }
    };
    fetchContacts();
  }, []);

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

            const normalized = normalizeStoredMessageForContact(message, contact);
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

        if (readMessageIDs.size > 0) {
          setThreads((prev) => {
            const next = { ...prev };
            Object.keys(next).forEach((username) => {
              next[username] = next[username].map((message) =>
                readMessageIDs.has(message.serverID) ? { ...message, delivered: true, read: true } : message
              );
            });
            return next;
          });

          await Promise.allSettled([...readMessageIDs].map((messageID) => markMessageRead(messageID)));
        }

        if (deliveredMessageIDs.size > 0) {
          setThreads((prev) => {
            const next = { ...prev };
            Object.keys(next).forEach((username) => {
              next[username] = next[username].map((message) =>
                deliveredMessageIDs.has(message.serverID) ? { ...message, delivered: true } : message
              );
            });
            return next;
          });

          await Promise.allSettled([...deliveredMessageIDs].map((messageID) => markMessageDelivered(messageID)));
        }

        syncCursorRef.current = nextAfterID;

        const threadSummariesResponse = await getMessageThreads({ limit: 200 });
        if (!cancelled) {
          const summaries = threadSummariesResponse || [];
          setThreadSummaries(buildThreadSummariesByUsername(summaries));
          setUnreadByUser(buildUnreadByUserFromThreadSummaries(summaries));
          syncCursorRef.current = Math.max(nextAfterID, getMaxThreadSummaryMessageID(summaries));
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

    const loadThreadHistory = async () => {
      try {
        const [inboxMessages, outboxMessages] = await Promise.all([
          getInbox({ withUserID: selectedContact.id, limit: 100 }),
          getOutbox({ limit: 200 }),
        ]);

        if (cancelled) {
          return;
        }

        const nextThread = buildThreadHistory(selectedContact, inboxMessages || [], outboxMessages || []);

        setThreads((prev) => ({
          ...prev,
          [selectedContact.username]: nextThread,
        }));

        const unreadServerIDs = nextThread
          .filter((message) => !message.sent && message.serverID && !message.read)
          .map((message) => message.serverID);

        if (unreadServerIDs.length > 0) {
          setThreads((prev) => ({
            ...prev,
            [selectedContact.username]: (prev[selectedContact.username] || []).map((message) =>
              unreadServerIDs.includes(message.serverID) ? { ...message, read: true, delivered: true } : message
            ),
          }));
          await Promise.allSettled(unreadServerIDs.map((messageID) => markMessageRead(messageID)));
          const threadSummariesResponse = await getMessageThreads({ limit: 200 });
          if (!cancelled) {
            const summaries = threadSummariesResponse || [];
            setThreadSummaries(buildThreadSummariesByUsername(summaries));
            setUnreadByUser(buildUnreadByUserFromThreadSummaries(summaries));
            syncCursorRef.current = Math.max(syncCursorRef.current, getMaxThreadSummaryMessageID(summaries));
          }
        }
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
  }, [selectedContact]);

  useEffect(() => {
    if (!lastWsMessage) {
      return;
    }

    if (lastWsMessage.type === "message_ack") {
      setThreads((prev) => {
        const next = { ...prev };
        Object.keys(next).forEach((username) => {
          next[username] = next[username].map((msg) =>
            msg.clientID === lastWsMessage.id || (msg.id === lastWsMessage.id && !msg.serverID)
              ? {
                ...msg,
                serverID: lastWsMessage.stored_message_id || msg.serverID,
                delivered: true,
                failed: false,
                errorMessage: "",
              }
              : msg
          );
        });
        return next;
      });

      if (selectedContact) {
        void (async () => {
          try {
            const [inboxMessages, outboxMessages] = await Promise.all([
              getInbox({ withUserID: selectedContact.id, limit: 100 }),
              getOutbox({ limit: 200 }),
            ]);
            setThreads((prev) => ({
              ...prev,
              [selectedContact.username]: buildThreadHistory(selectedContact, inboxMessages || [], outboxMessages || []),
            }));
            const summaries = await getMessageThreads({ limit: 200 });
            setThreadSummaries(buildThreadSummariesByUsername(summaries || []));
            setUnreadByUser(buildUnreadByUserFromThreadSummaries(summaries || []));
          } catch (err) {
            console.error("Failed to refresh thread after ack:", err);
          }
        })();
      }
      return;
    }

    if (lastWsMessage.type === "message_delivered" || lastWsMessage.type === "message_read") {
      setThreads((prev) => {
        const next = { ...prev };
        Object.keys(next).forEach((username) => {
          next[username] = next[username].map((msg) => {
            if (msg.serverID !== lastWsMessage.id) {
              return msg;
            }

            if (lastWsMessage.type === "message_read") {
              return { ...msg, delivered: true, read: true };
            }

            return { ...msg, delivered: true };
          });
        });
        return next;
      });
      void (async () => {
        try {
          const summaries = await getMessageThreads({ limit: 200 });
          setThreadSummaries(buildThreadSummariesByUsername(summaries || []));
          setUnreadByUser(buildUnreadByUserFromThreadSummaries(summaries || []));
        } catch (err) {
          console.error("Failed to refresh thread summaries after receipt:", err);
        }
      })();
      return;
    }

    if (lastWsMessage.type === "error") {
      setThreads((prev) => {
        const next = { ...prev };
        Object.keys(next).forEach((username) => {
          next[username] = next[username].map((msg) =>
            msg.clientID === lastWsMessage.id || (msg.id === lastWsMessage.id && !msg.serverID)
              ? {
                ...msg,
                serverID: lastWsMessage.stored_message_id || msg.serverID,
                failed: true,
                delivered: false,
                errorMessage: lastWsMessage.body || "Delivery failed.",
              }
              : msg
          );
        });
        return next;
      });

      if (selectedUsername && lastWsMessage.to === selectedUsername) {
        setError(null);
      }

      if (selectedContact && lastWsMessage.to === selectedContact.username && lastWsMessage.body?.startsWith("User is not online:")) {
        void (async () => {
          try {
            const [inboxMessages, outboxMessages] = await Promise.all([
              getInbox({ withUserID: selectedContact.id, limit: 100 }),
              getOutbox({ limit: 200 }),
            ]);
            setThreads((prev) => ({
              ...prev,
              [selectedContact.username]: buildThreadHistory(selectedContact, inboxMessages || [], outboxMessages || []),
            }));
            const summaries = await getMessageThreads({ limit: 200 });
            setThreadSummaries(buildThreadSummariesByUsername(summaries || []));
            setUnreadByUser(buildUnreadByUserFromThreadSummaries(summaries || []));
          } catch (err) {
            console.error("Failed to refresh thread after delivery error:", err);
          }
        })();
      }
      return;
    }

    if (lastWsMessage.type === "direct_message" && lastWsMessage.from) {
      const paymentUpdate = getPaymentUpdateMeta(lastWsMessage);
      if (paymentUpdate) {
        setThreads((prev) => {
          const next = { ...prev };
          const existing = next[lastWsMessage.from] || [];
          next[lastWsMessage.from] = updateThreadPaymentRequest(existing, paymentUpdate.requestId, {
            paymentStatus: paymentUpdate.status,
            paymentError: "",
          });
          return next;
        });
        return;
      }

      const normalized = addPaymentRequestMetadata({
        ...lastWsMessage,
        id: lastWsMessage.id || Date.now(),
        serverID: lastWsMessage.id || null,
        createdAt: lastWsMessage.created_at || new Date().toISOString(),
        sent: false,
        delivered: false,
        read: false,
      });

      setThreads((prev) => {
        const existing = prev[lastWsMessage.from] || [];
        return {
          ...prev,
          [lastWsMessage.from]: sortMessages(dedupeMessages([...existing, normalized])),
        };
      });
      setThreadSummaries((prev) => {
        const existing = prev[lastWsMessage.from];
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
              body: normalized.body,
              created_at: normalized.createdAt,
            },
          },
        };
      });

      if (selectedUsername !== lastWsMessage.from) {
        setUnreadByUser((prev) => ({
          ...prev,
          [lastWsMessage.from]: (prev[lastWsMessage.from] || 0) + 1,
        }));

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
    }
  }, [lastWsMessage, selectedContact, selectedUsername]);

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
        ws.send(JSON.stringify({
          id: Date.now(),
          type: "direct_message",
          to: message.from,
          body: encodeMicroPayload({
            kind: "payment_request_update",
            requestId: paymentRequest.requestId,
            status: "paid",
          }),
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
    />
  );
}
