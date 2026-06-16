'use client';

import {UseFormReturn} from 'react-hook-form';

import {ConsoleCrudModal} from '@/components/console-template';
import {Chain, RouteRuleGroup, RouteRuleValidationResult, Scope} from '@/lib/types';

import {RouteRuleForm} from './route-rule-form';
import {RouteRuleFormValues} from '../_lib/form';

type SelectOption = {
  value: string;
  label: string;
};

type RouteRuleEditorModalProps = {
  open: boolean;
  editingRule: boolean;
  form: UseFormReturn<RouteRuleFormValues>;
  groups: RouteRuleGroup[];
  chains: Chain[];
  scopes: Scope[];
  matchTypeOptions: SelectOption[];
  actionTypeOptions: SelectOption[];
  actionType: string;
  matchType: string;
  matchValuePlaceholder: string;
  selectedChain?: Chain;
  validationPending: boolean;
  validationResult: RouteRuleValidationResult | null;
  createPending: boolean;
  updatePending: boolean;
  t: (key: string) => string;
  routesT: (key: string, values?: Record<string, string | number>) => string;
  onClose: () => void;
  onSubmit: (values: RouteRuleFormValues) => void;
  onOpenRegexTester: () => void;
  validateMatchValue: (matchType: string, value: string) => string | true;
};

export function RouteRuleEditorModal({
  open,
  editingRule,
  form,
  groups,
  chains,
  scopes,
  matchTypeOptions,
  actionTypeOptions,
  actionType,
  matchType,
  matchValuePlaceholder,
  selectedChain,
  validationPending,
  validationResult,
  createPending,
  updatePending,
  t,
  routesT,
  onClose,
  onSubmit,
  onOpenRegexTester,
  validateMatchValue
}: RouteRuleEditorModalProps) {
  return (
    <ConsoleCrudModal onClose={onClose} open={open} subtitle={routesT('validation')} title={editingRule ? routesT('editRule') : routesT('createRule')}>
      <RouteRuleForm
        actionType={actionType}
        actionTypeOptions={actionTypeOptions}
        chains={chains}
        createPending={createPending}
        editingRule={editingRule}
        form={form}
        groups={groups}
        matchType={matchType}
        matchTypeOptions={matchTypeOptions}
        matchValuePlaceholder={matchValuePlaceholder}
        onCancel={onClose}
        onOpenRegexTester={onOpenRegexTester}
        onSubmit={onSubmit}
        routesT={routesT}
        scopes={scopes}
        selectedChain={selectedChain}
        t={t}
        updatePending={updatePending}
        validateMatchValue={validateMatchValue}
        validationPending={validationPending}
        validationResult={validationResult}
      />
    </ConsoleCrudModal>
  );
}
