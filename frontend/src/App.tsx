import { useState, type FormEvent } from "react";
import { BrowserRouter, Navigate, Route, Routes, useNavigate } from "react-router-dom";
import { Map } from "./components/Map";

type Mode = "login" | "signup";
const api = import.meta.env.VITE_API_URL ?? "http://localhost:8000";

function Auth({ mode }: { mode: Mode }) {
  const navigate = useNavigate();
  const [error, setError] = useState("");
  const isSignup = mode === "signup";

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");

    const form = new FormData(event.currentTarget);
    if (isSignup && form.get("password") !== form.get("confirmPassword")) {
      setError("Passwords do not match");
      return;
    }

    try {
      const response = await fetch(`${api}/${mode}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          username: form.get("username"),
          password: form.get("password"),
        }),
      });

      if (!response.ok) {
        const data = await response.json();
        setError(data.message);
        return;
      }
    } catch {
      setError("Could not reach backend");
      return;
    }

    navigate("/map");
  }

  return (
    <main className="auth">
      <form onSubmit={submit}>
        <h1>IsoMap</h1>
        <div className="tabs">
          <button type="button" onClick={() => navigate("/login")}>
            Log in
          </button>
          <button type="button" onClick={() => navigate("/signup")}>
            Sign up
          </button>
        </div>
        {error && <p className="error">{error}</p>}
        <input name="username" placeholder="Username" required />
        <input name="password" placeholder="Password" type="password" required />
        {isSignup && (
          <input
            name="confirmPassword"
            placeholder="Confirm password"
            type="password"
            required
          />
        )}
        <button>{isSignup ? "Sign up" : "Log in"}</button>
      </form>
    </main>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Navigate to="/login" replace />} />
        <Route path="/login" element={<Auth mode="login" />} />
        <Route path="/signup" element={<Auth mode="signup" />} />
        <Route path="/map" element={<Map />} />
        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    </BrowserRouter>
  );
}
