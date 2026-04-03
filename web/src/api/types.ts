export interface BotDto {
  id: string;
  ip: string;
  port: number;
  protocol: string;
  initialRoutine: string;
  currentRoutine: string;
  nextRoutine: string;
  isRunning: boolean;
  isPaused: boolean;
  isConnected: boolean;
  lastLog: string | null;
  lastActive: string | null;
}

export interface AddBotRequest {
  ip: string;
  port: number;
  protocol: string;
  routine: string;
}

export interface MetaInfo {
  mode: string;
  supportedRoutines: string[];
  protocols: string[];
  isRunning: boolean;
}

export interface QueueStatus {
  canQueue: boolean;
  queues: Record<string, { count: number }>;
  totalCount: number;
}

export interface SchemaProperty {
  type: string;
  description: string;
  value: unknown;
  enumValues?: string[];
  properties?: Record<string, SchemaProperty>;
}

export interface ConfigSchema {
  categories: Record<string, Record<string, SchemaProperty>>;
}
