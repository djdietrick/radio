// Small formatting helpers shared across screens.

// formatDuration renders milliseconds as m:ss or h:mm:ss.
export function formatDuration(ms: number): string {
  if (!ms || ms < 0) return "0:00";
  const totalSec = Math.floor(ms / 1000);
  const h = Math.floor(totalSec / 3600);
  const m = Math.floor((totalSec % 3600) / 60);
  const s = totalSec % 60;
  const ss = String(s).padStart(2, "0");
  if (h > 0) return `${h}:${String(m).padStart(2, "0")}:${ss}`;
  return `${m}:${ss}`;
}

// formatCount pluralizes a label, e.g. formatCount(1, "track") -> "1 track".
export function formatCount(n: number, label: string): string {
  return `${n} ${label}${n === 1 ? "" : "s"}`;
}
