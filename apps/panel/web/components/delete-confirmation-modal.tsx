'use client';

import {ReactNode, useMemo, useState} from 'react';
import {AlertTriangle, ListTree, Trash2} from 'lucide-react';
import {useTranslations} from 'next-intl';

import {ConsoleCrudModal} from '@/components/console-template';

export type DeleteImpactItem = {
  id: string;
  name: string;
  detail?: string;
};

export type DeleteImpactSection = {
  id: string;
  label: string;
  tone?: 'delete' | 'update';
  items?: DeleteImpactItem[];
  count?: number;
};

type DeleteConfirmationModalProps = {
  open: boolean;
  title: string;
  targetName: string;
  pending?: boolean;
  confirmLabel?: string;
  pendingLabel?: string;
  confirmIcon?: ReactNode;
  sections?: DeleteImpactSection[];
  onClose: () => void;
  onConfirm: () => void;
};

export function DeleteConfirmationModal({open, title, targetName, pending = false, confirmLabel, pendingLabel, confirmIcon, sections = [], onClose, onConfirm}: DeleteConfirmationModalProps) {
  const t = useTranslations('common');
  const [detailSectionId, setDetailSectionId] = useState('');
  const visibleSections = useMemo(
    () => sections
      .map((section) => ({...section, count: section.count ?? section.items?.length ?? 0}))
      .filter((section) => section.count > 0),
    [sections]
  );
  const detailSection = visibleSections.find((section) => section.id === detailSectionId);
  const actionIcon = confirmIcon === undefined ? <Trash2 size={14} /> : confirmIcon;
  const handleClose = () => {
    setDetailSectionId('');
    onClose();
  };

  return (
    <>
      <ConsoleCrudModal
        footer={(
          <>
            <button className="secondary-button" disabled={pending} onClick={handleClose} type="button">
              {t('cancel')}
            </button>
            <button className="danger-button" disabled={pending} onClick={onConfirm} type="button">
              {actionIcon}
              {pending ? pendingLabel || t('deleting') : confirmLabel || t('delete')}
            </button>
          </>
        )}
        onClose={handleClose}
        open={open}
        subtitle={targetName}
        title={title}
      >
        <div className="delete-confirmation">
          <div className="delete-confirmation-warning">
            <AlertTriangle size={18} />
            <span>{t('deleteDialogWarning')}</span>
          </div>
          <div className="delete-impact-summary">
            {visibleSections.length > 0 ? visibleSections.map((section) => {
              const hasDetails = Boolean(section.items?.length);
              const content = (
                <>
                  <span className={`delete-impact-dot is-${section.tone || 'delete'}`} />
                  <span>{section.label}</span>
                  <strong>{section.count}</strong>
                  {hasDetails ? <span className="delete-impact-link">{t('details')}</span> : null}
                </>
              );
              return hasDetails ? (
                <button className="delete-impact-row is-clickable" key={section.id} onClick={() => setDetailSectionId(section.id)} type="button">
                  {content}
                </button>
              ) : (
                <div className="delete-impact-row" key={section.id}>
                  {content}
                </div>
              );
            }) : (
              <div className="delete-impact-empty">
                {t('deleteDialogNoExtraImpact')}
              </div>
            )}
          </div>
        </div>
      </ConsoleCrudModal>

      {open && detailSection ? (
        <ConsoleCrudModal
          onClose={() => setDetailSectionId('')}
          open={true}
          subtitle={targetName}
          title={detailSection.label}
        >
          <div className="delete-impact-detail-list">
            {detailSection.items?.map((item) => (
              <div className="delete-impact-detail-item" key={item.id}>
                <ListTree size={16} />
                <div>
                  <strong>{item.name}</strong>
                  <span className="mono">{item.id}</span>
                  {item.detail ? <p>{item.detail}</p> : null}
                </div>
              </div>
            ))}
          </div>
        </ConsoleCrudModal>
      ) : null}
    </>
  );
}
