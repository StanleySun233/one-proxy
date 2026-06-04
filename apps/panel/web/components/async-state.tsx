'use client';

type AsyncStateProps = {
  title: string;
  detail: string;
  actionLabel?: string;
  onAction?: () => void;
};

export function AsyncState({title, detail, actionLabel, onAction}: AsyncStateProps) {
  return (
    <div className="async-state">
      <strong>{title}</strong>
      <p>{detail}</p>
      {actionLabel && onAction ? (
        <button className="secondary-button" onClick={onAction} type="button">
          {actionLabel}
        </button>
      ) : null}
    </div>
  );
}
