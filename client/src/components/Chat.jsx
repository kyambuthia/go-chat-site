import React, { useState, useEffect, useRef } from 'react';

function Chat({ token, onLogout }) {
    const [messages, setMessages] = useState([]);
    const [recipient, setRecipient] = useState('');
    const [body, setBody] = useState('');
    const webSocket = useRef(null);

    useEffect(() => {
        const wsUrl = `ws://${window.location.host}/ws?token=${token}`;
        const ws = new WebSocket(wsUrl);

        ws.onopen = () => console.log('WebSocket Connected');
        ws.onclose = () => console.log('WebSocket Disconnected');
        ws.onmessage = (event) => {
            const msg = JSON.parse(event.data);
            setMessages(prev => [...prev, `Received: ${msg.body}`]);
        };
        ws.onerror = (error) => console.error('WebSocket Error:', error);

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
            setMessages(prev => [...prev, `You to ${recipient}: ${body}`]);
            setBody('');
        }
    };

    return (
        <div>
            <h2>Chat</h2>
            <button onClick={onLogout}>Logout</button>
            <div id="messages" style={{ height: '300px', overflowY: 'scroll', border: '1px solid #ccc', padding: '10px', margin: '10px 0' }}>
                {messages.map((msg, i) => <p key={i}>{msg}</p>)}
            </div>
            <form onSubmit={handleSubmit}>
                <input type="text" value={recipient} onChange={e => setRecipient(e.target.value)} placeholder="Recipient Username" required />
                <input type="text" value={body} onChange={e => setBody(e.target.value)} placeholder="Message" required />
                <button type="submit">Send</button>
            </form>
        </div>
    );
}

export default Chat;
