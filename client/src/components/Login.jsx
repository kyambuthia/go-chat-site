import React, { useState } from 'react';
import { loginUser } from '../api';

export default function Login({ onLogin, onShowRegister }) {
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');

    const handleSubmit = async (e) => {
        e.preventDefault();
        setError('');
        try {
            const result = await loginUser(username, password);
            if (result.token) {
      onLogin(data.token);
            } else {
                throw new Error(result.message || 'Login failed');
            }
        } catch (err) {
            setError(err.message);
        }
    };

    return (
        <form onSubmit={handleSubmit}>
            <h2>Login</h2>
            <input type="text" value={username} onChange={e => setUsername(e.target.value)} placeholder="Username" required />
            <input type="password" value={password} onChange={e => setPassword(e.target.value)} placeholder="Password" required />
            <button type="submit">Login</button>
            {error && <p style={{color: 'red'}}>{error}</p>}
        </form>
    );
  }
