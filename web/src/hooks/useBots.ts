import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';
import { botApi, getSignalRConnection } from '../api/client';
import type { BotDto } from '../api/types';

export function useBots() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: ['bots'],
    queryFn: () => botApi.list(),
    refetchInterval: 3000,
  });

  // Listen for real-time bot status changes via SignalR
  useEffect(() => {
    const conn = getSignalRConnection();

    const handler = (bot: BotDto) => {
      queryClient.setQueryData<BotDto[]>(['bots'], (old) => {
        if (!old) return [bot];
        const idx = old.findIndex((b) => b.id === bot.id);
        if (idx === -1) return [...old, bot];
        const next = [...old];
        next[idx] = bot;
        return next;
      });
    };

    conn.on('BotStatusChanged', handler);
    if (conn.state === 'Disconnected') {
      conn.start().catch(console.error);
    }

    return () => {
      conn.off('BotStatusChanged', handler);
    };
  }, [queryClient]);

  return query;
}

function useInvalidateBots() {
  const qc = useQueryClient();
  return () => qc.invalidateQueries({ queryKey: ['bots'] });
}

export function useAddBot() {
  const invalidate = useInvalidateBots();
  return useMutation({
    mutationFn: botApi.add,
    onSuccess: invalidate,
  });
}

export function useRemoveBot() {
  const invalidate = useInvalidateBots();
  return useMutation({
    mutationFn: botApi.remove,
    onSuccess: invalidate,
  });
}

export function useBotAction() {
  const invalidate = useInvalidateBots();
  return useMutation({
    mutationFn: (args: { id: string; action: 'start' | 'stop' | 'pause' | 'resume' | 'restart' }) =>
      botApi[args.action](args.id),
    onSuccess: invalidate,
  });
}

export function useStartAll() {
  const invalidate = useInvalidateBots();
  return useMutation({ mutationFn: botApi.startAll, onSuccess: invalidate });
}

export function useStopAll() {
  const invalidate = useInvalidateBots();
  return useMutation({ mutationFn: botApi.stopAll, onSuccess: invalidate });
}
