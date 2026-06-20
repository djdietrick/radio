import { ReactNode } from "react";
import { Navigate, useLocation } from "react-router-dom";
import { useAuth } from "../store/auth";

// RequireAuth gates routes behind a valid session, bouncing to /login while
// preserving the attempted location so we can return after sign-in.
export function RequireAuth({ children }: { children: ReactNode }) {
  const authed = useAuth((s) => s.authed);
  const location = useLocation();
  if (!authed) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }
  return <>{children}</>;
}
