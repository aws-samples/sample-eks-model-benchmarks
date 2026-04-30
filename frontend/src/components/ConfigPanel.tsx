import { useState } from "react";

export interface ConfigField {
  label: string;
  value: string | number | null | undefined;
}

interface Props {
  headline: ConfigField[];
  details?: ConfigField[];
}

function formatValue(v: ConfigField["value"]): string {
  if (v === null || v === undefined || v === "") return "—";
  if (typeof v === "number") return String(v);
  return v;
}

export default function ConfigPanel({ headline, details }: Props) {
  const [open, setOpen] = useState(false);

  return (
    <div className="panel p-6 mb-8">
      <div className="flex items-baseline justify-between mb-4 pb-2 border-b border-line">
        <h2 className="font-sans text-[14px] font-medium tracking-mech text-ink-0">
          Configuration
        </h2>
        {details && details.length > 0 && (
          <button
            type="button"
            onClick={() => setOpen((v) => !v)}
            className="eyebrow text-ink-1 hover:text-signal focus:outline-none focus:text-signal"
          >
            {open ? "− DETAILS" : "+ DETAILS"}
          </button>
        )}
      </div>
      <dl className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {headline.map(({ label, value }) => (
          <div key={label}>
            <dt className="eyebrow">{label}</dt>
            <dd className="font-mono text-[12.5px] text-ink-0">{formatValue(value)}</dd>
          </div>
        ))}
      </dl>
      {open && details && details.length > 0 && (
        <dl className="mt-5 pt-5 border-t border-line/60 grid grid-cols-2 md:grid-cols-4 gap-4">
          {details.map(({ label, value }) => (
            <div key={label}>
              <dt className="eyebrow">{label}</dt>
              <dd className="font-mono text-[12.5px] text-ink-1">{formatValue(value)}</dd>
            </div>
          ))}
        </dl>
      )}
    </div>
  );
}
