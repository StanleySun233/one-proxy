'use client';

import {useQuery, useQueryClient} from '@tanstack/react-query';
import {KeyRound, Monitor, PlugZap, RefreshCw, Save, Terminal, Trash2, Unplug} from 'lucide-react';
import {useTranslations} from 'next-intl';
import {useSearchParams} from 'next/navigation';
import {useCallback, useEffect, useMemo, useRef, useState} from 'react';

import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {ConsolePage} from '@/components/console-template';
import {createRemoteCredential, createRemoteSession, deleteRemoteCredential, getNodeAccessPaths, getRemoteCredentials} from '@/lib/api';
import {decryptRemoteSecret, encryptRemoteSecret} from '@/lib/remote-vault';
import type {RemoteCredential, RemoteCredentialScope, RemoteProtocol, RemoteSecret, RemoteSession} from '@/lib/types';

import {CredentialModeToggle} from './credential-mode-toggle';
import {ManualSecretFields} from './manual-secret-fields';
import {remoteTCPPaths, statusClass, validSecret, type GuacamoleRuntime, type RemoteStatus} from './remote-helpers';

type RemotePageProps = {
  protocol: RemoteProtocol;
};

export function RemotePage({protocol}: RemotePageProps) {
  const t = useTranslations('remote');
  const shellT = useTranslations('shell');
  const queryClient = useQueryClient();
  const searchParams = useSearchParams();
  const {session, activeTenant} = useAuth();
  const accessToken = session?.accessToken || '';
  const activeTenantId = session?.activeTenantId || null;
  const canSaveTenant = !!activeTenantId && (session?.account.role === 'super_admin' || activeTenant?.role === 'tenant_admin');
  const displayMountRef = useRef<HTMLDivElement | null>(null);
  const displayShellRef = useRef<HTMLDivElement | null>(null);
  const clientRef = useRef<any>(null);
  const resizeObserverRef = useRef<ResizeObserver | null>(null);
  const [accessPathId, setAccessPathId] = useState(searchParams.get('pathId') || '');
  const [credentialMode, setCredentialMode] = useState<'manual' | 'saved'>('manual');
  const [credentialId, setCredentialId] = useState('');
  const [vaultPassphrase, setVaultPassphrase] = useState('');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [privateKey, setPrivateKey] = useState('');
  const [passphrase, setPassphrase] = useState('');
  const [saveName, setSaveName] = useState('');
  const [saveScope, setSaveScope] = useState<RemoteCredentialScope>('personal');
  const [saveVaultPassphrase, setSaveVaultPassphrase] = useState('');
  const [status, setStatus] = useState<RemoteStatus>('idle');
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);

  const accessPathsQuery = useQuery({
    queryKey: ['node-access-paths', accessToken, activeTenantId, 'remote'],
    queryFn: () => getNodeAccessPaths(accessToken, activeTenantId),
    enabled: !!accessToken
  });
  const credentialsQuery = useQuery({
    queryKey: ['remote-credentials', accessToken, activeTenantId, protocol],
    queryFn: () => getRemoteCredentials(accessToken, activeTenantId, protocol),
    enabled: !!accessToken
  });

  const tcpPaths = useMemo(() => remoteTCPPaths(accessPathsQuery.data || []), [accessPathsQuery.data]);
  const selectedCredential = (credentialsQuery.data || []).find((item) => item.id === credentialId) || null;
  const selectedPath = tcpPaths.find((item) => item.id === accessPathId) || null;
  const title = protocol === 'ssh' ? shellT('remoteSSH') : shellT('remoteRDP');
  const SecretIcon = protocol === 'ssh' ? Terminal : Monitor;

  useEffect(() => {
    const queryPathId = searchParams.get('pathId') || '';
    if (queryPathId) {
      setAccessPathId(queryPathId);
    }
  }, [searchParams]);

  useEffect(() => {
    if (!accessPathId && tcpPaths[0]) {
      setAccessPathId(tcpPaths[0].id);
    }
  }, [accessPathId, tcpPaths]);

  useEffect(() => {
    if (credentialMode === 'saved' && selectedCredential) {
      setUsername(selectedCredential.username);
    }
  }, [credentialMode, selectedCredential]);

  useEffect(() => {
    if (!canSaveTenant && saveScope === 'tenant') setSaveScope('personal');
  }, [canSaveTenant, saveScope]);

  const cleanupRemoteClient = useCallback(() => {
    resizeObserverRef.current?.disconnect();
    resizeObserverRef.current = null;
    if (clientRef.current) {
      clientRef.current.disconnect();
      clientRef.current = null;
    }
    if (displayMountRef.current) {
      displayMountRef.current.innerHTML = '';
    }
  }, []);

  useEffect(() => {
    return () => {
      cleanupRemoteClient();
    };
  }, [cleanupRemoteClient]);

  const fitDisplay = useCallback((client: any) => {
    const shell = displayShellRef.current;
    if (!shell || !client) {
      return;
    }
    const display = client.getDisplay();
    const width = display.getWidth();
    const height = display.getHeight();
    if (!width || !height) {
      return;
    }
    const scale = Math.max(0.1, Math.min(shell.clientWidth / width, shell.clientHeight / height, 2));
    display.scale(scale);
  }, []);

  const startGuacamoleClient = useCallback(async (remoteSession: RemoteSession) => {
    const mount = displayMountRef.current;
    const shell = displayShellRef.current;
    if (!mount || !shell) {
      throw new Error('remote_display_unavailable');
    }
    cleanupRemoteClient();
    const module = await import('guacamole-common-js') as unknown as {default?: GuacamoleRuntime} & GuacamoleRuntime;
    const Guacamole = (module.default || module) as GuacamoleRuntime;
    const tunnel = new Guacamole.WebSocketTunnel(remoteSession.tunnelUrl);
    const client = new Guacamole.Client(tunnel);
    clientRef.current = client;
    client.onerror = () => {
      setStatus('failed');
      setError(t('connectionFailed'));
    };
    client.onstatechange = (state: number) => {
      if (state === 3) {
        setStatus('connected');
      } else if (state === 5) {
        setStatus('disconnected');
      }
    };
    client.onsync = () => fitDisplay(client);
    const displayElement = client.getDisplay().getElement() as HTMLElement;
    displayElement.classList.add('remote-guac-display');
    mount.appendChild(displayElement);
    const mouse = new Guacamole.Mouse(displayElement);
    mouse.onmousedown = mouse.onmouseup = mouse.onmousemove = (mouseState: unknown) => client.sendMouseState(mouseState, true);
    const keyboard = new Guacamole.Keyboard(shell);
    keyboard.onkeydown = (keysym: number) => {
      client.sendKeyEvent(1, keysym);
      return false;
    };
    keyboard.onkeyup = (keysym: number) => {
      client.sendKeyEvent(0, keysym);
    };
    resizeObserverRef.current = new ResizeObserver(() => fitDisplay(client));
    resizeObserverRef.current.observe(shell);
    shell.focus();
    client.connect(`token=${encodeURIComponent(remoteSession.token)}`);
    fitDisplay(client);
  }, [cleanupRemoteClient, fitDisplay, t]);

  const resolveSecret = async (): Promise<RemoteSecret> => {
    if (credentialMode === 'saved') {
      if (!selectedCredential) {
        throw new Error(t('selectCredential'));
      }
      if (!vaultPassphrase) {
        throw new Error(t('vaultPassphraseRequired'));
      }
      return decryptRemoteSecret(selectedCredential.encryptedPayload, vaultPassphrase);
    }
    return {password, privateKey, passphrase};
  };

  const handleConnect = async () => {
    setError('');
    if (!activeTenantId) {
      setError(t('tenantRequired'));
      return;
    }
    if (!selectedPath || !username) {
      setError(t('missingConnectionInput'));
      return;
    }
    setStatus('connecting');
    try {
      const secret = await resolveSecret();
      if (!validSecret(protocol, secret)) {
        throw new Error(t('missingSecret'));
      }
      const rect = displayShellRef.current?.getBoundingClientRect();
      const remoteSession = await createRemoteSession(accessToken, activeTenantId, {
        accessPathId,
        credentialId: credentialMode === 'saved' ? credentialId : undefined,
        protocol,
        username,
        password: secret.password,
        privateKey: secret.privateKey,
        passphrase: secret.passphrase,
        width: Math.max(1024, Math.floor(rect?.width || 1280)),
        height: Math.max(640, Math.floor(rect?.height || 800)),
        dpi: 96
      });
      await startGuacamoleClient(remoteSession);
    } catch (caught) {
      setStatus('failed');
      setError(caught instanceof Error ? caught.message : t('connectionFailed'));
    }
  };

  const handleDisconnect = () => {
    cleanupRemoteClient();
    setStatus('disconnected');
  };

  const handleSaveCredential = async () => {
    setError('');
    if (!saveName || !username || !saveVaultPassphrase) {
      setError(t('missingSaveInput'));
      return;
    }
    const secret = {password, privateKey, passphrase};
    if (!validSecret(protocol, secret)) {
      setError(t('missingSecret'));
      return;
    }
    setSaving(true);
    try {
      const encryptedPayload = await encryptRemoteSecret(secret, saveVaultPassphrase);
      await createRemoteCredential(accessToken, activeTenantId, {
        name: saveName,
        protocol,
        scope: saveScope,
        username,
        secretType: secret.privateKey ? 'private_key' : 'password',
        encryptedPayload
      });
      setSaveName('');
      setSaveVaultPassphrase('');
      await queryClient.invalidateQueries({queryKey: ['remote-credentials', accessToken, activeTenantId, protocol]});
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : t('saveFailed'));
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteCredential = async (credential: RemoteCredential) => {
    setError('');
    await deleteRemoteCredential(accessToken, activeTenantId, credential.id);
    if (credential.id === credentialId) {
      setCredentialId('');
      setCredentialMode('manual');
    }
    await queryClient.invalidateQueries({queryKey: ['remote-credentials', accessToken, activeTenantId, protocol]});
  };

  return (
    <AuthGate>
      <ConsolePage title={title}>
        <div className="remote-layout">
          <section className="remote-control-panel">
            <div className="remote-panel-head">
              <SecretIcon size={18} />
              <h4>{t('connection')}</h4>
            </div>
            <div className="remote-form-grid">
              <label className="field-stack">
                <span>{t('accessPath')}</span>
                <select className="field-select" onChange={(event) => setAccessPathId(event.target.value)} value={accessPathId}>
                  {tcpPaths.map((path) => (
                    <option key={path.id} value={path.id}>{path.name}</option>
                  ))}
                </select>
              </label>
              <label className="field-stack">
                <span>{t('credentialMode')}</span>
                <CredentialModeToggle manualLabel={t('manual')} mode={credentialMode} onChange={setCredentialMode} savedLabel={t('saved')} />
              </label>
              {credentialMode === 'saved' ? (
                <>
                  <label className="field-stack">
                    <span>{t('credential')}</span>
                    <select className="field-select" onChange={(event) => setCredentialId(event.target.value)} value={credentialId}>
                      <option value="">{t('selectCredential')}</option>
                      {(credentialsQuery.data || []).map((credential) => (
                        <option key={credential.id} value={credential.id}>{credential.name}</option>
                      ))}
                    </select>
                  </label>
                  <label className="field-stack">
                    <span>{t('vaultPassphrase')}</span>
                    <input className="field-input" onChange={(event) => setVaultPassphrase(event.target.value)} type="password" value={vaultPassphrase} />
                  </label>
                </>
              ) : null}
              <label className="field-stack">
                <span>{t('username')}</span>
                <input className="field-input" onChange={(event) => setUsername(event.target.value)} value={username} />
              </label>
              {credentialMode === 'manual' ? (
                <ManualSecretFields passphrase={passphrase} password={password} privateKey={privateKey} protocol={protocol} setPassphrase={setPassphrase} setPassword={setPassword} setPrivateKey={setPrivateKey} t={t} />
              ) : null}
            </div>
            <div className="remote-actions">
              <button className="primary-button" disabled={status === 'connecting' || tcpPaths.length === 0} onClick={handleConnect} type="button">
                <PlugZap size={16} />
                {t('connect')}
              </button>
              <button className="secondary-button" disabled={!clientRef.current} onClick={handleDisconnect} type="button">
                <Unplug size={16} />
                {t('disconnect')}
              </button>
              <button className="ghost-button" onClick={() => {
                accessPathsQuery.refetch();
                credentialsQuery.refetch();
              }} type="button">
                <RefreshCw size={16} />
                {t('refresh')}
              </button>
            </div>
            {error ? <p className="error-text">{error}</p> : null}
            <div className="remote-path-summary">
              <span>{selectedPath ? `${selectedPath.targetHost}:${selectedPath.targetPort}` : t('noPath')}</span>
              <span className={`conn-status ${statusClass[status]}`}><span className="conn-status-dot" />{t(`status.${status}`)}</span>
            </div>
          </section>

          <section className="remote-screen-panel">
            <div className="remote-screen-head">
              <span>{title}</span>
              <span>{selectedPath?.name || t('noPath')}</span>
            </div>
            <div className="remote-screen" ref={displayShellRef} tabIndex={0}>
              <div className="remote-screen-empty">{status === 'connected' || status === 'connecting' ? '' : t('screenIdle')}</div>
              <div className="remote-display-mount" ref={displayMountRef} />
            </div>
          </section>
        </div>

        <section className="remote-vault-panel">
          <div className="remote-panel-head">
            <KeyRound size={18} />
            <h4>{t('vault')}</h4>
          </div>
          <div className="remote-save-grid">
            <label className="field-stack">
              <span>{t('saveName')}</span>
              <input className="field-input" onChange={(event) => setSaveName(event.target.value)} value={saveName} />
            </label>
            <label className="field-stack">
              <span>{t('scope')}</span>
              <select className="field-select" onChange={(event) => setSaveScope(event.target.value as RemoteCredentialScope)} value={saveScope}>
                <option value="personal">{t('personal')}</option>
                {canSaveTenant ? <option value="tenant">{t('tenant')}</option> : null}
              </select>
            </label>
            <label className="field-stack">
              <span>{t('newVaultPassphrase')}</span>
              <input className="field-input" onChange={(event) => setSaveVaultPassphrase(event.target.value)} type="password" value={saveVaultPassphrase} />
            </label>
            <button className="secondary-button" disabled={saving || credentialMode !== 'manual'} onClick={handleSaveCredential} type="button">
              <Save size={16} />
              {t('save')}
            </button>
          </div>
          <div className="remote-credential-list">
            {(credentialsQuery.data || []).map((credential) => (
              <div className="remote-credential-row" key={credential.id}>
                <button className="ghost-button remote-credential-main" onClick={() => {
                  setCredentialMode('saved');
                  setCredentialId(credential.id);
                }} type="button">
                  <KeyRound size={15} />
                  <span>{credential.name}</span>
                  <small>{credential.username}</small>
                  <small>{t(credential.scope)}</small>
                </button>
                <button aria-label={t('deleteCredential')} className="ghost-button remote-icon-button" onClick={() => handleDeleteCredential(credential)} type="button">
                  <Trash2 size={15} />
                </button>
              </div>
            ))}
            {credentialsQuery.data?.length === 0 ? <div className="remote-empty-row">{t('noCredentials')}</div> : null}
          </div>
        </section>
      </ConsolePage>
    </AuthGate>
  );
}
