'use client';

import {UseFormReturn} from 'react-hook-form';

export type RouteRuleGroupFormValues = {
  name: string;
  description: string;
  enabled: boolean;
};

type RouteRuleGroupFormProps = {
  form: UseFormReturn<RouteRuleGroupFormValues>;
  editing: boolean;
  pending: boolean;
  t: (key: string) => string;
  routesT: (key: string) => string;
  onCancel: () => void;
  onSubmit: (values: RouteRuleGroupFormValues) => void;
};

export function defaultRouteRuleGroupFormValues(): RouteRuleGroupFormValues {
  return {
    name: '',
    description: '',
    enabled: true
  };
}

export function RouteRuleGroupForm({form, editing, pending, t, routesT, onCancel, onSubmit}: RouteRuleGroupFormProps) {
  return (
    <form
      className="sub-grid"
      onSubmit={(event) => {
        form.handleSubmit(onSubmit)(event);
      }}
    >
      <div className="field-stack">
        <span>{t('common.name')}</span>
        <input
          aria-invalid={form.formState.errors.name ? 'true' : 'false'}
          className="field-input"
          {...form.register('name', {required: routesT('routeGroupNameRequired')})}
        />
        {form.formState.errors.name ? <p className="error-text">{form.formState.errors.name.message}</p> : null}
      </div>
      <div className="field-stack">
        <span>{routesT('routeGroupDescription')}</span>
        <textarea className="field-textarea" rows={3} {...form.register('description')} />
      </div>
      <label className="toggle-inline">
        <input type="checkbox" {...form.register('enabled')} />
        <span>{t('common.enabled')}</span>
      </label>
      <div className="submit-row">
        <button className="primary-button" disabled={pending} type="submit">
          {pending ? t('common.submitting') : editing ? routesT('saveGroup') : routesT('createGroup')}
        </button>
        {editing ? (
          <button className="secondary-button" onClick={onCancel} type="button">
            {t('common.cancel')}
          </button>
        ) : null}
      </div>
    </form>
  );
}
