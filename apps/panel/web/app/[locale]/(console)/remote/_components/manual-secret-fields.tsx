import type {RemoteProtocol} from '@/lib/types';

type ManualSecretFieldsProps = {
  protocol: RemoteProtocol;
  password: string;
  privateKey: string;
  passphrase: string;
  setPassword: (value: string) => void;
  setPrivateKey: (value: string) => void;
  setPassphrase: (value: string) => void;
  t: (key: string) => string;
};

export function ManualSecretFields({protocol, password, privateKey, passphrase, setPassword, setPrivateKey, setPassphrase, t}: ManualSecretFieldsProps) {
  return (
    <>
      <label className="field-stack">
        <span>{t('password')}</span>
        <input className="field-input" onChange={(event) => setPassword(event.target.value)} type="password" value={password} />
      </label>
      {protocol === 'ssh' ? (
        <>
          <label className="field-stack remote-wide-field">
            <span>{t('privateKey')}</span>
            <textarea className="field-textarea" onChange={(event) => setPrivateKey(event.target.value)} value={privateKey} />
          </label>
          <label className="field-stack">
            <span>{t('passphrase')}</span>
            <input className="field-input" onChange={(event) => setPassphrase(event.target.value)} type="password" value={passphrase} />
          </label>
        </>
      ) : null}
    </>
  );
}
