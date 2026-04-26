'use client';

import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {useForm} from 'react-hook-form';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {PageHero} from '@/components/page-hero';
import {createNodeAccessPath, createNodeOnboardingTask, getNodeAccessPaths, getNodeOnboardingTasks, getNodes} from '@/lib/control-plane-api';
import {formatControlPlaneError, joinList, splitList} from '@/lib/presentation';

type AccessPathFormValues = {
  name: string;
  mode: string;
  targetNodeId: string;
  entryNodeId: string;
  relayNodeIds: string;
  targetHost: string;
  targetPort: string;
};

type OnboardingTaskFormValues = {
  mode: string;
  pathId: string;
  targetNodeId: string;
  targetHost: string;
  targetPort: string;
};

export default function OnboardingPage() {
  const t = useTranslations();
  const pageT = useTranslations('pages');
  const {session} = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';

  const pathForm = useForm<AccessPathFormValues>({
    defaultValues: {
      name: '',
      mode: 'upstream_pull',
      targetNodeId: '',
      entryNodeId: '',
      relayNodeIds: '',
      targetHost: '',
      targetPort: ''
    }
  });
  const taskForm = useForm<OnboardingTaskFormValues>({
    defaultValues: {
      mode: 'upstream_pull',
      pathId: '',
      targetNodeId: '',
      targetHost: '',
      targetPort: ''
    }
  });

  const pathMode = pathForm.watch('mode');
  const taskMode = taskForm.watch('mode');

  const nodesQuery = useQuery({
    queryKey: ['nodes', accessToken],
    queryFn: () => getNodes(accessToken),
    enabled: !!accessToken
  });
  const pathsQuery = useQuery({
    queryKey: ['node-access-paths', accessToken],
    queryFn: () => getNodeAccessPaths(accessToken),
    enabled: !!accessToken
  });
  const tasksQuery = useQuery({
    queryKey: ['node-onboarding-tasks', accessToken],
    queryFn: () => getNodeOnboardingTasks(accessToken),
    enabled: !!accessToken
  });

  const createPathMutation = useMutation({
    mutationFn: (payload: {
      name: string;
      mode: string;
      targetNodeId: string;
      entryNodeId: string;
      relayNodeIds: string[];
      targetHost: string;
      targetPort: number;
    }) => createNodeAccessPath(accessToken, payload),
    onSuccess: () => {
      toast.success('access path created');
      queryClient.invalidateQueries({queryKey: ['node-access-paths']});
      pathForm.reset({
        name: '',
        mode: 'upstream_pull',
        targetNodeId: '',
        entryNodeId: '',
        relayNodeIds: '',
        targetHost: '',
        targetPort: ''
      });
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const createTaskMutation = useMutation({
    mutationFn: (payload: {
      mode: string;
      pathId: string;
      targetNodeId: string;
      targetHost: string;
      targetPort: number;
    }) => createNodeOnboardingTask(accessToken, payload),
    onSuccess: () => {
      toast.success('onboarding task created');
      queryClient.invalidateQueries({queryKey: ['node-onboarding-tasks']});
      taskForm.reset({
        mode: 'upstream_pull',
        pathId: '',
        targetNodeId: '',
        targetHost: '',
        targetPort: ''
      });
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const nodes = nodesQuery.data || [];
  const paths = pathsQuery.data || [];
  const tasks = tasksQuery.data || [];

  return (
    <AuthGate>
      <div className="page-stack">
        <PageHero eyebrow="Onboarding" title={pageT('onboardingTitle')} description={pageT('onboardingDesc')} />

        <section className="forms-grid">
          <article className="panel-card">
            <h3>Create access path</h3>
            <form
              className="sub-grid"
              onSubmit={pathForm.handleSubmit((values) => {
                createPathMutation.mutate({
                  name: values.name.trim(),
                  mode: values.mode,
                  targetNodeId: values.targetNodeId.trim(),
                  entryNodeId: values.entryNodeId.trim(),
                  relayNodeIds: splitList(values.relayNodeIds),
                  targetHost: values.targetHost.trim(),
                  targetPort: values.targetPort ? Number(values.targetPort) : 0
                });
              })}
            >
              <div className="field-stack">
                <span>Name</span>
                <input
                  aria-invalid={pathForm.formState.errors.name ? 'true' : 'false'}
                  className="field-input"
                  placeholder="node1-to-node2"
                  {...pathForm.register('name', {required: 'name is required'})}
                />
                {pathForm.formState.errors.name ? <p className="error-text">{pathForm.formState.errors.name.message}</p> : null}
              </div>
              <div className="field-stack">
                <span>Mode</span>
                <select className="field-select" {...pathForm.register('mode', {required: true})}>
                  <option value="upstream_pull">upstream_pull</option>
                  <option value="relay_chain">relay_chain</option>
                  <option value="direct">direct</option>
                </select>
              </div>
              <div className="field-stack">
                <span>Target node id</span>
                <select className="field-select" {...pathForm.register('targetNodeId')}>
                  <option value="">Optional target node</option>
                  {nodes.map((node) => (
                    <option key={node.id} value={node.id}>
                      {node.name}
                    </option>
                  ))}
                </select>
              </div>
              <div className="field-stack">
                <span>Entry node id</span>
                <select className="field-select" {...pathForm.register('entryNodeId')}>
                  <option value="">Optional entry node</option>
                  {nodes.map((node) => (
                    <option key={node.id} value={node.id}>
                      {node.name}
                    </option>
                  ))}
                </select>
              </div>
              <div className="field-stack">
                <span>Relay node ids</span>
                <input className="field-input" placeholder="node-a, node-b" {...pathForm.register('relayNodeIds')} />
              </div>
              <div className="field-stack">
                <span>Target host</span>
                <input
                  aria-invalid={pathForm.formState.errors.targetHost ? 'true' : 'false'}
                  className="field-input"
                  placeholder="127.0.0.1"
                  {...pathForm.register('targetHost', {
                    validate: (value) =>
                      pathMode === 'upstream_pull' || value.trim() !== '' ? true : 'target host is required for direct or relay_chain'
                  })}
                />
                {pathForm.formState.errors.targetHost ? <p className="error-text">{pathForm.formState.errors.targetHost.message}</p> : null}
              </div>
              <div className="field-stack">
                <span>Target port</span>
                <input
                  aria-invalid={pathForm.formState.errors.targetPort ? 'true' : 'false'}
                  className="field-input"
                  placeholder="2888"
                  type="number"
                  {...pathForm.register('targetPort', {
                    validate: (value) =>
                      pathMode === 'upstream_pull' || Number(value) > 0 ? true : 'target port must be greater than 0 for direct or relay_chain'
                  })}
                />
                {pathForm.formState.errors.targetPort ? <p className="error-text">{pathForm.formState.errors.targetPort.message}</p> : null}
              </div>
              <div className="submit-row">
                <button className="primary-button" disabled={createPathMutation.isPending} type="submit">
                  {createPathMutation.isPending ? t('common.submitting') : 'Create path'}
                </button>
              </div>
            </form>
          </article>

          <article className="panel-card soft-card">
            <h3>Create onboarding task</h3>
            <form
              className="sub-grid"
              onSubmit={taskForm.handleSubmit((values) => {
                createTaskMutation.mutate({
                  mode: values.mode,
                  pathId: values.pathId.trim(),
                  targetNodeId: values.targetNodeId.trim(),
                  targetHost: values.targetHost.trim(),
                  targetPort: values.targetPort ? Number(values.targetPort) : 0
                });
              })}
            >
              <div className="field-stack">
                <span>Mode</span>
                <select className="field-select" {...taskForm.register('mode', {required: true})}>
                  <option value="upstream_pull">upstream_pull</option>
                  <option value="relay_chain">relay_chain</option>
                  <option value="direct">direct</option>
                </select>
              </div>
              <div className="field-stack">
                <span>Path</span>
                <select
                  aria-invalid={taskForm.formState.errors.pathId ? 'true' : 'false'}
                  className="field-select"
                  {...taskForm.register('pathId', {
                    validate: (value) => (taskMode === 'direct' || value.trim() !== '' ? true : 'path is required for upstream_pull or relay_chain')
                  })}
                >
                  <option value="">Select path</option>
                  {paths.map((path) => (
                    <option key={path.id} value={path.id}>
                      {path.name}
                    </option>
                  ))}
                </select>
                {taskForm.formState.errors.pathId ? <p className="error-text">{taskForm.formState.errors.pathId.message}</p> : null}
              </div>
              <div className="field-stack">
                <span>Target node id</span>
                <select className="field-select" {...taskForm.register('targetNodeId')}>
                  <option value="">Optional target node</option>
                  {nodes.map((node) => (
                    <option key={node.id} value={node.id}>
                      {node.name}
                    </option>
                  ))}
                </select>
              </div>
              <div className="field-stack">
                <span>Target host</span>
                <input
                  aria-invalid={taskForm.formState.errors.targetHost ? 'true' : 'false'}
                  className="field-input"
                  placeholder="127.0.0.1"
                  {...taskForm.register('targetHost', {
                    validate: (value) => (taskMode !== 'direct' || value.trim() !== '' ? true : 'target host is required for direct mode')
                  })}
                />
                {taskForm.formState.errors.targetHost ? <p className="error-text">{taskForm.formState.errors.targetHost.message}</p> : null}
              </div>
              <div className="field-stack">
                <span>Target port</span>
                <input
                  aria-invalid={taskForm.formState.errors.targetPort ? 'true' : 'false'}
                  className="field-input"
                  placeholder="2888"
                  type="number"
                  {...taskForm.register('targetPort', {
                    validate: (value) => (taskMode !== 'direct' || Number(value) > 0 ? true : 'target port must be greater than 0 for direct mode')
                  })}
                />
                {taskForm.formState.errors.targetPort ? <p className="error-text">{taskForm.formState.errors.targetPort.message}</p> : null}
              </div>
              <div className="submit-row">
                <button className="primary-button" disabled={createTaskMutation.isPending} type="submit">
                  {createTaskMutation.isPending ? t('common.submitting') : 'Create task'}
                </button>
              </div>
            </form>
          </article>
        </section>

        <section className="two-column-grid">
          <article className="panel-card">
            <div className="panel-toolbar">
              <h3>Access paths</h3>
              <span className="badge">{paths.length}</span>
            </div>
            {pathsQuery.isPending ? (
              <AsyncState detail={t('common.loading')} title="Loading access paths" />
            ) : pathsQuery.isError ? (
              <AsyncState
                actionLabel={t('common.retry')}
                detail={formatControlPlaneError(pathsQuery.error)}
                onAction={() => void pathsQuery.refetch()}
                title="Failed to load access paths"
              />
            ) : paths.length === 0 ? (
              <AsyncState detail="Create a path before creating relay or upstream onboarding tasks." title={t('common.empty')} />
            ) : (
              <div className="stack-list">
                {paths.map((path) => (
                  <div className="stack-item" key={path.id}>
                    <div className="stack-head">
                      <strong>{path.name}</strong>
                      <span className="badge">{path.mode}</span>
                    </div>
                    <div className="inline-cluster">
                      <span className="muted-text">target: {path.targetNodeId || path.targetHost || '-'}</span>
                      <span className="muted-text">entry: {path.entryNodeId || '-'}</span>
                    </div>
                    <span className="mono">{path.relayNodeIds.length > 0 ? joinList(path.relayNodeIds) : '-'}</span>
                  </div>
                ))}
              </div>
            )}
          </article>

          <article className="panel-card warm-card">
            <div className="panel-toolbar">
              <h3>Onboarding tasks</h3>
              <span className="badge">{tasks.length}</span>
            </div>
            {tasksQuery.isPending ? (
              <AsyncState detail={t('common.loading')} title="Loading onboarding tasks" />
            ) : tasksQuery.isError ? (
              <AsyncState
                actionLabel={t('common.retry')}
                detail={formatControlPlaneError(tasksQuery.error)}
                onAction={() => void tasksQuery.refetch()}
                title="Failed to load onboarding tasks"
              />
            ) : tasks.length === 0 ? (
              <AsyncState detail="Tasks will appear here once an operator triggers direct, relay_chain or upstream_pull onboarding." title={t('common.empty')} />
            ) : (
              <div className="stack-list">
                {tasks.map((task) => (
                  <div className="stack-item" key={task.id}>
                    <div className="stack-head">
                      <strong>{task.targetNodeId || task.targetHost || task.id}</strong>
                      <span className={`badge ${task.status === 'connected' ? 'is-good' : 'is-warn'}`}>{task.status}</span>
                    </div>
                    <span className="muted-text">
                      {task.mode} · {task.statusMessage}
                    </span>
                    <span className="mono">{task.createdAt}</span>
                  </div>
                ))}
              </div>
            )}
          </article>
        </section>
      </div>
    </AuthGate>
  );
}
