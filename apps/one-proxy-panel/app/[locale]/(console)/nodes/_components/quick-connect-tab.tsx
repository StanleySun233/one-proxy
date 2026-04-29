'use client';

import {useTranslations} from 'next-intl';
import {UseFormReturn} from 'react-hook-form';
import {useQuery} from '@tanstack/react-query';

import {fetchEnums} from '@/lib/api';
import {QuickConnectFormValues} from './types';

export function QuickConnectTab({
  form,
  submitting,
  onSubmit
}: {
  form: UseFormReturn<QuickConnectFormValues>;
  submitting: boolean;
  onSubmit: (data: QuickConnectFormValues) => void;
}) {
  const t = useTranslations();
  const {data: enums} = useQuery({queryKey: ['enums'], queryFn: () => fetchEnums()});
  const modeOptions = enums?.node_mode ? Object.entries(enums.node_mode).map(([value, item]) => ({value, label: item.name})) : [];
  return (
    <form className="nodes-form-grid" onSubmit={(e) => { e.preventDefault(); e.stopPropagation(); form.handleSubmit(onSubmit)(e); }}>
      <div className="field-stack">
        <span>{t('nodes.quickConnect.nodeAddress')}</span>
        <input className="field-input" placeholder="relay.example.com:2988" {...form.register('address', {required: t('nodes.quickConnect.addressRequired')})} />
        {form.formState.errors.address ? <p className="error-text">{form.formState.errors.address.message}</p> : null}
      </div>
      <div className="field-stack">
        <span>{t('nodes.quickConnect.joinPassword')}</span>
        <input className="field-input" type="password" placeholder="password" {...form.register('password', {required: t('nodes.quickConnect.passwordRequired')})} />
        {form.formState.errors.password ? <p className="error-text">{form.formState.errors.password.message}</p> : null}
      </div>
      <div className="field-stack">
        <span>{t('nodes.quickConnect.newJoinPassword')}</span>
        <input className="field-input" type="password" placeholder={t('nodes.quickConnect.newJoinPasswordHint')} {...form.register('newPassword')} />
        <p className="field-hint">If the node started without `NODE_JOIN_PASSWORD`, enter `password` above and set the replacement password here.</p>
      </div>
      <div className="field-stack">
        <span>{t('nodes.quickConnect.name')}</span>
        <input className="field-input" placeholder="node-b" {...form.register('name', {required: t('nodes.quickConnect.nameRequired')})} />
        {form.formState.errors.name ? <p className="error-text">{form.formState.errors.name.message}</p> : null}
      </div>
      <div className="field-stack">
        <span>{t('nodes.quickConnect.scopeKey')}</span>
        <input className="field-input" placeholder="target-scope" {...form.register('scopeKey', {required: t('nodes.quickConnect.scopeRequired')})} />
        {form.formState.errors.scopeKey ? <p className="error-text">{form.formState.errors.scopeKey.message}</p> : null}
      </div>
      <div className="field-stack">
        <span>{t('nodes.quickConnect.mode')}</span>
        <select className="field-select" {...form.register('mode', {required: true})}>
          {modeOptions.map((opt) => (
            <option key={opt.value} value={opt.value}>{opt.label}</option>
          ))}
        </select>
      </div>
      <div className="field-stack">
        <span>{t('nodes.quickConnect.parentNodeId')}</span>
        <input className="field-input" placeholder="optional upstream node id" {...form.register('parentNodeId')} />
      </div>
      <div className="field-stack">
        <span>{t('nodes.quickConnect.publicHost')}</span>
        <input className="field-input" placeholder="optional, fallback to address host" {...form.register('publicHost')} />
      </div>
      <div className="field-stack">
        <span>{t('nodes.quickConnect.publicPort')}</span>
        <input className="field-input" placeholder="optional, fallback to address port" type="number" {...form.register('publicPort')} />
      </div>
      <div className="field-stack nodes-form-full">
        <span>{t('nodes.quickConnect.controlPlaneUrl')}</span>
        <input className="field-input" placeholder="https://panel.example.com" {...form.register('controlPlaneUrl', {required: t('nodes.quickConnect.controlPlaneUrlRequired')})} />
        {form.formState.errors.controlPlaneUrl ? <p className="error-text">{form.formState.errors.controlPlaneUrl.message}</p> : null}
      </div>
      <div className="submit-row nodes-form-full">
        <button className="primary-button" disabled={submitting} type="submit">
          {submitting ? t('nodes.quickConnect.submitting') : t('nodes.quickConnect.connectNode')}
        </button>
        <p className="field-hint">{t('nodes.quickConnect.nodeDefaultPasswordHint')}</p>
      </div>
    </form>
  );
}
