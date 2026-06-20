import { formatDuration } from "../format";
import { usePlayer } from "../store/player";

// QueuePanel shows the upcoming library queue with the current track highlighted.
// Clicking a row jumps to it; the × removes it. Hidden in radio mode (a station's
// queue is server-resolved and not user-editable).
export function QueuePanel({ onClose }: { onClose: () => void }) {
  const mode = usePlayer((s) => s.mode);
  const queue = usePlayer((s) => s.queue);
  const index = usePlayer((s) => s.index);
  const jumpTo = usePlayer((s) => s.jumpTo);
  const removeFromQueue = usePlayer((s) => s.removeFromQueue);

  return (
    <aside className="queue-panel">
      <div className="queue-panel__head">
        <h3>Queue</h3>
        <button className="btn btn--ghost" onClick={onClose}>
          ✕
        </button>
      </div>
      {mode === "radio" ? (
        <p className="muted">A station is playing. Its queue is set by the broadcast.</p>
      ) : queue.length === 0 ? (
        <p className="muted">Queue is empty.</p>
      ) : (
        <ol className="queue-panel__list">
          {queue.map((t, i) => (
            <li
              key={`${t.id}-${i}`}
              className={`queue-panel__row${i === index ? " is-active" : ""}`}
            >
              <button className="queue-panel__title" onClick={() => jumpTo(i)}>
                <span className="queue-panel__name">{t.title}</span>
                <span className="queue-panel__artist">{t.artistName}</span>
              </button>
              <span className="muted">{formatDuration(t.durationMs)}</span>
              <button
                className="btn btn--ghost"
                title="Remove"
                onClick={() => removeFromQueue(i)}
              >
                ✕
              </button>
            </li>
          ))}
        </ol>
      )}
    </aside>
  );
}
