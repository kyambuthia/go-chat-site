import { useState, useEffect, useMemo } from "react";
import { getContacts, sendMoney } from "../api";
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
          <AvatarImage src="" alt="" />
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
                {msg.sent && msg.delivered && <span className="message-status-icon"><CheckIcon /></span>}
              </div>
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

export default function Chat({ ws, selectedContact, setSelectedContact, onlineUsers, lastWsMessage }) {
  const [threads, setThreads] = useState({});
  const [unreadByUser, setUnreadByUser] = useState({});
  const [contacts, setContacts] = useState([]);
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const selectedUsername = selectedContact?.username;
  const messages = selectedUsername ? (threads[selectedUsername] || []) : [];

  useEffect(() => {
    const fetchContacts = async () => {
      try {
        setLoading(true);
        const response = await getContacts();
        setContacts(response || []);
      } catch (err) {
        console.error("Failed to fetch contacts:", err);
        setError(err.message);
        setContacts([]);
      } finally {
        setLoading(false);
      }
    };
    fetchContacts();
  }, []);

  useEffect(() => {
    if (!lastWsMessage) {
      return;
    }

    if (lastWsMessage.type === "message_ack") {
      setThreads((prev) => {
        const next = { ...prev };
        Object.keys(next).forEach((username) => {
          next[username] = next[username].map((msg) =>
            msg.id === lastWsMessage.id ? { ...msg, delivered: true } : msg
          );
        });
        return next;
      });
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
        createdAt: new Date().toISOString(),
      });

      setThreads((prev) => ({
        ...prev,
        [lastWsMessage.from]: [...(prev[lastWsMessage.from] || []), normalized],
      }));

      if (selectedUsername !== lastWsMessage.from) {
        setUnreadByUser((prev) => ({
          ...prev,
          [lastWsMessage.from]: (prev[lastWsMessage.from] || 0) + 1,
        }));
      }
    }
  }, [lastWsMessage, selectedUsername]);

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
      const aOnline = onlineUsers.includes(a.username) ? 1 : 0;
      const bOnline = onlineUsers.includes(b.username) ? 1 : 0;
      if (aOnline !== bOnline) {
        return bOnline - aOnline;
      }
      return a.username.localeCompare(b.username);
    });
  }, [contacts, onlineUsers, query, unreadByUser]);

  const handleSelectContact = (contact) => {
    setSelectedContact(contact);
  };

  const handleSendMessage = (message) => {
    if (!selectedUsername) {
      return;
    }

    const normalized = addPaymentRequestMetadata({
      ...message,
      from: "Me",
      sent: true,
      delivered: false,
      createdAt: message.createdAt || new Date().toISOString(),
    });

    setThreads((prev) => ({
      ...prev,
      [selectedUsername]: [...(prev[selectedUsername] || []), normalized],
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
                return (
                  <li key={contact.id} onClick={() => handleSelectContact(contact)}>
                    <Avatar className={`avatar-placeholder ${onlineUsers.includes(contact.username) ? "online" : ""}`}>
                      <AvatarImage src="" alt="" />
                      <AvatarFallback><PersonIcon width="24" height="24" /></AvatarFallback>
                    </Avatar>
                    <div className="contact-meta">
                      <span>{contact.display_name || contact.username}</span>
                      <small>{onlineUsers.includes(contact.username) ? "Online" : "Offline"}</small>
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
