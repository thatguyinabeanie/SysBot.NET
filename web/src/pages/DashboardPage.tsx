import { AddBotForm } from '../components/AddBotForm';
import { BotList } from '../components/BotList';
import { QueueStatus } from '../components/QueueStatus';
import { useStartAll, useStopAll } from '../hooks/useBots';
import { useMeta } from '../hooks/useQueues';

export function DashboardPage() {
  const startAll = useStartAll();
  const stopAll = useStopAll();
  const { data: meta } = useMeta();

  return (
    <div className="flex flex-col gap-5 h-full max-w-5xl mx-auto">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div className="flex items-center gap-3">
          {meta && (
            <span className="text-xs font-medium px-2.5 py-1 rounded-full bg-accent-subtle text-accent border border-accent-border">
              {meta.mode}
            </span>
          )}
          <QueueStatus />
        </div>
        <div className="flex gap-2">
          <button onClick={() => startAll.mutate()} disabled={startAll.isPending}
            className="bg-ok hover:brightness-110 text-white text-xs font-medium px-4 py-1.5 rounded-lg transition-all cursor-pointer disabled:opacity-50 shadow-sm">
            Start All
          </button>
          <button onClick={() => stopAll.mutate()} disabled={stopAll.isPending}
            className="bg-surface-3/80 hover:bg-surface-3 text-txt-secondary text-xs font-medium px-4 py-1.5 rounded-lg transition-colors cursor-pointer disabled:opacity-50 border border-border-subtle">
            Stop All
          </button>
        </div>
      </div>

      <AddBotForm />
      <div className="border-t border-border-subtle" />
      <div className="flex-1 overflow-y-auto">
        <BotList />
      </div>
    </div>
  );
}
