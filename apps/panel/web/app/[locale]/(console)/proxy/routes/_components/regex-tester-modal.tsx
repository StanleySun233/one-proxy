'use client';

import {useState} from 'react';
import {X} from 'lucide-react';
import {useTranslations} from 'next-intl';

type RegexTesterModalProps = {
  initialPattern: string;
  onClose: () => void;
};

export function RegexTesterModal({initialPattern, onClose}: RegexTesterModalProps) {
  const t = useTranslations();
  const routesT = useTranslations('proxyRoutes');
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
          <h3>{routesT('regexTester')}</h3>
          <button className="secondary-button" onClick={onClose} type="button">
            <X size={16} />
          </button>
        </div>

        <label className="field-stack">
          <span>{routesT('regexPattern')}</span>
          <input
            aria-invalid={result && !result.valid ? 'true' : 'false'}
            className="field-input mono"
            onChange={(e) => setPattern(e.target.value)}
            placeholder={routesT('regexPatternPlaceholder')}
            value={pattern}
          />
        </label>

        <label className="field-stack">
          <span>{routesT('testString')}</span>
          <input
            className="field-input mono"
            onChange={(e) => setTestString(e.target.value)}
            placeholder={routesT('testStringPlaceholder')}
            value={testString}
          />
        </label>

        <div className="submit-row">
          <button className="primary-button" disabled={!pattern || !testString} onClick={handleTest} type="button">
            {routesT('test')}
          </button>
        </div>

        {result && (
          <div className="token-box">
            {!result.valid ? (
              <>
                <strong style={{color: 'var(--danger)'}}>{routesT('invalidRegexTitle')}</strong>
                <span className="field-hint" style={{color: 'var(--danger)'}}>{result.error}</span>
              </>
            ) : result.matches ? (
              <>
                <strong style={{color: 'var(--success)'}}>{routesT('matches')}</strong>
                {result.groups.length > 0 && (
                  <div className="field-stack" style={{gap: 4}}>
                    <span>{routesT('capturedGroups')}</span>
                    {result.groups.map((group, i) => (
                      <span className="mono" key={i}>
                        {routesT('groupLabel', {index: i + 1})}: {group || t('common.empty')}
                      </span>
                    ))}
                  </div>
                )}
              </>
            ) : (
              <strong style={{color: 'var(--muted)'}}>{routesT('noMatches')}</strong>
            )}
          </div>
        )}

        <div className="submit-row" style={{justifyContent: 'flex-end'}}>
          <button className="secondary-button" onClick={onClose} type="button">
            {t('common.close')}
          </button>
        </div>
      </div>
    </div>
  );
}
