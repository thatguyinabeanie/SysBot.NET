import type { BotDto } from '../api/types';

function getLampStyle(bot: BotDto): { color: string; pulse: boolean; label: string } {
  if (!bot.isRunning) return { color: 'bg-txt-faint/60', pulse: false, label: 'Stopped' };
  if (!bot.isConnected) return { color: 'bg-info', pulse: true, label: 'Connecting' };
  if (bot.isPaused) return { color: 'bg-warn', pulse: false, label: 'Paused' };
  if (bot.currentRoutine === 'Idle' && bot.nextRoutine === 'Idle') {
    return { color: 'bg-warn', pulse: false, label: 'Idle' };
  }

  if (!bot.lastActive) return { color: 'bg-ok', pulse: true, label: 'Active' };
  const seconds = (Date.now() - new Date(bot.lastActive).getTime()) / 1000;
  if (seconds < 30) return { color: 'bg-ok', pulse: true, label: 'Active' };
  if (seconds < 60) return { color: 'bg-warn', pulse: false, label: 'Slow' };
  return { color: 'bg-danger', pulse: false, label: 'Stale' };
}

export function StatusLamp({ bot }: { bot: BotDto }) {
  const { color, pulse, label } = getLampStyle(bot);
  return (
    <div className="relative flex items-center justify-center" title={label}>
      {pulse && (
        <span className={`absolute w-3 h-3 rounded-full ${color} opacity-40 animate-ping`} />
      )}
      <span className={`relative w-3 h-3 rounded-full ${color} shadow-sm`} />
    </div>
  );
}
