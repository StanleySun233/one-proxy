'use client';

import {UseFormReturn} from 'react-hook-form';

import {ConsoleCrudModal} from '@/components/console-template';

import {RouteRuleGroupForm, RouteRuleGroupFormValues} from './route-rule-group-form';

type RouteGroupEditorModalProps = {
  open: boolean;
  editing: boolean;
  pending: boolean;
  form: UseFormReturn<RouteRuleGroupFormValues>;
  t: (key: string) => string;
  routesT: (key: string) => string;
  onClose: () => void;
  onSubmit: (values: RouteRuleGroupFormValues) => void;
};

export function RouteGroupEditorModal({open, editing, pending, form, t, routesT, onClose, onSubmit}: RouteGroupEditorModalProps) {
  return (
    <ConsoleCrudModal onClose={onClose} open={open} title={editing ? routesT('editGroup') : routesT('createGroup')}>
      <RouteRuleGroupForm
        editing={editing}
        form={form}
        onCancel={onClose}
        onSubmit={onSubmit}
        pending={pending}
        routesT={routesT}
        t={t}
      />
    </ConsoleCrudModal>
  );
}
