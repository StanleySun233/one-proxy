'use client';

import {Copy, X} from 'lucide-react';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';

import {CompiledChainConfig} from '@/lib/types';

type CompilationPreviewModalProps = {
  config: CompiledChainConfig;
  onClose: () => void;
};

export function CompilationPreviewModal({config, onClose}: CompilationPreviewModalProps) {
  const t = useTranslations();
  const chainsT = useTranslations('proxyChains');
  const handleCopy = () => {
    navigator.clipboard.writeText(JSON.stringify(config, null, 2)).then(
      () => toast.success(chainsT('copiedConfig')),
      () => toast.error(chainsT('copyConfigFailed'))
    );
  };

  return (
    <div className="dialog-backdrop" onClick={onClose}>
      <div className="dialog-panel" onClick={(e) => e.stopPropagation()} style={{maxWidth: 640}}>
        <div className="panel-toolbar">
          <h3>{chainsT('compilationPreview')}</h3>
          <button className="secondary-button" onClick={onClose} type="button">
            <X size={16} />
          </button>
        </div>

        <div className="field-stack">
          <span>{chainsT('routingPath')}</span>
          <div className="token-box">
            <div className="mono">{config.routingPath}</div>
          </div>
        </div>

        <div className="field-stack">
          <span>{chainsT('compiledConfig')}</span>
          <div style={{position: 'relative'}}>
            <pre className="command-block" style={{maxHeight: 320, overflow: 'auto', fontSize: 13}}>
              {JSON.stringify(config, null, 2)}
            </pre>
            <button
              className="secondary-button"
              onClick={handleCopy}
              style={{position: 'absolute', top: 8, right: 8}}
              type="button"
            >
              <Copy size={14} />
              {t('common.copy')}
            </button>
          </div>
        </div>

        <div className="submit-row" style={{justifyContent: 'flex-end'}}>
          <button className="secondary-button" onClick={onClose} type="button">
            {t('common.close')}
          </button>
        </div>
      </div>
    </div>
  );
}
