import { useState } from 'react';
import { useTheme } from '../hooks/useTheme';

const tabs = ['Dashboard', 'Settings', 'Logs'] as const;
type Tab = (typeof tabs)[number];

const tabIcons: Record<Tab, string> = {
  Dashboard: '⬡',
  Settings: '⚙',
  Logs: '▤',
};

const themeIcons: Record<string, string> = {
  light: '☀',
  dark: '☾',
  system: '◑',
};

export function Layout({ children }: { children: Record<Tab, React.ReactNode> }) {
  const [active, setActive] = useState<Tab>('Dashboard');
  const { theme, cycle } = useTheme();

  return (
    <div className="min-h-screen bg-surface-0 text-txt flex flex-col">
      {/* Header */}
      <header className="bg-surface-1 border-b border-border px-6 py-3 flex items-center gap-6">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center text-white text-sm font-bold shadow-sm">
            S
          </div>
          <h1 className="text-base font-semibold tracking-tight text-txt">
            SysBot<span className="text-accent">.NET</span>
          </h1>
        </div>
        <nav className="flex gap-0.5 ml-2 bg-surface-2/60 rounded-lg p-0.5">
          {tabs.map((tab) => (
            <button
              key={tab}
              onClick={() => setActive(tab)}
              className={`px-4 py-1.5 text-sm rounded-md transition-all duration-150 cursor-pointer flex items-center gap-1.5 ${
                active === tab
                  ? 'bg-surface-3 text-txt shadow-sm'
                  : 'text-txt-muted hover:text-txt-secondary hover:bg-surface-3/50'
              }`}
            >
              <span className="text-xs opacity-50">{tabIcons[tab]}</span>
              {tab}
            </button>
          ))}
        </nav>

        {/* Theme toggle */}
        <button
          onClick={cycle}
          title={`Theme: ${theme}`}
          className="ml-auto w-8 h-8 flex items-center justify-center rounded-lg bg-surface-2/60 hover:bg-surface-3 text-txt-muted hover:text-txt-secondary transition-colors cursor-pointer text-sm"
        >
          {themeIcons[theme]}
        </button>
      </header>

      {/* Content */}
      <main className="flex-1 p-6 overflow-hidden">
        {children[active]}
      </main>
    </div>
  );
}
