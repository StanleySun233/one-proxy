import {ReactNode} from 'react';
import {X} from 'lucide-react';

type ConsolePageProps = {
  eyebrow?: string;
  title: string;
  actions?: ReactNode;
  children: ReactNode;
};

export function ConsolePage({eyebrow, title, actions, children}: ConsolePageProps) {
  return (
    <section className="console-page">
      <div className="console-page-head">
        <div className="console-page-title">
          {eyebrow ? <p>{eyebrow}</p> : null}
          <h3>{title}</h3>
        </div>
        {actions ? <div className="console-page-actions">{actions}</div> : null}
      </div>
      <div className="console-page-body">{children}</div>
    </section>
  );
}

export function ConsoleFilterBar({title, children, actions}: {title?: ReactNode; children?: ReactNode; actions?: ReactNode}) {
  return (
    <div className="console-filter-bar">
      {title ? <div className="console-filter-title">{title}</div> : null}
      <div className="console-filter-fields">{children}</div>
      {actions ? <div className="console-filter-actions">{actions}</div> : null}
    </div>
  );
}

type ConsoleListProps = {
  title?: string;
  count?: number;
  actions?: ReactNode;
  children: ReactNode;
};

export function ConsoleList({title, count, actions, children}: ConsoleListProps) {
  return (
    <section className="console-list">
      {title || typeof count === 'number' || actions ? (
        <div className="console-list-head">
          <div className="console-list-title">
            {title ? <h4>{title}</h4> : null}
            {typeof count === 'number' ? <span>{count}</span> : null}
          </div>
          {actions ? <div className="console-list-actions">{actions}</div> : null}
        </div>
      ) : null}
      <div className="console-list-body">{children}</div>
    </section>
  );
}

type ConsoleCrudModalProps = {
  open: boolean;
  title: string;
  subtitle?: string;
  onClose: () => void;
  children: ReactNode;
  footer?: ReactNode;
};

export function ConsoleCrudModal({open, title, subtitle, onClose, children, footer}: ConsoleCrudModalProps) {
  if (!open) {
    return null;
  }
  return (
    <div className="console-modal-backdrop" role="presentation">
      <section aria-modal="true" className="console-modal" role="dialog">
        <div className="console-modal-head">
          <div>
            <h3>{title}</h3>
            {subtitle ? <p>{subtitle}</p> : null}
          </div>
          <button aria-label="Close" className="ghost-button console-modal-close" onClick={onClose} type="button">
            <X size={16} />
          </button>
        </div>
        <div className="console-modal-body">{children}</div>
        {footer ? <div className="console-modal-footer">{footer}</div> : null}
      </section>
    </div>
  );
}
