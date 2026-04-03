import * as signalR from '@microsoft/signalr';
import type { BotDto, AddBotRequest, MetaInfo, QueueStatus, ConfigSchema } from './types';

// --- REST helpers ---

async function fetchJson<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...init?.headers },
  });
  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText);
    throw new Error(`${res.status}: ${text}`);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

// --- Bot API ---

export const botApi = {
  list: () => fetchJson<BotDto[]>('/api/bots'),
  add: (req: AddBotRequest) =>
    fetchJson<BotDto>('/api/bots', { method: 'POST', body: JSON.stringify(req) }),
  remove: (id: string) =>
    fetchJson<void>(`/api/bots/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  start: (id: string) =>
    fetchJson<BotDto>(`/api/bots/${encodeURIComponent(id)}/start`, { method: 'POST' }),
  stop: (id: string) =>
    fetchJson<BotDto>(`/api/bots/${encodeURIComponent(id)}/stop`, { method: 'POST' }),
  pause: (id: string) =>
    fetchJson<BotDto>(`/api/bots/${encodeURIComponent(id)}/pause`, { method: 'POST' }),
  resume: (id: string) =>
    fetchJson<BotDto>(`/api/bots/${encodeURIComponent(id)}/resume`, { method: 'POST' }),
  restart: (id: string) =>
    fetchJson<BotDto>(`/api/bots/${encodeURIComponent(id)}/restart`, { method: 'POST' }),
  startAll: () => fetchJson<{ count: number }>('/api/bots/start-all', { method: 'POST' }),
  stopAll: () => fetchJson<{ count: number }>('/api/bots/stop-all', { method: 'POST' }),
};

// --- Config API ---

export const configApi = {
  get: () => fetchJson<Record<string, unknown>>('/api/config'),
  getHub: () => fetchJson<Record<string, unknown>>('/api/config/hub'),
  patchHub: (patch: Record<string, unknown>) =>
    fetchJson<Record<string, unknown>>('/api/config/hub', {
      method: 'PATCH',
      body: JSON.stringify(patch),
    }),
  getSchema: () => fetchJson<ConfigSchema>('/api/config/hub/schema'),
};

// --- Status API ---

export const statusApi = {
  meta: () => fetchJson<MetaInfo>('/api/meta'),
  queues: () => fetchJson<QueueStatus>('/api/queues'),
};

// --- SignalR connection ---

export type LogEntry = {
  timestamp: string;
  identity: string;
  message: string;
};

let connection: signalR.HubConnection | null = null;

export function getSignalRConnection(): signalR.HubConnection {
  if (!connection) {
    connection = new signalR.HubConnectionBuilder()
      .withUrl('/hubs/logs')
      .withAutomaticReconnect()
      .build();
  }
  return connection;
}
