import { useEffect, useRef, useState, useCallback } from 'react';
import { getSignalRConnection } from '../api/client';
import type { LogEntry } from '../api/client';

const MAX_ENTRIES = 5000;

export function useLogs() {
  const bufferRef = useRef<LogEntry[]>([]);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const rafRef = useRef<number>(0);

  const flush = useCallback(() => {
    setLogs([...bufferRef.current]);
    rafRef.current = 0;
  }, []);

  useEffect(() => {
    const conn = getSignalRConnection();

    const onLog = (timestamp: string, identity: string, message: string) => {
      const entry: LogEntry = { timestamp, identity, message };
      const buf = bufferRef.current;
      buf.push(entry);
      // Trim from the front if over limit
      if (buf.length > MAX_ENTRIES) {
        bufferRef.current = buf.slice(buf.length - MAX_ENTRIES);
      }
      // Batch DOM updates via requestAnimationFrame
      if (!rafRef.current) {
        rafRef.current = requestAnimationFrame(flush);
      }
    };

    const onEcho = (timestamp: string, message: string) => {
      onLog(timestamp, 'Echo', message);
    };

    conn.on('ReceiveLog', onLog);
    conn.on('ReceiveEcho', onEcho);

    if (conn.state === 'Disconnected') {
      conn.start().catch(console.error);
    }

    return () => {
      conn.off('ReceiveLog', onLog);
      conn.off('ReceiveEcho', onEcho);
      if (rafRef.current) cancelAnimationFrame(rafRef.current);
    };
  }, [flush]);

  const clear = useCallback(() => {
    bufferRef.current = [];
    setLogs([]);
  }, []);

  return { logs, clear };
}
