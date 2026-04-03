import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { configApi } from '../api/client';

export function useConfigSchema() {
  return useQuery({
    queryKey: ['config-schema'],
    queryFn: configApi.getSchema,
  });
}

export function useHubConfig() {
  return useQuery({
    queryKey: ['config-hub'],
    queryFn: configApi.getHub,
  });
}

export function usePatchHubConfig() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: configApi.patchHub,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['config-hub'] });
      qc.invalidateQueries({ queryKey: ['config-schema'] });
    },
  });
}
