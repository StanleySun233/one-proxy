type CredentialMode = 'manual' | 'saved';

type CredentialModeToggleProps = {
  mode: CredentialMode;
  onChange: (mode: CredentialMode) => void;
  manualLabel: string;
  savedLabel: string;
};

export function CredentialModeToggle({mode, onChange, manualLabel, savedLabel}: CredentialModeToggleProps) {
  return (
    <div className="remote-mode-toggle" role="group">
      <button className={mode === 'manual' ? 'is-active' : ''} onClick={() => onChange('manual')} type="button">
        {manualLabel}
      </button>
      <button className={mode === 'saved' ? 'is-active' : ''} onClick={() => onChange('saved')} type="button">
        {savedLabel}
      </button>
    </div>
  );
}
