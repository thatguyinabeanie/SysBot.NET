import { LogViewer } from '../components/LogViewer';
import { useLogs } from '../hooks/useLogs';

export function LogsPage() {
  const { logs, clear } = useLogs();

  return (
    <div className="h-full">
      <LogViewer logs={logs} onClear={clear} />
    </div>
  );
}
