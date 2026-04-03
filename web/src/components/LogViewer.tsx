import { useRef, useEffect, useState } from 'react';
import type { LogEntry } from '../api/client';

export function LogViewer({ logs, onClear }: { logs: LogEntry[]; onClear: () => void }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);
  const [filter, setFilter] = useState('');

  useEffect(() => {
    if (autoScroll && containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [logs, autoScroll]);

  const handleScroll = () => {
    const el = containerRef.current;
    if (!el) return;
    setAutoScroll(el.scrollHeight - el.scrollTop - el.clientHeight < 40);
  };

  const filtered = filter
    ? logs.filter((l) =>
        l.identity.toLowerCase().includes(filter.toLowerCase()) ||
        l.message.toLowerCase().includes(filter.toLowerCase()))
    : logs;

  return (
    <div className="flex flex-col h-full gap-3">
      <div className="flex items-center gap-2">
        <div className="relative flex-1">
          <span className="absolute left-3 top-1/2 -translate-y-1/2 text-txt-faint text-sm">&#x1F50D;</span>
          <input
            type="text"
            placeholder="Filter by bot name or message..."
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="w-full bg-surface-1 border border-border rounded-lg pl-9 pr-3 py-2 text-sm text-txt focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent/50 transition-all placeholder:text-txt-faint"
          />
        </div>
        <button onClick={onClear}
          className="text-xs font-medium px-3 py-2 rounded-lg bg-surface-2/60 hover:bg-surface-3 text-txt-muted hover:text-txt-secondary cursor-pointer transition-colors">
          Clear
        </button>
        <div className={`text-xs px-2.5 py-1 rounded-full font-medium ${
          autoScroll ? 'bg-ok-subtle text-ok' : 'bg-surface-2 text-txt-muted'
        }`}>
          {autoScroll ? 'Live' : 'Paused'}
        </div>
      </div>

      <div
        ref={containerRef}
        onScroll={handleScroll}
        className="flex-1 overflow-y-auto font-mono text-xs bg-surface-0 rounded-xl p-4 border border-border-subtle leading-relaxed"
      >
        {filtered.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-txt-faint gap-2">
            <span className="text-2xl opacity-40">▤</span>
            <span className="text-sm">No log entries yet</span>
            <span className="text-xs text-txt-faint/60">Logs will appear here when bots are running</span>
          </div>
        ) : (
          filtered.map((log, i) => (
            <div key={i} className="flex gap-2 py-0.5 hover:bg-surface-2/40 rounded px-1 -mx-1">
              <span className="text-txt-faint shrink-0 select-none">{log.timestamp}</span>
              <span className="text-accent/80 shrink-0 font-medium">{log.identity}</span>
              <span className="text-txt-secondary">{log.message}</span>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
