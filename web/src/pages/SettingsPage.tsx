import { useState } from "react";
import { api } from "../api";
import { useCreateUser, useUsers } from "../hooks/queries";
import { useAuth } from "../store/auth";

export function SettingsPage() {
  const isAdmin = useAuth((s) => s.isAdmin);
  const [status, setStatus] = useState<string | null>(null);

  const run = async (label: string, fn: () => Promise<unknown>) => {
    setStatus(`${label}…`);
    try {
      await fn();
      setStatus(`${label} started.`);
    } catch (e) {
      setStatus(`${label} failed: ${String(e)}`);
    }
  };

  return (
    <div>
      <h1>Settings</h1>

      <section className="card">
        <h2>Library maintenance</h2>
        <div className="detail-head__actions">
          <button className="btn" onClick={() => run("Rescan", api.triggerScan)}>
            Rescan library
          </button>
          <button className="btn" onClick={() => run("Backfill", api.triggerBackfill)}>
            Backfill durations &amp; art
          </button>
        </div>
        {status && <p className="muted">{status}</p>}
      </section>

      {isAdmin && <UserAdmin />}
    </div>
  );
}

function UserAdmin() {
  const { data: users } = useUsers();
  const createUser = useCreateUser();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [admin, setAdmin] = useState(false);

  const create = (e: React.FormEvent) => {
    e.preventDefault();
    if (!username.trim() || !password) return;
    createUser.mutate(
      { username: username.trim(), password, isAdmin: admin },
      {
        onSuccess: () => {
          setUsername("");
          setPassword("");
          setAdmin(false);
        },
      },
    );
  };

  return (
    <section className="card">
      <h2>Users</h2>
      <ul className="row-list">
        {users?.map((u) => (
          <li key={u.id} className="row-list__item">
            <span className="row-list__main">
              {u.username}
              {u.isAdmin && <span className="badge">admin</span>}
            </span>
          </li>
        ))}
      </ul>

      <form className="station-form" onSubmit={create}>
        <h3>Add user</h3>
        <div className="field">
          <label>Username</label>
          <input value={username} onChange={(e) => setUsername(e.target.value)} />
        </div>
        <div className="field">
          <label>Password</label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </div>
        <label className="check">
          <input type="checkbox" checked={admin} onChange={(e) => setAdmin(e.target.checked)} />
          Administrator
        </label>
        <button className="btn btn--primary" type="submit" disabled={createUser.isPending}>
          Create user
        </button>
        {createUser.isError && <p className="error">{String(createUser.error)}</p>}
      </form>
    </section>
  );
}
