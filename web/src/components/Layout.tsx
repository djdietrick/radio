import { useEffect, useState } from "react";
import { Link, NavLink, Outlet, useNavigate } from "react-router-dom";
import { api } from "../api";
import { useAuth } from "../store/auth";
import { PlayerBar } from "./PlayerBar";
import { QueuePanel } from "./QueuePanel";
import { RadioController } from "./RadioController";
import { SearchBox } from "./SearchBox";

// Layout is the authenticated app shell: a sidebar for navigation, a top bar
// with global search, the routed page content, a persistent player, and the
// always-mounted radio controller.
export function Layout() {
  const [queueOpen, setQueueOpen] = useState(false);
  const isAdmin = useAuth((s) => s.isAdmin);
  const username = useAuth((s) => s.username);
  const setUser = useAuth((s) => s.setUser);
  const logout = useAuth((s) => s.logout);
  const navigate = useNavigate();

  // Hydrate the current user (admin flag, username) once on mount.
  useEffect(() => {
    api
      .me()
      .then((u) => setUser(u.username, u.isAdmin))
      .catch(() => undefined);
  }, [setUser]);

  const doLogout = () => {
    logout();
    navigate("/login", { replace: true });
  };

  return (
    <div className="app">
      <nav className="sidebar">
        <Link to="/" className="sidebar__brand">
          Radio
        </Link>
        <NavLink to="/albums" className="sidebar__link">
          Albums
        </NavLink>
        <NavLink to="/artists" className="sidebar__link">
          Artists
        </NavLink>
        <NavLink to="/playlists" className="sidebar__link">
          Playlists
        </NavLink>
        <NavLink to="/stations" className="sidebar__link">
          Stations
        </NavLink>
        <NavLink to="/settings" className="sidebar__link">
          Settings
        </NavLink>
      </nav>

      <div className="main">
        <header className="topbar">
          <SearchBox />
          <div className="topbar__right">
            <button className="btn btn--ghost" onClick={() => setQueueOpen((o) => !o)}>
              Queue
            </button>
            <span className="topbar__user">
              {username}
              {isAdmin ? " (admin)" : ""}
            </span>
            <button className="btn btn--ghost" onClick={doLogout}>
              Log out
            </button>
          </div>
        </header>

        <main className="content">
          <Outlet />
        </main>
      </div>

      {queueOpen && <QueuePanel onClose={() => setQueueOpen(false)} />}

      <PlayerBar />
      <RadioController />
    </div>
  );
}
