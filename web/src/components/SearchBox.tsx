import { useState } from "react";
import { useNavigate } from "react-router-dom";

// SearchBox routes to the search page on submit. Results live there (not in a
// dropdown) so they get full track/album/artist sections.
export function SearchBox() {
  const [q, setQ] = useState("");
  const navigate = useNavigate();

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    const term = q.trim();
    if (term) navigate(`/search?q=${encodeURIComponent(term)}`);
  };

  return (
    <form className="searchbox" onSubmit={submit}>
      <input
        type="search"
        placeholder="Search tracks, albums, artists…"
        value={q}
        onChange={(e) => setQ(e.target.value)}
      />
    </form>
  );
}
