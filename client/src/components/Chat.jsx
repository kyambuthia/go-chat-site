import React, { useState, useEffect, useRef } from 'react';

function Chat({ token, onLogout }) {
    const [messages, setMessages] = useState([]);
    const [recipient, setRecipient] = useState('');
    const [body, setBody] = useState('');
    const webSocket = useRef(null);
    const messagesEndRef = useRef(null);

    const scrollToBottom = () => {
        messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }

    useEffect(scrollToBottom, [messages]);

    useEffect(() => {
        const wsUrl = `ws://${window.location.host}/ws?token=${token}`;
        const ws = new WebSocket(wsUrl);

        ws.onopen = () => setMessages(prev => [...prev, { type: 'system', body: 'Connected to chat'}]);
        ws.onclose = () => setMessages(prev => [...prev, { type: 'system', body: 'Disconnected from chat'}]);
        
        ws.onmessage = (event) => {
            const msg = JSON.parse(event.data);
            if (msg.type === 'error') {
                setMessages(prev => [...prev, { type: 'system', body: `Error: ${msg.body}` }]);
            } else if (msg.type === 'ack') {
                // Optional: handle acknowledgements
                console.log('Message delivered:', msg.body);
            } else {
                setMessages(prev => [...prev, { type: 'received', body: msg.body }]);
            }
        };
        
        ws.onerror = (error) => {
            console.error('WebSocket Error:', error);
            setMessages(prev => [...prev, { type: 'system', body: 'WebSocket connection error.'}]);
        }

        webSocket.current = ws;

        return () => {
            ws.close();
        };
    }, [token]);

    const handleSubmit = (e) => {
        e.preventDefault();
        if (webSocket.current && recipient && body) {
            const msg = { type: 'direct_message', to: recipient, body };
            webSocket.current.send(JSON.stringify(msg));
            setMessages(prev => [...prev, { type: 'sent', body: `To ${recipient}: ${body}` }]);
            setBody('');
        }
    };

    return (
        <div className="chat-container">
            <div style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center'}}>
                <h2>Chat</h2>
                <button onClick={onLogout}>Logout</button>
            </div>
            <div className="messages">
                {messages.map((msg, i) => (
                    <div key={i} className={`message ${msg.type}`}>
                        {msg.body}
                    </div>
                ))}
                <div ref={messagesEndRef} />
            </div>
            <form onSubmit={handleSubmit}>
                <input type="text" value={recipient} onChange={e => setRecipient(e.target.value)} placeholder="Recipient" required />
                <input type="text" value={body} onChange={e => setBody(e.target.value)} placeholder="Message" required style={{gridColumn: 'span 2'}}/>
                <button type="submit">Send</button>
            </form>
        </div>
    );
}

export default Chat;
