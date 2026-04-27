'use client';

import {ReactNode, useCallback, useEffect, useRef, useState} from 'react';
import {ChevronDown} from 'lucide-react';

type CapsuleSelectProps = {
  icon: ReactNode;
  value: string;
  onChange: (value: string) => void;
  options: {value: string; label: string}[];
  'aria-label'?: string;
  className?: string;
};

export function CapsuleSelect({icon, value, onChange, options, ...props}: CapsuleSelectProps) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [open]);

  const handleSelect = useCallback(
    (v: string) => {
      onChange(v);
      setOpen(false);
    },
    [onChange]
  );

  const selectedLabel = options.find((o) => o.value === value)?.label || value;

  return (
    <div className="capsule-select-shell" ref={ref}>
      <span aria-hidden="true" className="capsule-select-icon">
        {icon}
      </span>
      <button
        type="button"
        className="capsule-select-trigger"
        aria-label={props['aria-label']}
        aria-expanded={open}
        onClick={() => setOpen((prev) => !prev)}
      >
        <span>{selectedLabel}</span>
        <ChevronDown className={`capsule-select-arrow${open ? ' is-open' : ''}`} size={14} />
      </button>
      {open ? (
        <div className="capsule-select-menu">
          {options.map((opt) => (
            <button
              key={opt.value}
              type="button"
              className={`capsule-select-option${opt.value === value ? ' is-active' : ''}`}
              onClick={() => handleSelect(opt.value)}
            >
              {opt.label}
            </button>
          ))}
        </div>
      ) : null}
    </div>
  );
}

import {Children, ReactNode} from 'react';

export function CapsuleSelectGroup({children}: {children: ReactNode}) {
  const items = Children.toArray(children).filter(Boolean);

  return (
    <div className="capsule-select-group">
      {items.map((child, index) => (
        <div className="capsule-select-item" key={index}>
          {index > 0 ? <span aria-hidden="true" className="capsule-select-divider" /> : null}
          {child}
        </div>
      ))}
    </div>
  );
}
