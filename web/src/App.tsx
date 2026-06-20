import { Navigate, Route, Routes } from "react-router-dom";
import { Layout } from "./components/Layout";
import { RequireAuth } from "./components/RequireAuth";
import { AlbumPage } from "./pages/AlbumPage";
import { AlbumsPage } from "./pages/AlbumsPage";
import { ArtistPage } from "./pages/ArtistPage";
import { ArtistsPage } from "./pages/ArtistsPage";
import { LoginPage } from "./pages/LoginPage";
import { PlaylistPage } from "./pages/PlaylistPage";
import { PlaylistsPage } from "./pages/PlaylistsPage";
import { SearchPage } from "./pages/SearchPage";
import { SettingsPage } from "./pages/SettingsPage";
import { StationPage } from "./pages/StationPage";
import { StationsPage } from "./pages/StationsPage";

// Route map. Everything under the Layout requires auth; /login is standalone.
export function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        element={
          <RequireAuth>
            <Layout />
          </RequireAuth>
        }
      >
        <Route index element={<Navigate to="/albums" replace />} />
        <Route path="/albums" element={<AlbumsPage />} />
        <Route path="/albums/:albumId" element={<AlbumPage />} />
        <Route path="/artists" element={<ArtistsPage />} />
        <Route path="/artists/:artistId" element={<ArtistPage />} />
        <Route path="/search" element={<SearchPage />} />
        <Route path="/playlists" element={<PlaylistsPage />} />
        <Route path="/playlists/:playlistId" element={<PlaylistPage />} />
        <Route path="/stations" element={<StationsPage />} />
        <Route path="/stations/:stationId" element={<StationPage />} />
        <Route path="/settings" element={<SettingsPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
