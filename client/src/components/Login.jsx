import React, { useState } from 'react';
import { loginUser } from '../api';

export default function Login({ onLogin, onShowRegister }) {
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');

    const handleSubmit = async (e) => {
      e.preventDefault();
      try {
        const data = await loginUser(username, password);
        if (data && data.token) {
          onLogin(data.token);
        } else {
          setMessage("Login failed: No token received.");
        }
      } catch (error) {
        setMessage(error.message);
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
