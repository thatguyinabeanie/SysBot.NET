import type { BotDto } from '../api/types';
import { useBotAction, useRemoveBot } from '../hooks/useBots';
import { StatusLamp } from './StatusLamp';

export function BotCard({ bot }: { bot: BotDto }) {
  const action = useBotAction();
  const remove = useRemoveBot();

  const send = (a: 'start' | 'stop' | 'pause' | 'resume' | 'restart') =>
    action.mutate({ id: bot.id, action: a });

  const statusText = !bot.isRunning
    ? 'Stopped'
    : bot.isPaused
      ? 'Paused'
      : bot.isConnected
        ? 'Running'
        : 'Connecting';

  const statusStyle =
    statusText === 'Running' ? 'bg-ok-subtle text-ok border-ok/20' :
    statusText === 'Paused' ? 'bg-warn-subtle text-warn border-warn/20' :
    statusText === 'Connecting' ? 'bg-info-subtle text-info border-info/20' :
    'bg-surface-2 text-txt-muted border-border-subtle';

  return (
    <div className="group flex items-center gap-4 p-4 rounded-xl bg-surface-1 border border-border-subtle hover:border-border transition-all duration-200 hover:shadow-lg hover:shadow-black/5 dark:hover:shadow-black/20">
      <StatusLamp bot={bot} />

      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2.5">
          <span className="font-mono text-sm font-medium text-txt">
            {bot.protocol === 'USB' ? `USB:${bot.port}` : bot.ip}
          </span>
          <span className="text-[11px] font-medium px-2 py-0.5 rounded-full bg-accent-subtle text-accent border border-accent-border">
            {bot.initialRoutine}
          </span>
          <span className={`text-[11px] font-medium px-2 py-0.5 rounded-full border ${statusStyle}`}>
            {statusText}
          </span>
        </div>
        {bot.lastLog && (
          <p className="text-xs text-txt-muted truncate mt-1 font-mono">{bot.lastLog}</p>
        )}
      </div>

      <div className="flex gap-1.5 opacity-60 group-hover:opacity-100 transition-opacity">
        {!bot.isRunning && (
          <ActionBtn onClick={() => send('start')} label="Start" variant="primary" />
        )}
        {bot.isRunning && !bot.isPaused && (
          <>
            <ActionBtn onClick={() => send('pause')} label="Pause" variant="muted" />
            <ActionBtn onClick={() => send('stop')} label="Stop" variant="danger" />
          </>
        )}
        {bot.isPaused && (
          <ActionBtn onClick={() => send('resume')} label="Resume" variant="primary" />
        )}
        <ActionBtn onClick={() => send('restart')} label="Restart" variant="muted" />
        <ActionBtn
          onClick={() => { if (confirm(`Remove bot ${bot.id}?`)) remove.mutate(bot.id); }}
          label="Remove"
          variant="danger"
        />
      </div>
    </div>
  );
}

const variantStyles: Record<string, string> = {
  primary: 'bg-accent hover:bg-accent-hover text-white',
  danger: 'bg-danger-subtle hover:bg-danger/20 text-danger border border-danger/20',
  muted: 'bg-surface-3/60 hover:bg-surface-3 text-txt-secondary',
};

function ActionBtn({ onClick, label, variant }: { onClick: () => void; label: string; variant: string }) {
  return (
    <button
      onClick={onClick}
      className={`text-xs font-medium px-2.5 py-1 rounded-lg ${variantStyles[variant] ?? variantStyles.muted} transition-colors cursor-pointer`}
    >
      {label}
    </button>
  );
}
