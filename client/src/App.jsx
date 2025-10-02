import React, { useState, useEffect } from 'react';
import Login from './components/Login';
import Register from './components/Register';
import Chat from './components/Chat';

function App() {
    const [token, setToken] = useState(null);
    const [isRegistering, setIsRegistering] = useState(false);

    useEffect(() => {
        const storedToken = localStorage.getItem('token');
        if (storedToken) {
            setToken(storedToken);
        }
    }, []);

    const handleLogout = () => {
        localStorage.removeItem('token');
        setToken(null);
    };

    if (token) {
        return <Chat token={token} onLogout={handleLogout} />;
    }

    return (
        <div className="app-container">
            {isRegistering ? (
                <Register onRegisterSuccess={() => setIsRegistering(false)} />
            ) : (
                <Login onLoginSuccess={setToken} />
            )}
            <div className="auth-switch-button">
                <button onClick={() => setIsRegistering(!isRegistering)}>
                    {isRegistering ? 'Switch to Login' : 'Switch to Register'}
                </button>
            </div>
        </div>
    );
}

export default App;
