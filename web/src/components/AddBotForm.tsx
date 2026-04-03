import { useState } from 'react';
import { useAddBot } from '../hooks/useBots';
import { useMeta } from '../hooks/useQueues';

const inputClass = 'bg-surface-0 border border-border rounded-lg px-3 py-1.5 text-sm text-txt focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent/50 transition-all placeholder:text-txt-faint';

export function AddBotForm() {
  const { data: meta } = useMeta();
  const addBot = useAddBot();

  const [ip, setIp] = useState('192.168.0.1');
  const [port, setPort] = useState(6000);
  const [protocol, setProtocol] = useState('WiFi');
  const [routine, setRoutine] = useState('FlexTrade');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    addBot.mutate(
      { ip, port, protocol, routine },
      { onSuccess: () => setIp('192.168.0.1') },
    );
  };

  return (
    <form
      onSubmit={handleSubmit}
      className="flex items-end gap-3 flex-wrap p-4 rounded-xl bg-surface-1/60 border border-border-subtle"
    >
      {protocol === 'WiFi' && (
        <Field label="IP Address">
          <input type="text" value={ip} onChange={(e) => setIp(e.target.value)}
            className={`${inputClass} w-40`} placeholder="192.168.0.1" />
        </Field>
      )}

      <Field label="Port">
        <input type="number" value={port} onChange={(e) => setPort(Number(e.target.value))}
          readOnly={protocol === 'WiFi'}
          className={`${inputClass} w-22 read-only:opacity-50`} />
      </Field>

      <Field label="Protocol">
        <select value={protocol}
          onChange={(e) => { setProtocol(e.target.value); if (e.target.value === 'WiFi') setPort(6000); }}
          className={inputClass}>
          {(meta?.protocols ?? ['WiFi', 'USB']).map((p) => (
            <option key={p} value={p}>{p}</option>
          ))}
        </select>
      </Field>

      <Field label="Routine">
        <select value={routine} onChange={(e) => setRoutine(e.target.value)} className={inputClass}>
          {(meta?.supportedRoutines ?? ['FlexTrade']).map((r) => (
            <option key={r} value={r}>{r}</option>
          ))}
        </select>
      </Field>

      <button type="submit" disabled={addBot.isPending}
        className="bg-accent hover:bg-accent-hover text-white text-sm font-medium px-4 py-1.5 rounded-lg transition-colors cursor-pointer disabled:opacity-50 shadow-sm">
        + Add Bot
      </button>

      {addBot.isError && (
        <span className="text-danger text-xs">{String(addBot.error)}</span>
      )}
    </form>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="flex flex-col gap-1">
      <span className="text-[11px] font-medium text-txt-muted uppercase tracking-wider">{label}</span>
      {children}
    </label>
  );
}
