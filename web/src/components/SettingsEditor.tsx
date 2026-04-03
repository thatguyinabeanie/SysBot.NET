import { useState, useEffect } from 'react';
import { useConfigSchema, useHubConfig, usePatchHubConfig } from '../hooks/useConfig';
import { SettingsSection } from './SettingsSection';

export function SettingsEditor() {
  const { data: schema, isLoading: schemaLoading } = useConfigSchema();
  const { data: config, isLoading: configLoading } = useHubConfig();
  const patchConfig = usePatchHubConfig();

  const [edits, setEdits] = useState<Record<string, unknown>>({});
  const [activeCategory, setActiveCategory] = useState<string | null>(null);
  const [activeSection, setActiveSection] = useState<string | null>(null);

  useEffect(() => { if (config) setEdits({}); }, [config]);

  // Set initial active category when schema loads
  useEffect(() => {
    if (schema && !activeCategory) {
      const first = Object.keys(schema.categories)[0];
      if (first) setActiveCategory(first);
    }
  }, [schema, activeCategory]);

  // Reset active section when category changes
  useEffect(() => {
    if (schema && activeCategory) {
      const sections = Object.keys(schema.categories[activeCategory] ?? {});
      setActiveSection(sections[0] ?? null);
    }
  }, [schema, activeCategory]);

  if (schemaLoading || configLoading) {
    return <div className="flex items-center justify-center h-64 text-txt-muted"><span className="animate-pulse">Loading settings...</span></div>;
  }
  if (!schema || !config) {
    return <p className="text-danger">Failed to load settings.</p>;
  }

  const categories = Object.keys(schema.categories);
  const sections = activeCategory ? Object.keys(schema.categories[activeCategory] ?? {}) : [];
  const activeSectionSchema = activeCategory && activeSection
    ? schema.categories[activeCategory]?.[activeSection]
    : null;

  const handleChange = (name: string, value: unknown) => {
    setEdits((prev) => ({ ...prev, [name]: value }));
  };

  const hasEdits = Object.keys(edits).length > 0;

  return (
    <div className="flex gap-6 h-full max-w-6xl mx-auto">
      {/* Sidebar — categories */}
      <nav className="w-44 shrink-0 flex flex-col gap-1">
        <h2 className="text-[11px] font-semibold text-txt-muted uppercase tracking-wider px-3 mb-2">Categories</h2>
        {categories.map((cat) => (
          <button
            key={cat}
            onClick={() => setActiveCategory(cat)}
            className={`text-left text-sm px-3 py-2 rounded-lg transition-colors cursor-pointer ${
              activeCategory === cat
                ? 'bg-accent-subtle text-accent font-medium'
                : 'text-txt-secondary hover:text-txt hover:bg-surface-2/50'
            }`}
          >
            {cat}
          </button>
        ))}

        <div className="mt-auto pt-4 border-t border-border-subtle">
          {patchConfig.isSuccess && <span className="text-ok text-xs font-medium block px-3 mb-2">Saved</span>}
          {patchConfig.isError && <span className="text-danger text-xs block px-3 mb-2">Error saving</span>}
          <button
            onClick={() => patchConfig.mutate(edits)}
            disabled={!hasEdits || patchConfig.isPending}
            className="w-full bg-accent hover:bg-accent-hover disabled:opacity-40 disabled:cursor-not-allowed text-white text-sm font-medium px-3 py-2 rounded-lg transition-colors cursor-pointer shadow-sm"
          >
            {hasEdits ? 'Save Changes' : 'No Changes'}
          </button>
        </div>
      </nav>

      {/* Main content */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Section tabs */}
        {sections.length > 1 && (
          <div className="flex gap-1 mb-4 flex-wrap bg-surface-2/40 rounded-lg p-1 self-start">
            {sections.map((sec) => (
              <button
                key={sec}
                onClick={() => setActiveSection(sec)}
                className={`text-xs font-medium px-3 py-1.5 rounded-md transition-colors cursor-pointer ${
                  activeSection === sec
                    ? 'bg-surface-0 text-txt shadow-sm'
                    : 'text-txt-muted hover:text-txt-secondary hover:bg-surface-3/50'
                }`}
              >
                {sec}
              </button>
            ))}
          </div>
        )}

        {/* Active section content */}
        <div className="flex-1 overflow-y-auto">
          {activeSectionSchema && activeSection && (
            <SettingsSection
              key={`${activeCategory}-${activeSection}`}
              name={activeSection}
              schema={activeSectionSchema}
              value={(config as Record<string, unknown>)[activeSection] ?? activeSectionSchema.value}
              onChange={handleChange}
            />
          )}
        </div>
      </div>
    </div>
  );
}
