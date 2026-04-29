'use client';

import {useTranslations} from 'next-intl';
import {UseFormReturn} from 'react-hook-form';
import {useQuery} from '@tanstack/react-query';

import {fetchEnums} from '@/lib/api';
import {NodeFormValues} from './types';

export function ManualNodeTab({
  form,
  submitting,
  onSubmit
}: {
  form: UseFormReturn<NodeFormValues>;
  submitting: boolean;
  onSubmit: (data: NodeFormValues) => void;
}) {
  const t = useTranslations();
  const {data: enums} = useQuery({queryKey: ['enums'], queryFn: () => fetchEnums()});
  const modeOptions = enums?.node_mode ? Object.entries(enums.node_mode).map(([value, item]) => ({value, label: item.name})) : [];
  return (
    <form className="nodes-form-grid" onSubmit={form.handleSubmit(onSubmit)}>
      <div className="field-stack">
        <span>{t('nodes.manual.name')}</span>
        <input className="field-input" placeholder="relay-a" {...form.register('name', {required: t('nodes.manual.nameRequired')})} />
        {form.formState.errors.name ? <p className="error-text">{form.formState.errors.name.message}</p> : null}
      </div>
      <div className="field-stack">
        <span>{t('nodes.manual.mode')}</span>
        <select className="field-select" {...form.register('mode', {required: true})}>
          {modeOptions.map((opt) => (
            <option key={opt.value} value={opt.value}>{opt.label}</option>
          ))}
        </select>
      </div>
      <div className="field-stack">
        <span>{t('nodes.manual.scopeKey')}</span>
        <input className="field-input" placeholder="target-scope" {...form.register('scopeKey', {required: t('nodes.manual.scopeRequired')})} />
        {form.formState.errors.scopeKey ? <p className="error-text">{form.formState.errors.scopeKey.message}</p> : null}
      </div>
      <div className="field-stack">
        <span>{t('nodes.manual.parentNodeId')}</span>
        <input className="field-input" placeholder="optional upstream node id" {...form.register('parentNodeId')} />
      </div>
      <div className="field-stack">
        <span>{t('nodes.manual.publicHost')}</span>
        <input className="field-input" placeholder="relay.example.com" {...form.register('publicHost')} />
      </div>
      <div className="field-stack">
        <span>{t('nodes.manual.publicPort')}</span>
        <input
          className="field-input"
          placeholder="2988"
          type="number"
          {...form.register('publicPort', {
            validate: (value) => {
              if (!value) {
                return true;
              }
              return Number(value) > 0 || t('nodes.manual.portValidation');
            }
          })}
        />
        {form.formState.errors.publicPort ? <p className="error-text">{form.formState.errors.publicPort.message}</p> : null}
      </div>
      <div className="submit-row nodes-form-full">
        <button className="primary-button" disabled={submitting} type="submit">
          {submitting ? t('nodes.manual.submitting') : t('nodes.manual.createNodeRecord')}
        </button>
        <p className="field-hint">{t('nodes.manual.manualHint')}</p>
      </div>
    </form>
  );
}
