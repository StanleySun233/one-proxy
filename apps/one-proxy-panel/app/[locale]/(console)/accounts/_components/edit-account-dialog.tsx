'use client';

import {useMutation, useQuery} from '@tanstack/react-query';
import {useState} from 'react';
import {toast} from 'sonner';

import {fetchEnums, updateAccount} from '@/lib/control-plane-api';
import {formatControlPlaneError} from '@/lib/presentation';

type Props = {
  open: boolean;
  onClose: () => void;
  account: {id: string; account: string; role: string; status: string};
  onSaved: () => void;
  accessToken: string;
};

export default function EditAccountDialog({open, onClose, account, onSaved, accessToken}: Props) {
  const [password, setPassword] = useState('');
  const [role, setRole] = useState(account.role);
  const [status, setStatus] = useState(account.status);
  const {data: enums} = useQuery({queryKey: ['enums'], queryFn: () => fetchEnums()});
  const accountStatusOptions = enums?.account_status ? Object.entries(enums.account_status).map(([value, item]) => ({value, label: item.name})) : [];

  const updateMutation = useMutation({
    mutationFn: (payload: {password?: string; role?: string; status?: string}) =>
      updateAccount(accessToken, account.id, payload),
    onSuccess: () => {
      toast.success('account updated');
      onSaved();
      onClose();
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const handleSave = () => {
    const payload: {password?: string; role?: string; status?: string} = {};
    if (password) payload.password = password;
    payload.role = role;
    payload.status = status;
    updateMutation.mutate(payload);
  };

  if (!open) return null;

  return (
    <div className="dialog-backdrop" onClick={onClose}>
      <div className="dialog-panel" onClick={(e) => e.stopPropagation()}>
        <h3>Edit account: {account.account}</h3>
        <div className="sub-grid">
          <div className="field-stack">
            <span>Password</span>
            <input
              className="field-input"
              type="password"
              placeholder="Leave blank to keep current password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>
          <div className="field-stack">
            <span>Role</span>
            <input
              className="field-input"
              value={role}
              onChange={(e) => setRole(e.target.value)}
            />
          </div>
          <div className="field-stack">
            <span>Status</span>
            <select className="field-input" value={status} onChange={(e) => setStatus(e.target.value)}>
              {accountStatusOptions.map((opt) => (
                <option key={opt.value} value={opt.value}>{opt.label}</option>
              ))}
            </select>
          </div>
          <div className="submit-row">
            <button className="secondary-button" onClick={onClose} type="button">Cancel</button>
            <button className="primary-button" disabled={updateMutation.isPending} onClick={handleSave} type="button">
              {updateMutation.isPending ? 'Saving…' : 'Save'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
