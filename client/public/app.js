import { registerUser, loginUser } from '../src/api.js';

const registerForm = document.getElementById('registerForm');
const loginForm = document.getElementById('loginForm');
const chatForm = document.getElementById('chatForm');
const loginStatus = document.getElementById('loginStatus');
const messagesDiv = document.getElementById('messages');

let ws;

registerForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    const username = document.getElementById('regUser').value;
    const password = document.getElementById('regPass').value;
    try {
        const result = await registerUser(username, password);
        alert(`Registration successful for ${result.username}!`);
    } catch (err) {
        alert('Registration failed.');
    }
});

loginForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    const username = document.getElementById('loginUser').value;
    const password = document.getElementById('loginPass').value;
    try {
        const result = await loginUser(username, password);
        if (!result.token) {
            throw new Error('Login failed');
        }
        localStorage.setItem('token', result.token);
        loginStatus.textContent = `Logged in as ${username}`;
        connectChat(result.token);
    } catch (err) {
        alert('Login failed.');
    }
});

chatForm.addEventListener('submit', (e) => {
    e.preventDefault();
    if (!ws) {
        alert('You must be logged in to chat.');
        return;
    }
    const to = document.getElementById('chatTo').value;
    const body = document.getElementById('chatMsg').value;
    ws.send(JSON.stringify({ type: 'direct_message', to, body }));
    document.getElementById('chatMsg').value = ''; // Clear input
    addMessage(`You to ${to}: ${body}`);
});

function connectChat(token) {
    // The WebSocket connection requires the raw token in the protocol header,
    // which is not possible in browser JS. We will adjust the server to accept it as a query param.
    const wsUrl = `ws://${window.location.host}/ws?token=${token}`;
    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
        console.log('WebSocket connected');
    };

    ws.onmessage = (event) => {
        const msg = JSON.parse(event.data);
        addMessage(`Received: ${msg.body}`)
    };

    ws.onclose = () => {
        console.log('WebSocket disconnected');
        loginStatus.textContent = 'Disconnected. Please log in again.';
        ws = null;
    };

    ws.onerror = (err) => {
        console.error('WebSocket error:', err);
        alert('WebSocket connection failed.');
    };
}

function addMessage(text) {
    const p = document.createElement('p');
    p.textContent = text;
    messagesDiv.appendChild(p);
    messagesDiv.scrollTop = messagesDiv.scrollHeight;
}
