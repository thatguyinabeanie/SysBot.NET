import { useBots } from '../hooks/useBots';
import { BotCard } from './BotCard';

export function BotList() {
  const { data: bots, isLoading, error } = useBots();

  if (isLoading) return <p className="text-txt-muted animate-pulse">Loading bots...</p>;
  if (error) return <p className="text-danger">Failed to load bots: {String(error)}</p>;
  if (!bots?.length) return <p className="text-txt-faint text-center py-8">No bots configured.</p>;

  return (
    <div className="flex flex-col gap-2">
      {bots.map((bot) => (
        <BotCard key={bot.id} bot={bot} />
      ))}
    </div>
  );
}
