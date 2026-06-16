'use client';

import {Copy} from 'lucide-react';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';

import {NameTag} from '@/components/common/name-tag';
import {ConsoleCrudModal} from '@/components/console-template';
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
    <ConsoleCrudModal
      footer={<button className="secondary-button" onClick={onClose} type="button">{t('common.close')}</button>}
      onClose={onClose}
      open
      title={chainsT('compilationPreview')}
    >
        <div className="field-stack">
          <span>{chainsT('routingPath')}</span>
          <div className="token-box">
            <NodeTagPath labels={config.hops.map((hop) => hop.nodeName)} />
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

    </ConsoleCrudModal>
  );
}

function NodeTagPath({labels}: {labels: string[]}) {
  if (labels.length === 0) {
    return <span className="muted-text">-</span>;
  }
  return (
    <span className="tag-path">
      {labels.map((label, index) => (
        <span className="tag-path-step" key={`${label}-${index}`}>
          {index > 0 ? <span className="tag-path-arrow">→</span> : null}
          <NameTag kind="node">{label}</NameTag>
        </span>
      ))}
    </span>
  );
}
