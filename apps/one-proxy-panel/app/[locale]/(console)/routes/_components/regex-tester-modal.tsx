'use client';

import {useState} from 'react';
import {X} from 'lucide-react';

type RegexTesterModalProps = {
  initialPattern: string;
  onClose: () => void;
};

export function RegexTesterModal({initialPattern, onClose}: RegexTesterModalProps) {
  const [pattern, setPattern] = useState(initialPattern);
  const [testString, setTestString] = useState('');
  const [result, setResult] = useState<{valid: boolean; matches: boolean; groups: string[]; error: string} | null>(null);

  const handleTest = () => {
    let regex: RegExp;
    try {
      regex = new RegExp(pattern);
    } catch (e) {
      setResult({valid: false, matches: false, groups: [], error: (e as Error).message});
      return;
    }

    const match = regex.exec(testString);
    if (match) {
      setResult({
        valid: true,
        matches: true,
        groups: match.slice(1),
        error: ''
      });
    } else {
      setResult({valid: true, matches: false, groups: [], error: ''});
    }
  };

  return (
    <div className="dialog-backdrop" onClick={onClose}>
      <div className="dialog-panel" onClick={(e) => e.stopPropagation()}>
        <div className="panel-toolbar">
          <h3>Regex Tester</h3>
          <button className="secondary-button" onClick={onClose} type="button">
            <X size={16} />
          </button>
        </div>

        <label className="field-stack">
          <span>Regex Pattern</span>
          <input
            aria-invalid={result && !result.valid ? 'true' : 'false'}
            className="field-input mono"
            onChange={(e) => setPattern(e.target.value)}
            placeholder="^https://.*\\.example\\.com/.*"
            value={pattern}
          />
        </label>

        <label className="field-stack">
          <span>Test String</span>
          <input
            className="field-input mono"
            onChange={(e) => setTestString(e.target.value)}
            placeholder="https://api.example.com/v1/users"
            value={testString}
          />
        </label>

        <div className="submit-row">
          <button className="primary-button" disabled={!pattern || !testString} onClick={handleTest} type="button">
            Test
          </button>
        </div>

        {result && (
          <div className="token-box">
            {!result.valid ? (
              <>
                <strong style={{color: 'var(--danger)'}}>Invalid Regex</strong>
                <span className="field-hint" style={{color: 'var(--danger)'}}>{result.error}</span>
              </>
            ) : result.matches ? (
              <>
                <strong style={{color: 'var(--success)'}}>Matches</strong>
                {result.groups.length > 0 && (
                  <div className="field-stack" style={{gap: 4}}>
                    <span>Captured Groups:</span>
                    {result.groups.map((group, i) => (
                      <span className="mono" key={i}>
                        Group {i + 1}: {group || '(empty)'}
                      </span>
                    ))}
                  </div>
                )}
              </>
            ) : (
              <strong style={{color: 'var(--muted)'}}>No Matches</strong>
            )}
          </div>
        )}

        <div className="submit-row" style={{justifyContent: 'flex-end'}}>
          <button className="secondary-button" onClick={onClose} type="button">
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
