'use client';

import {Copy, X} from 'lucide-react';
import {toast} from 'sonner';

import {CompiledChainConfig} from '@/lib/control-plane-types';

type CompilationPreviewModalProps = {
  config: CompiledChainConfig;
  onClose: () => void;
};

export function CompilationPreviewModal({config, onClose}: CompilationPreviewModalProps) {
  const handleCopy = () => {
    navigator.clipboard.writeText(JSON.stringify(config, null, 2)).then(
      () => toast.success('copied to clipboard'),
      () => toast.error('failed to copy')
    );
  };

  return (
    <div className="dialog-backdrop" onClick={onClose}>
      <div className="dialog-panel" onClick={(e) => e.stopPropagation()} style={{maxWidth: 640}}>
        <div className="panel-toolbar">
          <h3>Compilation Preview</h3>
          <button className="secondary-button" onClick={onClose} type="button">
            <X size={16} />
          </button>
        </div>

        <div className="field-stack">
          <span>Routing Path</span>
          <div className="token-box">
            <div className="mono" style={{display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap'}}>
              <span className="badge is-neutral">user</span>
              {config.routingPath.map((hop, i) => (
                <span key={i} style={{display: 'inline-flex', alignItems: 'center', gap: 8}}>
                  <span className="muted-text">→</span>
                  <span className="badge is-good">{hop}</span>
                </span>
              ))}
              <span className="muted-text">→</span>
              <span className="badge is-warn">{config.destinationScope}</span>
            </div>
          </div>
        </div>

        <div className="field-stack">
          <span>Compiled Config (JSON)</span>
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
              Copy
            </button>
          </div>
        </div>

        <div className="submit-row" style={{justifyContent: 'flex-end'}}>
          <button className="secondary-button" onClick={onClose} type="button">
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
