'use client';

type RoutePageActionsProps = {
  publishPending: boolean;
  createRuleDisabled: boolean;
  t: (key: string) => string;
  routesT: (key: string) => string;
  onPublish?: () => void;
  onCreateGroup?: () => void;
  onCreateRule?: () => void;
};

export function RoutePageActions({publishPending, createRuleDisabled, t, routesT, onPublish, onCreateGroup, onCreateRule}: RoutePageActionsProps) {
  return (
    <>
      {onPublish ? (
        <button className="primary-button" disabled={publishPending} onClick={onPublish} type="button">
          {publishPending ? t('common.submitting') : routesT('publishRevision')}
        </button>
      ) : null}
      {onCreateGroup ? (
        <button className="primary-button" onClick={onCreateGroup} type="button">
          {routesT('createGroup')}
        </button>
      ) : null}
      {onCreateRule ? (
        <button className="primary-button" disabled={createRuleDisabled} onClick={onCreateRule} type="button">
          {routesT('createRule')}
        </button>
      ) : null}
    </>
  );
}
