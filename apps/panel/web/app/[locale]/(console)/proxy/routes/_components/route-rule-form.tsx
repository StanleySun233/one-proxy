'use client';

import {UseFormReturn} from 'react-hook-form';

import {Chain, RouteRuleValidationResult, Scope} from '@/lib/types';

import {RouteRuleFormValues} from '../_lib/form';
import {ValidationPanel} from './validation-panel';

type SelectOption = {
  value: string;
  label: string;
};

type RouteRuleFormProps = {
  form: UseFormReturn<RouteRuleFormValues>;
  chains: Chain[];
  scopes: Scope[];
  matchTypeOptions: SelectOption[];
  actionTypeOptions: SelectOption[];
  actionType: string;
  matchType: string;
  matchValuePlaceholder: string;
  selectedChain?: Chain;
  editingRule: boolean;
  validationPending: boolean;
  validationResult: RouteRuleValidationResult | null;
  createPending: boolean;
  updatePending: boolean;
  t: (key: string) => string;
  routesT: (key: string, values?: Record<string, string | number>) => string;
  onSubmit: (values: RouteRuleFormValues) => void;
  onCancel: () => void;
  onOpenRegexTester: () => void;
  validateMatchValue: (matchType: string, value: string) => string | true;
};

export function RouteRuleForm({
  form,
  chains,
  scopes,
  matchTypeOptions,
  actionTypeOptions,
  actionType,
  matchType,
  matchValuePlaceholder,
  selectedChain,
  editingRule,
  validationPending,
  validationResult,
  createPending,
  updatePending,
  t,
  routesT,
  onSubmit,
  onCancel,
  onOpenRegexTester,
  validateMatchValue
}: RouteRuleFormProps) {
  const scopeNameById = new Map(scopes.map((scope) => [scope.id, scope.name]));

  return (
    <div className="sub-grid">
      <div className="inline-cluster" style={{gap: 8}}>
        {validationPending && <span className="badge is-neutral">{t('common.validating')}</span>}
        {!validationPending && validationResult && (
          <span className={`badge ${validationResult.valid ? 'is-good' : 'is-danger'}`}>
            {validationResult.valid ? t('common.valid') : t('common.invalid')}
          </span>
        )}
      </div>
      <form
        className="sub-grid"
        onSubmit={(e) => {
          form.handleSubmit(onSubmit)(e);
        }}
      >
        <div className="field-stack">
          <span>{routesT('priority')}</span>
          <input
            aria-invalid={form.formState.errors.priority ? 'true' : 'false'}
            className="field-input"
            type="number"
            {...form.register('priority', {
              required: routesT('priorityRequired'),
              validate: (value) => Number(value) > 0 || routesT('priorityPositive')
            })}
          />
          {form.formState.errors.priority ? <p className="error-text">{form.formState.errors.priority.message}</p> : null}
        </div>
        <div className="field-stack">
          <span>{routesT('matchType')}</span>
          <select
            aria-invalid={form.formState.errors.matchType ? 'true' : 'false'}
            className="field-select"
            {...form.register('matchType', {required: routesT('matchTypeRequired')})}
          >
            {matchTypeOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
          {form.formState.errors.matchType ? <p className="error-text">{form.formState.errors.matchType.message}</p> : null}
        </div>
        <div className="field-stack">
          <span>{routesT('matchValue')}</span>
          <div className="inline-cluster">
            <input
              aria-invalid={form.formState.errors.matchValue ? 'true' : 'false'}
              className="field-input"
              placeholder={matchValuePlaceholder}
              style={{flex: 1}}
              {...form.register('matchValue', {
                required: routesT('matchValueRequired'),
                validate: (value) => validateMatchValue(matchType, value)
              })}
            />
            {matchType === 'url_regex' && (
              <button className="secondary-button" onClick={onOpenRegexTester} type="button">
                {routesT('testRegex')}
              </button>
            )}
          </div>
          {form.formState.errors.matchValue ? <p className="error-text">{form.formState.errors.matchValue.message}</p> : null}
        </div>
        <div className="field-stack">
          <span>{routesT('actionType')}</span>
          <select className="field-select" {...form.register('actionType', {required: true})}>
            {actionTypeOptions.map((opt) => (
              <option key={opt.value} value={opt.value}>{opt.label}</option>
            ))}
          </select>
        </div>
        <div className="field-stack">
          <span>{routesT('chain')}</span>
          <select
            aria-invalid={form.formState.errors.chainId ? 'true' : 'false'}
            className="field-select"
            disabled={actionType !== 'chain'}
            {...form.register('chainId', {
              validate: (value) => (actionType !== 'chain' || value.trim() !== '' ? true : routesT('chainRequired'))
            })}
          >
            <option value="">{t('common.selectChain')}</option>
            {chains.map((chain) => {
              const hopCount = Array.isArray(chain.hops) ? chain.hops.length : 0;
              const hopDisplay = hopCount > 0 ? ` (${Array.from({length: hopCount}, (_, i) => i + 1).join(' -> ')})` : '';
              return (
                <option key={chain.id} value={chain.id}>
                  {chain.name}
                  {hopDisplay}
                </option>
              );
            })}
          </select>
          {form.formState.errors.chainId ? <p className="error-text">{form.formState.errors.chainId.message}</p> : null}
          {selectedChain && actionType === 'chain' ? (
            <div className="field-hint">
              <span className="muted-text">
                {routesT('destinationScope')}: <strong>{scopeNameById.get(selectedChain.destinationScope) || t('common.unknown')}</strong>
              </span>
            </div>
          ) : null}
        </div>
        <div className="field-stack">
          <span>{routesT('destinationScope')}</span>
          <select
            aria-invalid={form.formState.errors.destinationScope ? 'true' : 'false'}
            className="field-select"
            disabled={actionType !== 'direct'}
            {...form.register('destinationScope', {
              validate: (value) => (actionType !== 'direct' || value.trim() !== '' ? true : routesT('destinationScopeRequired'))
            })}
          >
            <option value="">{routesT('destinationScopePlaceholder')}</option>
            {scopes.map((scope) => (
              <option key={scope.id} value={scope.id}>{scope.name}</option>
            ))}
          </select>
          {form.formState.errors.destinationScope ? <p className="error-text">{form.formState.errors.destinationScope.message}</p> : null}
        </div>
        <label className="toggle-inline">
          <input type="checkbox" {...form.register('enabled')} />
          <span>{t('common.enabled')}</span>
        </label>

        {validationResult && (
          <ValidationPanel pending={validationPending} result={validationResult} routesT={routesT} t={t} />
        )}

        <div className="submit-row">
          <button className="primary-button" disabled={createPending || updatePending} type="submit">
            {createPending || updatePending
              ? t('common.submitting')
              : editingRule ? routesT('saveRule') : routesT('createRule')}
          </button>
          {editingRule ? (
            <button className="secondary-button" onClick={onCancel} type="button">
              {t('common.cancel')}
            </button>
          ) : null}
        </div>
      </form>
    </div>
  );
}
