import React, { useState } from 'react';
import { registerUser } from '../api';

function Register({ onRegisterSuccess }) {
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');

    const handleSubmit = async (e) => {
        e.preventDefault();
        setError('');
        try {
            const result = await registerUser(username, password);
            if (result.id) {
                alert('Registration successful! Please log in.');
                onRegisterSuccess();
            } else {
                throw new Error(result.message || 'Registration failed');
            }
        } catch (err) {
            setError(err.message);
        }
    };

    return (
        <form onSubmit={handleSubmit}>
            <h2>Register</h2>
            <input type="text" value={username} onChange={e => setUsername(e.target.value)} placeholder="Username" required />
            <input type="password" value={password} onChange={e => setPassword(e.target.value)} placeholder="Password" required />
            <button type="submit">Register</button>
            {error && <p style={{color: 'red'}}>{error}</p>}
        </form>
    );
}

export default Register;
