import { useState, type FormEvent } from "react";
import { BrowserRouter, Navigate, Route, Routes, useNavigate } from "react-router-dom";
import { Map } from "./components/Map";

type Mode = "login" | "signup";

function Auth() {
  const [mode, setMode] = useState<Mode>("login");
  const navigate = useNavigate();

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    navigate("/map");
  }

  return (
    <main className="auth">
      <form onSubmit={submit}>
        <h1>IsoMap</h1>
        <div className="tabs">
          <button type="button" onClick={() => setMode("login")}>
            Log in
          </button>
          <button type="button" onClick={() => setMode("signup")}>
            Sign up
          </button>
        </div>
        <input placeholder="Username" required />
        <input placeholder="Password" type="password" required />
        <button>{mode === "login" ? "Log in" : "Sign up"}</button>
      </form>
    </main>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Auth />} />
        <Route path="/map" element={<Map />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
}
