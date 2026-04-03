import type { SchemaProperty } from '../api/types';

const inputClass = 'bg-surface-0 border border-border rounded-lg px-3 py-1 text-sm text-txt focus:outline-none focus:ring-2 focus:ring-accent/30 transition-all';

interface Props {
  name: string;
  schema: SchemaProperty;
  value: unknown;
  onChange: (name: string, value: unknown) => void;
}

export function SettingsField({ name, schema, value, onChange }: Props) {
  if (schema.type === 'object' && schema.properties) {
    return <NestedObject name={name} schema={schema} value={value as Record<string, unknown>} onChange={onChange} />;
  }

  return (
    <div className="flex items-center justify-between gap-6 py-2.5 px-2 rounded-lg hover:bg-surface-2/30 transition-colors">
      <div className="flex-1 min-w-0">
        <label className="text-sm text-txt font-medium">{name}</label>
        {schema.description && <p className="text-xs text-txt-muted mt-0.5 leading-relaxed">{schema.description}</p>}
      </div>
      <div className="shrink-0">
        {schema.type === 'boolean' ? (
          <button onClick={() => onChange(name, !value)}
            className={`relative w-10 h-5 rounded-full transition-colors cursor-pointer ${value ? 'bg-accent' : 'bg-surface-3'}`}>
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

function NestedObject({ name, schema, value, onChange }: {
  name: string; schema: SchemaProperty; value: Record<string, unknown>; onChange: (name: string, value: unknown) => void;
}) {
  return (
    <details className="group/nested">
      <summary className="cursor-pointer flex items-center gap-2 py-2.5 px-2 rounded-lg hover:bg-surface-2/30 transition-colors">
        <span className="text-txt-faint text-xs transition-transform group-open/nested:rotate-90">&#9654;</span>
        <span className="text-sm text-txt font-medium">{name}</span>
        {schema.description && <span className="text-xs text-txt-faint ml-1">{schema.description}</span>}
      </summary>
      <div className="ml-4 pl-4 border-l border-border-subtle mt-1 mb-2">
        {schema.properties && Object.entries(schema.properties).map(([key, prop]) => (
          <SettingsField key={key} name={key} schema={prop}
            value={(value ?? {})[key] ?? prop.value}
            onChange={(fieldName, fieldValue) => onChange(name, { ...value, [fieldName]: fieldValue })} />
        ))}
      </div>
    </details>
  );
}
