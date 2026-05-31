import { type FormEvent } from "react";
import { BrowserRouter, Navigate, Route, Routes, useNavigate } from "react-router-dom";
import { Map } from "./components/Map";

type Mode = "login" | "signup";

function Auth({ mode }: { mode: Mode }) {
  const navigate = useNavigate();
  const isSignup = mode === "signup";

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
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
        <input placeholder="Username" required />
        <input placeholder="Password" type="password" required />
        {isSignup && (
          <input placeholder="Confirm password" type="password" required />
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
