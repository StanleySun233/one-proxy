'use client';

import {Children, ReactNode, SelectHTMLAttributes} from 'react';

type CapsuleSelectProps = SelectHTMLAttributes<HTMLSelectElement> & {
  icon: ReactNode;
};

export function CapsuleSelect({icon, className, children, ...props}: CapsuleSelectProps) {
  return (
    <div className="capsule-select-shell">
      <span aria-hidden="true" className="capsule-select-icon">
        {icon}
      </span>
      <select className={className ? `capsule-select-field ${className}` : 'capsule-select-field'} {...props}>
        {children}
      </select>
    </div>
  );
}

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
