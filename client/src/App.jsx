import { useState, useEffect } from "react";
import Login from "./components/Login";
import Register from "./components/Register";
import Chat from "./components/Chat";
import { setToken } from "./api";

function App() {
  const [token, setTokenState] = useState(localStorage.getItem("token"));
  const [ws, setWs] = useState(null);
  const [isRegistering, setIsRegistering] = useState(false);

  useEffect(() => {
    if (token) {
      // Set token for API calls
      setToken(token);
      // Connect to WebSocket
      const connect = () => {
        const ws = new WebSocket(`ws://${window.location.host}/ws?token=${token}`);
        ws.onopen = () => {
          console.log("WebSocket connected");
          setWs(ws);
        };
        ws.onclose = () => {
          console.log("WebSocket disconnected. Retrying in 5 seconds...");
          setTimeout(connect, 5000);
        };
        ws.onerror = (err) => {
          console.error("WebSocket error:", err);
          ws.close();
        };
      };
      connect();
    } else {
      if (ws) {
        ws.close();
        setWs(null);
      }
    }

    // Cleanup on component unmount
    return () => {
      if (ws) {
        ws.close();
      }
    };
  }, [token]);

  const handleLoginSuccess = (data) => {
    localStorage.setItem("token", data.token);
    setTokenState(data.token);
  };

  const handleLogout = () => {
    localStorage.removeItem("token");
    setTokenState(null);
  };

  if (token && ws) {
    return (
      <div className="app-container">
        <button onClick={handleLogout} className="logout-button">Logout</button>
        <Chat ws={ws} />
      </div>
    );
  }

  return (
    <div className="auth-container">
      {isRegistering ? (
        <Register onRegisterSuccess={() => setIsRegistering(false)} />
      ) : (
        <Login onLoginSuccess={handleLoginSuccess} />
      )}
      <button onClick={() => setIsRegistering(!isRegistering)} className="auth-switch-button">
        {isRegistering ? "Back to Login" : "Create an account"}
      </button>
    </div>
  );
}

export default App;
