'use client';

import {RouteRuleValidationResult} from '@/lib/types';

type ValidationPanelProps = {
  result: RouteRuleValidationResult;
  pending: boolean;
  t: (key: string) => string;
  routesT: (key: string) => string;
};

export function ValidationPanel({result, pending, t, routesT}: ValidationPanelProps) {
  return (
    <div className="probe-results-section">
      <div className="section-header">
        <h4>{routesT('validation')}</h4>
        {pending ? <span className="badge is-neutral">{t('common.validating')}</span> : (
          <span className={`badge ${result.valid ? 'is-good' : 'is-danger'}`}>
            {result.valid ? t('common.valid') : t('common.invalid')}
          </span>
        )}
      </div>
      {result.errors.map((msg, i) => (
        <div className="token-box" key={`err-${i}`} style={{borderColor: 'var(--danger)'}}>
          <span className="field-hint" style={{color: 'var(--danger)'}}>{msg}</span>
        </div>
      ))}
      {result.warnings.map((msg, i) => (
        <div className="token-box" key={`warn-${i}`} style={{borderColor: 'var(--accent)'}}>
          <span className="field-hint" style={{color: 'var(--accent)'}}>{msg}</span>
        </div>
      ))}
      {result.matchValueValidation && !result.matchValueValidation.valid && (
        <div className="token-box" style={{borderColor: 'var(--danger)'}}>
          <span className="field-hint" style={{color: 'var(--danger)'}}>{result.matchValueValidation.message}</span>
        </div>
      )}
      {result.chainValidation && !result.chainValidation.valid && (
        <div className="token-box" style={{borderColor: 'var(--danger)'}}>
          <span className="field-hint" style={{color: 'var(--danger)'}}>{routesT('chainNotFound')}</span>
        </div>
      )}
      {result.scopeValidation && !result.scopeValidation.valid && (
        <div className="token-box" style={{borderColor: 'var(--danger)'}}>
          <span className="field-hint" style={{color: 'var(--danger)'}}>{routesT('scopeNotFound')}</span>
        </div>
      )}
      {result.scopeValidation && result.scopeValidation.valid && !result.scopeValidation.matchesChainFinalHop && (
        <div className="token-box" style={{borderColor: 'var(--accent)'}}>
          <span className="field-hint" style={{color: 'var(--accent)'}}>{routesT('scopeChainMismatch')}</span>
        </div>
      )}
    </div>
  );
}
