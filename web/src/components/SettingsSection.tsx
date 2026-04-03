import type { SchemaProperty } from '../api/types';

const inputClass = 'bg-surface-0 border border-border rounded-lg px-3 py-1.5 text-sm text-txt focus:outline-none focus:ring-2 focus:ring-accent/30 transition-all';

interface Props {
  name: string;
  schema: SchemaProperty;
  value: unknown;
  onChange: (name: string, value: unknown) => void;
}

export function SettingsSection({ name, schema, value, onChange }: Props) {
  // If it's a nested object, render as a card with all properties flat
  if (schema.type === 'object' && schema.properties) {
    const objValue = (value ?? {}) as Record<string, unknown>;
    return (
      <div className="rounded-xl border border-border-subtle bg-surface-1/60 overflow-hidden">
        <div className="px-4 py-3 bg-surface-2/40 border-b border-border-subtle">
          <h3 className="text-sm font-semibold text-txt">{name}</h3>
          {schema.description && (
            <p className="text-xs text-txt-muted mt-0.5">{schema.description}</p>
          )}
        </div>
        <div className="divide-y divide-border-subtle/50">
          {Object.entries(schema.properties).map(([key, prop]) => {
            // Nested objects within a section get rendered as a sub-section
            if (prop.type === 'object' && prop.properties) {
              return (
                <div key={key} className="px-4 py-3">
                  <h4 className="text-xs font-semibold text-txt-muted uppercase tracking-wider mb-2">{key}</h4>
                  {prop.description && <p className="text-xs text-txt-faint mb-3">{prop.description}</p>}
                  <div className="space-y-0.5">
                    {Object.entries(prop.properties).map(([subKey, subProp]) => (
                      <FieldRow key={subKey} name={subKey} schema={subProp}
                        value={((objValue[key] ?? {}) as Record<string, unknown>)[subKey] ?? subProp.value}
                        onChange={(fieldName, fieldValue) => {
                          onChange(name, {
                            ...objValue,
                            [key]: { ...((objValue[key] ?? {}) as Record<string, unknown>), [fieldName]: fieldValue },
                          });
                        }}
                      />
                    ))}
                  </div>
                </div>
              );
            }

            return (
              <FieldRow key={key} name={key} schema={prop}
                value={objValue[key] ?? prop.value}
                onChange={(fieldName, fieldValue) => {
                  onChange(name, { ...objValue, [fieldName]: fieldValue });
                }}
              />
            );
          })}
        </div>
      </div>
    );
  }

  // Top-level scalar — render as a standalone field row in a card
  return (
    <div className="rounded-xl border border-border-subtle bg-surface-1/60 overflow-hidden">
      <FieldRow name={name} schema={schema} value={value} onChange={onChange} />
    </div>
  );
}

function FieldRow({ name, schema, value, onChange }: Props) {
  return (
    <div className="flex items-center justify-between gap-6 px-4 py-3 hover:bg-surface-2/20 transition-colors">
      <div className="flex-1 min-w-0">
        <span className="text-sm text-txt font-medium">{name}</span>
        {schema.description && (
          <p className="text-xs text-txt-muted mt-0.5 leading-relaxed">{schema.description}</p>
        )}
      </div>
      <div className="shrink-0">
        {schema.type === 'boolean' ? (
          <button
            onClick={() => onChange(name, !value)}
            className={`relative w-10 h-5 rounded-full transition-colors cursor-pointer ${value ? 'bg-accent' : 'bg-surface-3'}`}
          >
            <span className={`absolute top-0.5 w-4 h-4 rounded-full bg-white shadow-sm transition-transform ${value ? 'left-5.5' : 'left-0.5'}`} />
          </button>
        ) : schema.type === 'enum' && schema.enumValues ? (
          <select value={String(value ?? '')} onChange={(e) => onChange(name, e.target.value)} className={inputClass}>
            {schema.enumValues.map((v) => <option key={v} value={v}>{v}</option>)}
          </select>
        ) : schema.type === 'integer' || schema.type === 'number' ? (
          <input type="number" value={value as number ?? 0} onChange={(e) => onChange(name, Number(e.target.value))}
            className={`${inputClass} w-28 text-right`} />
        ) : (
          <input type="text" value={String(value ?? '')} onChange={(e) => onChange(name, e.target.value)}
            className={`${inputClass} w-52`} />
        )}
      </div>
    </div>
  );
}
