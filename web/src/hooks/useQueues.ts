import { useQuery } from '@tanstack/react-query';
import { statusApi } from '../api/client';

export function useQueues() {
  return useQuery({
    queryKey: ['queues'],
    queryFn: statusApi.queues,
    refetchInterval: 5000,
  });
}

export function useMeta() {
  return useQuery({
    queryKey: ['meta'],
    queryFn: statusApi.meta,
  });
}
