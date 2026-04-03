import { useQueues } from '../hooks/useQueues';

export function QueueStatus() {
  const { data } = useQueues();
  if (!data) return null;

  return (
    <div className="flex items-center gap-2 text-xs">
      <span className={`px-2 py-0.5 rounded-full font-medium border ${
        data.canQueue
          ? 'bg-ok-subtle text-ok border-ok/20'
          : 'bg-danger-subtle text-danger border-danger/20'
      }`}>
        {data.canQueue ? 'Queues Open' : 'Queues Closed'}
      </span>
      {Object.entries(data.queues).map(([name, q]) => (
        <span key={name} className="px-2 py-0.5 bg-surface-2 text-txt-muted rounded-full">
          {name} <span className="text-txt-secondary font-medium">{q.count}</span>
        </span>
      ))}
    </div>
  );
}
