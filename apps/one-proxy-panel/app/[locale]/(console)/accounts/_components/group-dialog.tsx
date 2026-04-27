'use client';

import {useMutation, useQuery} from '@tanstack/react-query';
import {useEffect, useState} from 'react';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';

import {
  createGroup,
  getAccounts,
  getGroup,
  getNodes,
  setGroupAccounts,
  setGroupScopes,
  updateGroup
} from '@/lib/api';
import {formatControlPlaneError} from '@/lib/presentation';

type GroupFields = {
  name: string;
  description: string;
  enabled: boolean;
};

type Props = {
  open: boolean;
  onClose: () => void;
  onSaved: () => void;
  accessToken: string;
  group?: {id: string; name: string; description: string; enabled: boolean} | null;
};

export default function GroupDialog({open, onClose, onSaved, accessToken, group}: Props) {
  const t = useTranslations();
  const isEdit = !!group;

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [enabled, setEnabled] = useState(true);
  const [selectedAccounts, setSelectedAccounts] = useState<string[]>([]);
  const [selectedScopes, setSelectedScopes] = useState<string[]>([]);

  // Fetch existing group detail when editing (for accounts & scopes)
  const groupDetailQuery = useQuery({
    queryKey: ['group', group?.id, accessToken],
    queryFn: () => getGroup(accessToken, group!.id),
    enabled: isEdit && open && !!accessToken
  });

  // Fetch accounts for member selector
  const accountsQuery = useQuery({
    queryKey: ['accounts', accessToken],
    queryFn: () => getAccounts(accessToken),
    enabled: open && !!accessToken
  });

  // Fetch nodes to extract scope keys
  const nodesQuery = useQuery({
    queryKey: ['nodes', accessToken],
    queryFn: () => getNodes(accessToken),
    enabled: open && !!accessToken
  });

  const scopeKeys = [...new Set((nodesQuery.data || []).map((n) => n.scopeKey).filter(Boolean))].sort();

  useEffect(() => {
    if (open) {
      setName(group?.name || '');
      setDescription(group?.description || '');
      setEnabled(group?.enabled ?? true);

      if (groupDetailQuery.data) {
        setSelectedAccounts(groupDetailQuery.data.accounts.map((a) => a.id));
        setSelectedScopes(groupDetailQuery.data.scopes);
      } else if (!isEdit) {
        setSelectedAccounts([]);
        setSelectedScopes([]);
      }
    }
  }, [open, group, groupDetailQuery.data, isEdit]);

  const createMutation = useMutation({
    mutationFn: async (payload: GroupFields) => {
      const created = await createGroup(accessToken, payload);
      await setGroupAccounts(accessToken, created.id, selectedAccounts);
      await setGroupScopes(accessToken, created.id, selectedScopes);
    },
    onSuccess: () => {
      toast.success(t('shell.groupCreateSuccess'));
      onSaved();
      onClose();
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const updateMutation = useMutation({
    mutationFn: async (payload: GroupFields) => {
      await updateGroup(accessToken, group!.id, payload);
      await setGroupAccounts(accessToken, group!.id, selectedAccounts);
      await setGroupScopes(accessToken, group!.id, selectedScopes);
    },
    onSuccess: () => {
      toast.success(t('shell.groupUpdateSuccess'));
      onSaved();
      onClose();
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const isPending = createMutation.isPending || updateMutation.isPending;

  const handleSave = () => {
    const payload: GroupFields = {
      name: name.trim(),
      description: description.trim(),
      enabled
    };
    if (!payload.name) {
      toast.error('Name is required');
      return;
    }
    if (isEdit) {
      updateMutation.mutate(payload);
    } else {
      createMutation.mutate(payload);
    }
  };

  const toggleAccount = (accountId: string) => {
    setSelectedAccounts((prev) =>
      prev.includes(accountId) ? prev.filter((id) => id !== accountId) : [...prev, accountId]
    );
  };

  const toggleScope = (scopeKey: string) => {
    setSelectedScopes((prev) =>
      prev.includes(scopeKey) ? prev.filter((k) => k !== scopeKey) : [...prev, scopeKey]
    );
  };

  if (!open) return null;

  const accounts = accountsQuery.data || [];

  return (
    <div className="dialog-backdrop" onClick={onClose}>
      <div className="dialog-panel" onClick={(e) => e.stopPropagation()} style={{maxWidth: 560, maxHeight: '90vh', overflowY: 'auto'}}>
        <h3>{isEdit ? `Edit group: ${group!.name}` : t('shell.groupCreate')}</h3>
        <div className="sub-grid">
          <div className="field-stack">
            <span>{t('shell.groupName')}</span>
            <input
              className="field-input"
              placeholder="e.g. Dev Team"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <div className="field-stack">
            <span>{t('shell.groupDescription')}</span>
            <textarea
              className="field-textarea"
              placeholder="Optional description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>
          <div className="field-stack">
            <label className="inline-cluster" style={{justifyContent: 'flex-start'}}>
              <input
                type="checkbox"
                checked={enabled}
                onChange={(e) => setEnabled(e.target.checked)}
              />
              <span>{t('shell.groupEnabled')}</span>
            </label>
          </div>

          <div className="field-stack">
            <span>{t('shell.groupAccounts')}</span>
            {accountsQuery.isPending ? (
              <p className="muted-text">{t('common.loading')}</p>
            ) : accounts.length === 0 ? (
              <p className="muted-text">{t('common.empty')}</p>
            ) : (
              <div className="check-list">
                {accounts.map((account) => (
                  <label className="inline-cluster" key={account.id} style={{justifyContent: 'flex-start'}}>
                    <input
                      type="checkbox"
                      checked={selectedAccounts.includes(account.id)}
                      onChange={() => toggleAccount(account.id)}
                    />
                    <span>{account.account}</span>
                    <span className="badge is-neutral">{account.role}</span>
                  </label>
                ))}
              </div>
            )}
          </div>

          <div className="field-stack">
            <span>{t('shell.groupScopes')}</span>
            {nodesQuery.isPending ? (
              <p className="muted-text">{t('common.loading')}</p>
            ) : scopeKeys.length === 0 ? (
              <p className="muted-text">{t('common.empty')}</p>
            ) : (
              <div className="check-list">
                {scopeKeys.map((scopeKey) => (
                  <label className="inline-cluster" key={scopeKey} style={{justifyContent: 'flex-start'}}>
                    <input
                      type="checkbox"
                      checked={selectedScopes.includes(scopeKey)}
                      onChange={() => toggleScope(scopeKey)}
                    />
                    <span className="mono">{scopeKey}</span>
                  </label>
                ))}
              </div>
            )}
          </div>

          <div className="submit-row">
            <button className="secondary-button" onClick={onClose} type="button">
              Cancel
            </button>
            <button className="primary-button" disabled={isPending} onClick={handleSave} type="button">
              {isPending ? t('common.submitting') : isEdit ? 'Save' : t('shell.groupCreate')}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
