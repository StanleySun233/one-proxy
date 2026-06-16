'use client';

import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {useForm} from 'react-hook-form';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';
import {useCallback, useEffect, useMemo, useState} from 'react';

import {AuthGate} from '@/components/auth-gate';
import {ConsoleList, ConsolePage} from '@/components/console-template';
import {ResourceGrantModal} from '@/components/resource-grant-modal';
import {useAuth} from '@/components/auth-provider';
import {createRouteRule, createRouteRuleGroup, deleteRouteRule, deleteRouteRuleGroup, fetchEnums, getChains, getPolicyRevisions, getRouteRuleGroupDeleteImpact, getRouteRuleGroups, getRouteRules, getScopes, publishPolicy, updateRouteRule, updateRouteRuleGroup} from '@/lib/api';
import {RouteRule, RouteRuleGroup, RouteRuleGroupDeleteImpact} from '@/lib/types';
import {formatControlPlaneError} from '@/lib/presentation';

import {RegexTesterModal} from './_components/regex-tester-modal';
import {RoutePolicyHistory} from './_components/route-policy-history';
import {RouteDeleteDialogs} from './_components/route-delete-dialogs';
import {RouteGroupEditorModal} from './_components/route-group-editor-modal';
import {RoutePageActions} from './_components/route-page-actions';
import {RouteRuleEditorModal} from './_components/route-rule-editor-modal';
import {defaultRouteRuleGroupFormValues, RouteRuleGroupFormValues} from './_components/route-rule-group-form';
import {RouteRuleGroupTable} from './_components/route-rule-group-table';
import {RouteRuleFilterBar} from './_components/route-rule-filter-bar';
import {RouteRuleTable} from './_components/route-rule-table';
import {useRouteRuleValidation} from './_hooks/use-route-rule-validation';
import {routeRuleDeleteSections, routeRuleGroupDeleteSections} from './_lib/delete-sections';
import {defaultRouteRuleFormValues, routeRuleFormValues, routeRuleSubmitPayload, RouteRuleFormValues, RouteRuleSubmitPayload} from './_lib/form';
import {validateMatchValue} from './_lib/validation';

