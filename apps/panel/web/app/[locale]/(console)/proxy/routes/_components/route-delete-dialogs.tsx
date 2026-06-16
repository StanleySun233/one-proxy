'use client';

import {DeleteConfirmationModal, DeleteImpactSection} from '@/components/delete-confirmation-modal';
import {RouteRule, RouteRuleGroup} from '@/lib/types';

type RouteDeleteDialogsProps = {
  deletingRule: RouteRule | null;
  deletingGroup: RouteRuleGroup | null;
  routeDeleteSections: DeleteImpactSection[];
  groupDeleteSections: DeleteImpactSection[];
  rulePending: boolean;
  groupPending: boolean;
  routesT: (key: string) => string;
  onCloseRule: () => void;
  onConfirmRule: () => void;
  onCloseGroup: () => void;
  onConfirmGroup: () => void;
};

export function RouteDeleteDialogs({
  deletingRule,
  deletingGroup,
  routeDeleteSections,
  groupDeleteSections,
  rulePending,
  groupPending,
  routesT,
  onCloseRule,
  onConfirmRule,
  onCloseGroup,
  onConfirmGroup
}: RouteDeleteDialogsProps) {
  return (
    <>
      <DeleteConfirmationModal
        onClose={onCloseRule}
        onConfirm={onConfirmRule}
        open={Boolean(deletingRule)}
        pending={rulePending}
        sections={routeDeleteSections}
        targetName={deletingRule?.matchValue || ''}
        title={routesT('deleteConfirmTitle')}
      />
      <DeleteConfirmationModal
        onClose={onCloseGroup}
        onConfirm={onConfirmGroup}
        open={Boolean(deletingGroup)}
        pending={groupPending}
        sections={groupDeleteSections}
        targetName={deletingGroup?.name || ''}
        title={routesT('deleteGroupConfirmTitle')}
      />
    </>
  );
}