export default function RoutesPage() {
  const t = useTranslations();
  const routesT = useTranslations('proxyRoutes');
  const {session, activeTenant} = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';
  const activeTenantId = session?.activeTenantId || null;
  const canWrite = session?.account.role === 'super_admin' || activeTenant?.role === 'tenant_admin';
  const globalSuperAdmin = session?.account.role === 'super_admin';
  const {data: enums} = useQuery({queryKey: ['enums'], queryFn: () => fetchEnums()});
  const matchTypeKeys = Object.keys(enums?.match_type || {});
  const actionTypeKeys = Object.keys(enums?.action_type || {});
  const DEFAULT_MATCH_TYPE = matchTypeKeys.find(k => k === 'domain') || 'domain';
  const DEFAULT_ACTION_TYPE = actionTypeKeys.find(k => k === 'chain') || 'chain';
  const form = useForm<RouteRuleFormValues>({
    defaultValues: {...defaultRouteRuleFormValues(), matchType: DEFAULT_MATCH_TYPE, actionType: DEFAULT_ACTION_TYPE}
  });
  const groupForm = useForm<RouteRuleGroupFormValues>({
    defaultValues: defaultRouteRuleGroupFormValues()
  });
  const actionType = form.watch('actionType');
  const matchType = form.watch('matchType');
  const matchTypeOptions = enums?.match_type ? Object.entries(enums.match_type).map(([value, item]) => ({value, label: item.name})) : [];
  const actionTypeOptions = enums?.action_type ? Object.entries(enums.action_type).map(([value, item]) => ({value, label: item.name})) : [];
  const selectedChainId = form.watch('chainId');
  const [regexTesterOpen, setRegexTesterOpen] = useState(false);
  const [selectedGroupId, setSelectedGroupId] = useState('');
  const [editingGroupId, setEditingGroupId] = useState('');
  const [groupModalOpen, setGroupModalOpen] = useState(false);
  const [grantGroup, setGrantGroup] = useState<RouteRuleGroup | null>(null);
  const [deletingGroup, setDeletingGroup] = useState<RouteRuleGroup | null>(null);
  const [groupDeleteImpact, setGroupDeleteImpact] = useState<RouteRuleGroupDeleteImpact | null>(null);
  const [editingRuleId, setEditingRuleId] = useState('');
  const [deletingRule, setDeletingRule] = useState<RouteRule | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [matchFilter, setMatchFilter] = useState('');
  const [chainFilter, setChainFilter] = useState('');
  const [scopeFilter, setScopeFilter] = useState('');
  const [actionFilter, setActionFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const formValues = form.watch();
  const {validationPending, validationResult} = useRouteRuleValidation({accessToken, activeTenantId, editingRuleId, formValues});

  const routeGroupsQuery = useQuery({
    queryKey: ['route-rule-groups', accessToken, activeTenantId],
    queryFn: () => getRouteRuleGroups(accessToken, activeTenantId),
    enabled: !!accessToken
  });
  const routeRulesQuery = useQuery({
    queryKey: ['route-rules', accessToken, activeTenantId],
    queryFn: () => getRouteRules(accessToken, activeTenantId),
    enabled: !!accessToken
  });
  const chainsQuery = useQuery({
    queryKey: ['proxy-chains', accessToken, activeTenantId],
    queryFn: () => getChains(accessToken, activeTenantId),
    enabled: !!accessToken
  });
  const scopesQuery = useQuery({
    queryKey: ['proxy-scopes', accessToken, activeTenantId],
    queryFn: () => getScopes(accessToken, activeTenantId),
    enabled: !!accessToken
  });
  const policyRevisionsQuery = useQuery({
    queryKey: ['policy-revisions', accessToken, activeTenantId],
    queryFn: () => getPolicyRevisions(accessToken, activeTenantId),
    enabled: !!accessToken
  });
  const createGroupMutation = useMutation({
    mutationFn: (payload: RouteRuleGroupFormValues) => createRouteRuleGroup(accessToken, activeTenantId, {
      name: payload.name.trim(),
      description: payload.description.trim()
    }),
    onSuccess: (group) => {
      toast.success(routesT('createGroupSuccess'));
      queryClient.invalidateQueries({queryKey: ['route-rule-groups']});
      setSelectedGroupId(group.id);
      setGroupModalOpen(false);
      groupForm.reset(defaultRouteRuleGroupFormValues());
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });
  const updateGroupMutation = useMutation({
    mutationFn: (payload: RouteRuleGroupFormValues & {id: string}) => updateRouteRuleGroup(accessToken, activeTenantId, payload.id, {
      name: payload.name.trim(),
      description: payload.description.trim(),
      enabled: payload.enabled
    }),
    onSuccess: () => {
      toast.success(routesT('updateGroupSuccess'));
      queryClient.invalidateQueries({queryKey: ['route-rule-groups']});
      queryClient.invalidateQueries({queryKey: ['route-rules']});
      setEditingGroupId('');
      setGroupModalOpen(false);
      groupForm.reset(defaultRouteRuleGroupFormValues());
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });
  const groupDeleteImpactMutation = useMutation({
    mutationFn: (groupId: string) => getRouteRuleGroupDeleteImpact(accessToken, activeTenantId, groupId),
    onSuccess: (result) => setGroupDeleteImpact(result),
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
      setDeletingGroup(null);
    }
  });
  const deleteGroupMutation = useMutation({
    mutationFn: (groupId: string) => deleteRouteRuleGroup(accessToken, activeTenantId, groupId),
    onSuccess: () => {
      toast.success(routesT('deleteGroupSuccess'));
      queryClient.invalidateQueries({queryKey: ['route-rule-groups']});
      queryClient.invalidateQueries({queryKey: ['route-rules']});
      setDeletingGroup(null);
      setGroupDeleteImpact(null);
      setSelectedGroupId('');
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });
  const createRuleMutation = useMutation({
    mutationFn: (payload: RouteRuleSubmitPayload) => createRouteRule(accessToken, activeTenantId, payload),
    onSuccess: () => {
      toast.success(routesT('createSuccess'));
      queryClient.invalidateQueries({queryKey: ['route-rules']});
      queryClient.invalidateQueries({queryKey: ['route-rule-groups']});
      setCreateOpen(false);
      form.reset(defaultRouteRuleFormValues());
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });
  const updateRuleMutation = useMutation({
    mutationFn: (payload: RouteRuleSubmitPayload & {id: string}) => updateRouteRule(accessToken, activeTenantId, payload.id, {
      groupId: payload.groupId,
      priority: payload.priority,
      matchType: payload.matchType,
      matchValue: payload.matchValue,
      actionType: payload.actionType,
      chainId: payload.chainId,
      destinationScope: payload.destinationScope,
      enabled: payload.enabled
    }),
    onSuccess: () => {
      toast.success(routesT('updateSuccess'));
      queryClient.invalidateQueries({queryKey: ['route-rules']});
      queryClient.invalidateQueries({queryKey: ['route-rule-groups']});
      setEditingRuleId('');
      form.reset(defaultRouteRuleFormValues());
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });
  const deleteRuleMutation = useMutation({
    mutationFn: (ruleId: string) => deleteRouteRule(accessToken, activeTenantId, ruleId),
    onSuccess: () => {
      toast.success(routesT('deleteSuccess'));
      queryClient.invalidateQueries({queryKey: ['route-rules']});
      queryClient.invalidateQueries({queryKey: ['route-rule-groups']});
      if (editingRuleId) {
        setEditingRuleId('');
        form.reset(defaultRouteRuleFormValues());
      }
      setDeletingRule(null);
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });
  const publishMutation = useMutation({
    mutationFn: () => publishPolicy(accessToken, activeTenantId),
    onSuccess: () => {
      toast.success(routesT('publishSuccess'));
      queryClient.invalidateQueries({queryKey: ['policy-revisions']});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const routeGroups = routeGroupsQuery.data || [];
  const manageableGroups = useMemo(() => routeGroups.filter((group) => globalSuperAdmin || group.permission === 'manage'), [globalSuperAdmin, routeGroups]);
  const selectedGroup = routeGroups.find((group) => group.id === selectedGroupId) || null;
  const selectedGroupCanManage = Boolean(selectedGroup && (globalSuperAdmin || selectedGroup.permission === 'manage'));
  const routeRules = routeRulesQuery.data || [];
  const policyRevisions = policyRevisionsQuery.data || [];
  const chains = chainsQuery.data || [];
  const scopes = scopesQuery.data || [];
  const filteredRouteRules = useMemo(() => {
    return routeRules.filter((rule) => {
      if (selectedGroupId && rule.groupId !== selectedGroupId) {
        return false;
      }
      if (actionFilter && rule.actionType !== actionFilter) {
        return false;
      }
      if (chainFilter && rule.chainId !== chainFilter) {
        return false;
      }
      if (scopeFilter && rule.destinationScope !== scopeFilter) {
        return false;
      }
      if (statusFilter === 'enabled' && !rule.enabled) {
        return false;
      }
      if (statusFilter === 'disabled' && rule.enabled) {
        return false;
      }
      if (!matchFilter.trim()) {
        return true;
      }
      const keyword = matchFilter.trim().toLowerCase();
      return [rule.matchType, rule.matchValue]
        .some((value) => String(value || '').toLowerCase().includes(keyword));
    });
  }, [actionFilter, chainFilter, matchFilter, routeRules, scopeFilter, selectedGroupId, statusFilter]);

  const selectedChain = chains.find((c) => c.id === selectedChainId);
  const matchValuePlaceholder = matchType === 'default' ? '*' : routesT('matchValuePlaceholder', {type: matchType || routesT('value')});
  const editingRule = routeRules.find((rule) => rule.id === editingRuleId);

  const resetForm = useCallback(() => {
    setEditingRuleId('');
    setCreateOpen(false);
    form.reset({...defaultRouteRuleFormValues(), groupId: selectedGroupId});
  }, [form, selectedGroupId]);

  const startEdit = useCallback((rule: RouteRule) => {
    setCreateOpen(false);
    setEditingRuleId(rule.id);
    form.reset(routeRuleFormValues(rule));
  }, [form]);

  const startCreate = useCallback(() => {
    setEditingRuleId('');
    setCreateOpen(true);
    form.reset({...defaultRouteRuleFormValues(), groupId: selectedGroupId});
  }, [form, selectedGroupId]);

  const submitRouteRule = useCallback((values: RouteRuleFormValues) => {
    const payload = routeRuleSubmitPayload(values, chains);
    if (editingRuleId) {
      updateRuleMutation.mutate({id: editingRuleId, ...payload});
    } else {
      createRuleMutation.mutate(payload);
    }
  }, [chains, createRuleMutation, editingRuleId, updateRuleMutation]);

  const deleteRoute = useCallback((ruleId: string) => {
    setDeletingRule(routeRules.find((rule) => rule.id === ruleId) || null);
  }, [routeRules]);

  const startCreateGroup = useCallback(() => {
    setEditingGroupId('');
    setGroupModalOpen(true);
    groupForm.reset(defaultRouteRuleGroupFormValues());
  }, [groupForm]);

  const startEditGroup = useCallback((group: RouteRuleGroup) => {
    setEditingGroupId(group.id);
    setGroupModalOpen(true);
    groupForm.reset({name: group.name, description: group.description || '', enabled: group.enabled});
  }, [groupForm]);

  const resetGroupForm = useCallback(() => {
    setEditingGroupId('');
    setGroupModalOpen(false);
    groupForm.reset(defaultRouteRuleGroupFormValues());
  }, [groupForm]);

  const submitRouteRuleGroup = useCallback((values: RouteRuleGroupFormValues) => {
    if (editingGroupId) {
      updateGroupMutation.mutate({id: editingGroupId, ...values});
      return;
    }
    createGroupMutation.mutate(values);
  }, [createGroupMutation, editingGroupId, updateGroupMutation]);

  const openDeleteGroup = useCallback((group: RouteRuleGroup) => {
    setDeletingGroup(group);
    setGroupDeleteImpact(null);
    groupDeleteImpactMutation.mutate(group.id);
  }, [groupDeleteImpactMutation]);

  useEffect(() => {
    if (actionType !== 'chain') {
      return;
    }
    const nextScope = selectedChain?.destinationScope || '';
    if (form.getValues('destinationScope') !== nextScope) {
      form.setValue('destinationScope', nextScope, {shouldDirty: false, shouldValidate: false});
    }
  }, [actionType, form, selectedChain?.destinationScope]);
  useEffect(() => {
    if (routeGroups.length === 0) {
      if (selectedGroupId !== '') {
        setSelectedGroupId('');
      }
      return;
    }
    if (!routeGroups.some((group) => group.id === selectedGroupId)) {
      setSelectedGroupId(routeGroups[0].id);
    }
  }, [routeGroups, selectedGroupId]);
  const modalOpen = createOpen || Boolean(editingRule);
  const routeDeleteSections = routeRuleDeleteSections(deletingRule, routesT);
  const groupDeleteSections = routeRuleGroupDeleteSections(deletingGroup, groupDeleteImpact, routesT);

  return (
    <AuthGate>
      <ConsolePage
        actions={canWrite ? (
          <RoutePageActions
            createRuleDisabled={!selectedGroupCanManage}
            onCreateGroup={startCreateGroup}
            onCreateRule={startCreate}
            onPublish={() => publishMutation.mutate()}
            publishPending={publishMutation.isPending}
            routesT={routesT}
            t={t}
          />
        ) : null}
        title={t('shell.routeBoard')}
      >
        <RouteRuleFilterBar
          actionFilter={actionFilter}
          actionTypeOptions={actionTypeOptions}
          chainFilter={chainFilter}
          chains={chains}
          matchFilter={matchFilter}
          onActionFilterChange={setActionFilter}
          onChainFilterChange={setChainFilter}
          onMatchFilterChange={setMatchFilter}
          onScopeFilterChange={setScopeFilter}
          onStatusFilterChange={setStatusFilter}
          routesT={routesT}
          scopeFilter={scopeFilter}
          scopes={scopes}
          statusFilter={statusFilter}
          t={t}
        />

        <ConsoleList count={routeGroups.length} title={routesT('routeGroups')}>
          <RouteRuleGroupTable
            deletePending={deleteGroupMutation.isPending || groupDeleteImpactMutation.isPending}
            globalSuperAdmin={globalSuperAdmin}
            groups={routeGroups}
            groupsQuery={routeGroupsQuery}
            onDelete={canWrite ? openDeleteGroup : undefined}
            onEdit={canWrite ? startEditGroup : undefined}
            onGrant={canWrite ? setGrantGroup : undefined}
            onSelect={setSelectedGroupId}
            routesT={routesT}
            selectedGroupId={selectedGroupId}
            t={t}
          />
        </ConsoleList>

        <ConsoleList count={filteredRouteRules.length} title={selectedGroup ? `${routesT('routeRules')} - ${selectedGroup.name}` : routesT('routeRules')}>
          <RouteRuleTable
            chains={chains}
            deletePending={deleteRuleMutation.isPending}
            globalSuperAdmin={globalSuperAdmin}
            onDelete={canWrite ? deleteRoute : undefined}
            onEdit={canWrite ? startEdit : undefined}
            routeRules={filteredRouteRules}
            routeRulesQuery={routeRulesQuery}
            routesT={routesT}
            scopes={scopes}
            t={t}
          />
        </ConsoleList>

        <RoutePolicyHistory query={policyRevisionsQuery} revisions={policyRevisions} routesT={routesT} t={t} />

        {canWrite ? (
          <RouteGroupEditorModal
            editing={Boolean(editingGroupId)}
            form={groupForm}
            onClose={resetGroupForm}
            onSubmit={submitRouteRuleGroup}
            open={groupModalOpen}
            pending={createGroupMutation.isPending || updateGroupMutation.isPending}
            routesT={routesT}
            t={t}
          />
        ) : null}

        {canWrite ? (
          <RouteRuleEditorModal
            actionType={actionType}
            actionTypeOptions={actionTypeOptions}
            chains={chains}
            createPending={createRuleMutation.isPending}
            editingRule={!!editingRule}
            form={form}
            groups={manageableGroups}
            matchType={matchType}
            matchTypeOptions={matchTypeOptions}
            matchValuePlaceholder={matchValuePlaceholder}
            onClose={resetForm}
            onOpenRegexTester={() => setRegexTesterOpen(true)}
            onSubmit={submitRouteRule}
            open={modalOpen}
            routesT={routesT}
            scopes={scopes}
            selectedChain={selectedChain}
            t={t}
            updatePending={updateRuleMutation.isPending}
            validateMatchValue={(type, value) => validateMatchValue(type, value, routesT)}
            validationPending={validationPending}
            validationResult={validationResult}
          />
        ) : null}

        {regexTesterOpen && (
          <RegexTesterModal initialPattern={form.getValues('matchValue')} onClose={() => setRegexTesterOpen(false)} />
        )}

        {grantGroup ? (
          <ResourceGrantModal
            onChanged={() => {
              queryClient.invalidateQueries({queryKey: ['route-rule-groups']});
              queryClient.invalidateQueries({queryKey: ['route-rules']});
            }}
            onClose={() => setGrantGroup(null)}
            open={Boolean(grantGroup)}
            resourceId={grantGroup.id}
            resourceName={grantGroup.name}
            resourceType="route_rule_group"
          />
        ) : null}

        <RouteDeleteDialogs
          deletingGroup={deletingGroup}
          deletingRule={deletingRule}
          groupDeleteSections={groupDeleteSections}
          groupPending={deleteGroupMutation.isPending || groupDeleteImpactMutation.isPending}
          onCloseGroup={() => {
            setDeletingGroup(null);
            setGroupDeleteImpact(null);
          }}
          onCloseRule={() => setDeletingRule(null)}
          onConfirmGroup={() => {
            if (deletingGroup) {
              deleteGroupMutation.mutate(deletingGroup.id);
            }
          }}
          onConfirmRule={() => {
            if (deletingRule) {
              deleteRuleMutation.mutate(deletingRule.id);
            }
          }}
          routeDeleteSections={routeDeleteSections}
          routesT={routesT}
          rulePending={deleteRuleMutation.isPending}
        />
      </ConsolePage>
    </AuthGate>
  );
}
