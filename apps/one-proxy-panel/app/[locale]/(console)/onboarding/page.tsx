'use client';

import {useCallback, useMemo, useState} from 'react';
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {useForm} from 'react-hook-form';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {PageHero} from '@/components/page-hero';
import {
  createNodeAccessPath,
  createNodeOnboardingTask,
  deleteNodeAccessPath,
  fetchEnums,
  getNodeAccessPaths,
  getNodeOnboardingTasks,
  getNodes,
  updateNodeAccessPath,
  updateNodeOnboardingTaskStatus
} from '@/lib/control-plane-api';
import {FieldEnumMap, Node, NodeAccessPath, NodeOnboardingTask} from '@/lib/control-plane-types';
import {formatControlPlaneError, formatISODateTime, joinList, splitList} from '@/lib/presentation';

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

type PathEditorState = {
  name: string;
  mode: string;
  targetNodeId: string;
  entryNodeId: string;
  relayNodeIds: string;
  targetHost: string;
  targetPort: string;
  enabled: boolean;
};

type TaskEditorState = {
  status: string;
  statusMessage: string;
};

export default function OnboardingPage() {
  const t = useTranslations();
  const pageT = useTranslations('pages');
  const {session} = useAuth();
  const {data: enums} = useQuery({queryKey: ['enums'], queryFn: () => fetchEnums()});
  const pathModeOptions = enums?.path_mode ? Object.keys(enums.path_mode) : [];
  const taskStatusOptions = enums?.task_status ? Object.keys(enums.task_status) : [];
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';

  const [pathQuery, setPathQuery] = useState('');
  const [pathModeFilter, setPathModeFilter] = useState('all');
  const [pathEnabledFilter, setPathEnabledFilter] = useState('all');
  const [editingPathID, setEditingPathID] = useState('');
  const [pathEditorState, setPathEditorState] = useState<PathEditorState | null>(null);

  const [taskQuery, setTaskQuery] = useState('');
  const [taskModeFilter, setTaskModeFilter] = useState('all');
  const [taskStatusFilter, setTaskStatusFilter] = useState('all');
  const [editingTaskID, setEditingTaskID] = useState('');
  const [taskEditorState, setTaskEditorState] = useState<TaskEditorState | null>(null);

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

  const updatePathMutation = useMutation({
    mutationFn: (payload: {
      pathID: string;
      name: string;
      mode: string;
      targetNodeId: string;
      entryNodeId: string;
      relayNodeIds: string[];
      targetHost: string;
      targetPort: number;
      enabled: boolean;
    }) =>
      updateNodeAccessPath(accessToken, payload.pathID, {
        name: payload.name,
        mode: payload.mode,
        targetNodeId: payload.targetNodeId,
        entryNodeId: payload.entryNodeId,
        relayNodeIds: payload.relayNodeIds,
        targetHost: payload.targetHost,
        targetPort: payload.targetPort,
        enabled: payload.enabled
      }),
    onSuccess: () => {
      toast.success('access path updated');
      queryClient.invalidateQueries({queryKey: ['node-access-paths']});
      setEditingPathID('');
      setPathEditorState(null);
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const deletePathMutation = useMutation({
    mutationFn: (pathID: string) => deleteNodeAccessPath(accessToken, pathID),
    onSuccess: () => {
      toast.success('access path deleted');
      queryClient.invalidateQueries({queryKey: ['node-access-paths']});
      setEditingPathID('');
      setPathEditorState(null);
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

  const updateTaskStatusMutation = useMutation({
    mutationFn: (payload: {taskID: string; status: string; statusMessage: string}) =>
      updateNodeOnboardingTaskStatus(accessToken, payload.taskID, {
        status: payload.status,
        statusMessage: payload.statusMessage
      }),
    onSuccess: () => {
      toast.success('task status updated');
      queryClient.invalidateQueries({queryKey: ['node-onboarding-tasks']});
      setEditingTaskID('');
      setTaskEditorState(null);
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const nodes = nodesQuery.data || [];
  const paths = pathsQuery.data || [];
  const tasks = tasksQuery.data || [];

  const nodesByID = useMemo(() => new Map(nodes.map((node) => [node.id, node])), [nodes]);
  const pathsByID = useMemo(() => new Map(paths.map((path) => [path.id, path])), [paths]);
  const taskCountByPathID = useMemo(() => {
    const counts = new Map<string, number>();
    tasks.forEach((task) => {
      if (!task.pathId) {
        return;
      }
      counts.set(task.pathId, (counts.get(task.pathId) || 0) + 1);
    });
    return counts;
  }, [tasks]);
  const taskStatusClassName = useCallback((status: string) => enums?.task_status?.[status]?.meta?.className, [enums]);
  const taskSummaryByPathID = useMemo(() => {
    const summaries = new Map<string, {pending: number; failed: number; connected: number}>();
    tasks.forEach((task) => {
      if (!task.pathId) {
        return;
      }
      const current = summaries.get(task.pathId) || {pending: 0, failed: 0, connected: 0};
      const className = taskStatusClassName(task.status);
      if (className === 'is-warn') {
        current.pending += 1;
      } else if (className === 'is-danger') {
        current.failed += 1;
      } else if (className === 'is-good') {
        current.connected += 1;
      }
      summaries.set(task.pathId, current);
    });
    return summaries;
  }, [tasks, taskStatusClassName]);
  const onboardingSummary = useMemo(() => {
    const enabledPaths = paths.filter((path) => path.enabled).length;
    const relayPaths = paths.filter((path) => path.mode === 'relay_chain').length;
    const pendingTasks = tasks.filter((task) => taskStatusClassName(task.status) === 'is-warn').length;
    const failedTasks = tasks.filter((task) => taskStatusClassName(task.status) === 'is-danger').length;
    return {enabledPaths, relayPaths, pendingTasks, failedTasks};
  }, [paths, tasks, taskStatusClassName]);
  const availableTaskStatuses = useMemo(
    () => Array.from(new Set(tasks.map((task) => task.status))).sort(),
    [tasks]
  );

  const filteredPaths = useMemo(() => {
    const query = pathQuery.trim().toLowerCase();
    return paths.filter((path) => {
      if (pathModeFilter !== 'all' && path.mode !== pathModeFilter) {
        return false;
      }
      if (pathEnabledFilter !== 'all') {
        const enabled = pathEnabledFilter === 'enabled';
        if (path.enabled !== enabled) {
          return false;
        }
      }
      if (!query) {
        return true;
      }
      return [
        path.id,
        path.name,
        path.mode,
        path.targetNodeId,
        path.entryNodeId,
        path.targetHost,
        String(path.targetPort),
        path.relayNodeIds.join(' ')
      ]
        .join(' ')
        .toLowerCase()
        .includes(query);
    });
  }, [pathEnabledFilter, pathModeFilter, pathQuery, paths]);

  const filteredTasks = useMemo(() => {
    const query = taskQuery.trim().toLowerCase();
    return tasks.filter((task) => {
      if (taskModeFilter !== 'all' && task.mode !== taskModeFilter) {
        return false;
      }
      if (taskStatusFilter !== 'all' && task.status !== taskStatusFilter) {
        return false;
      }
      if (!query) {
        return true;
      }
      return [
        task.id,
        task.mode,
        task.pathId,
        task.targetNodeId,
        task.targetHost,
        task.status,
        task.statusMessage,
        task.requestedByAccountId,
        task.createdAt
      ]
        .join(' ')
        .toLowerCase()
        .includes(query);
    });
  }, [taskModeFilter, taskQuery, taskStatusFilter, tasks]);

  const editingPath = paths.find((path) => path.id === editingPathID) || null;
  const editingTask = tasks.find((task) => task.id === editingTaskID) || null;

  return (
    <AuthGate>
      <div className="page-stack">
        <PageHero eyebrow="Onboarding" title={pageT('onboardingTitle')} description={pageT('onboardingDesc')} />

        <section className="metrics-grid">
          <article className="metric-card panel-card">
            <span className="metric-label">Enabled paths</span>
            <strong>{onboardingSummary.enabledPaths}</strong>
            <span className="metric-foot">Reusable path definitions currently available for dispatch.</span>
          </article>
          <article className="metric-card panel-card soft-card">
            <span className="metric-label">Relay-chain paths</span>
            <strong>{onboardingSummary.relayPaths}</strong>
            <span className="metric-foot">Multi-hop definitions that need the clearest task visibility.</span>
          </article>
          <article className="metric-card panel-card warm-card">
            <span className="metric-label">Pending tasks</span>
            <strong>{onboardingSummary.pendingTasks}</strong>
            <span className="metric-foot">Onboarding work still waiting for node-side completion or operator follow-through.</span>
          </article>
          <article className="metric-card panel-card">
            <span className="metric-label">Failed tasks</span>
            <strong>{onboardingSummary.failedTasks}</strong>
            <span className="metric-foot">Tasks that need path or target remediation before retrying.</span>
          </article>
        </section>

        <section className="forms-grid">
          <article className="panel-card">
            <div className="panel-toolbar">
              <div>
                <p className="section-kicker">Create Path</p>
                <h3>Access path definition</h3>
                <p className="section-copy">Register the entry, relay sequence, and final target so later onboarding tasks can reuse a stable path record.</p>
              </div>
            </div>
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
              <label className="field-stack">
                <span>Name</span>
                <input
                  aria-invalid={pathForm.formState.errors.name ? 'true' : 'false'}
                  className="field-input"
                  placeholder="gateway-to-db"
                  {...pathForm.register('name', {required: 'name is required'})}
                />
                {pathForm.formState.errors.name ? <p className="error-text">{pathForm.formState.errors.name.message}</p> : null}
              </label>
              <label className="field-stack">
                <span>Mode</span>
                <select className="field-select" {...pathForm.register('mode', {required: true})}>
                  {pathModeOptions.map((mode) => (
                    <option key={mode} value={mode}>
                      {mode}
                    </option>
                  ))}
                </select>
              </label>
              <label className="field-stack">
                <span>Target node</span>
                <select className="field-select" {...pathForm.register('targetNodeId')}>
                  <option value="">Optional target node</option>
                  {nodes.map((node) => (
                    <option key={node.id} value={node.id}>
                      {node.name}
                    </option>
                  ))}
                </select>
              </label>
              <label className="field-stack">
                <span>Entry node</span>
                <select className="field-select" {...pathForm.register('entryNodeId')}>
                  <option value="">Optional entry node</option>
                  {nodes.map((node) => (
                    <option key={node.id} value={node.id}>
                      {node.name}
                    </option>
                  ))}
                </select>
              </label>
              <label className="field-stack">
                <span>Relay node ids</span>
                <input className="field-input" placeholder="relay-a, relay-b" {...pathForm.register('relayNodeIds')} />
              </label>
              <label className="field-stack">
                <span>Target host</span>
                <input
                  aria-invalid={pathForm.formState.errors.targetHost ? 'true' : 'false'}
                  className="field-input"
                  placeholder="db.internal.example.com"
                  {...pathForm.register('targetHost', {
                    validate: (value) =>
                      pathMode === 'upstream_pull' || value.trim() !== '' ? true : 'target host is required for direct or relay_chain'
                  })}
                />
                {pathForm.formState.errors.targetHost ? <p className="error-text">{pathForm.formState.errors.targetHost.message}</p> : null}
              </label>
              <label className="field-stack">
                <span>Target port</span>
                <input
                  aria-invalid={pathForm.formState.errors.targetPort ? 'true' : 'false'}
                  className="field-input"
                  placeholder="3306"
                  type="number"
                  {...pathForm.register('targetPort', {
                    validate: (value) =>
                      pathMode === 'upstream_pull' || Number(value) > 0 ? true : 'target port must be greater than 0 for direct or relay_chain'
                  })}
                />
                {pathForm.formState.errors.targetPort ? <p className="error-text">{pathForm.formState.errors.targetPort.message}</p> : null}
              </label>
              <div className="submit-row">
                <button className="primary-button" disabled={createPathMutation.isPending} type="submit">
                  {createPathMutation.isPending ? t('common.submitting') : 'Create path'}
                </button>
              </div>
            </form>
          </article>

          <article className="panel-card soft-card">
            <div className="panel-toolbar">
              <div>
                <p className="section-kicker">Create Task</p>
                <h3>Onboarding task dispatch</h3>
                <p className="section-copy">Generate a concrete execution task, then track and manually advance status until the node-side flow is fully wired.</p>
              </div>
            </div>
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
              <label className="field-stack">
                <span>Mode</span>
                <select className="field-select" {...taskForm.register('mode', {required: true})}>
                  {pathModeOptions.map((mode) => (
                    <option key={mode} value={mode}>
                      {mode}
                    </option>
                  ))}
                </select>
              </label>
              <label className="field-stack">
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
              </label>
              <label className="field-stack">
                <span>Target node</span>
                <select className="field-select" {...taskForm.register('targetNodeId')}>
                  <option value="">Optional target node</option>
                  {nodes.map((node) => (
                    <option key={node.id} value={node.id}>
                      {node.name}
                    </option>
                  ))}
                </select>
              </label>
              <label className="field-stack">
                <span>Target host</span>
                <input
                  aria-invalid={taskForm.formState.errors.targetHost ? 'true' : 'false'}
                  className="field-input"
                  placeholder="db.internal.example.com"
                  {...taskForm.register('targetHost', {
                    validate: (value) => (taskMode !== 'direct' || value.trim() !== '' ? true : 'target host is required for direct mode')
                  })}
                />
                {taskForm.formState.errors.targetHost ? <p className="error-text">{taskForm.formState.errors.targetHost.message}</p> : null}
              </label>
              <label className="field-stack">
                <span>Target port</span>
                <input
                  aria-invalid={taskForm.formState.errors.targetPort ? 'true' : 'false'}
                  className="field-input"
                  placeholder="3306"
                  type="number"
                  {...taskForm.register('targetPort', {
                    validate: (value) => (taskMode !== 'direct' || Number(value) > 0 ? true : 'target port must be greater than 0 for direct mode')
                  })}
                />
                {taskForm.formState.errors.targetPort ? <p className="error-text">{taskForm.formState.errors.targetPort.message}</p> : null}
              </label>
              <div className="submit-row">
                <button className="primary-button" disabled={createTaskMutation.isPending} type="submit">
                  {createTaskMutation.isPending ? t('common.submitting') : 'Create task'}
                </button>
              </div>
            </form>
          </article>
        </section>

        <section className="panel-card">
          <div className="panel-toolbar">
            <div>
              <p className="section-kicker">Path Registry</p>
              <h3>Access paths</h3>
              <p className="section-copy">Query definitions, inspect relay hops, and maintain records with update and delete actions.</p>
            </div>
            <div className="inline-cluster">
              <span className="badge">{filteredPaths.length} shown</span>
              <span className="badge">{paths.length} total</span>
            </div>
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
            <AsyncState detail="Create the first path before dispatching relay or upstream onboarding tasks." title={t('common.empty')} />
          ) : (
            <div className="registry-stack">
              <div className="registry-toolbar">
                <label className="field-stack registry-filter">
                  <span>Search</span>
                  <input
                    className="field-input"
                    onChange={(event) => setPathQuery(event.target.value)}
                    placeholder="Search by name, id, host, node, or relay hop"
                    type="search"
                    value={pathQuery}
                  />
                </label>
                <label className="field-stack registry-filter registry-filter-short">
                  <span>Mode</span>
                  <select className="field-select" onChange={(event) => setPathModeFilter(event.target.value)} value={pathModeFilter}>
                    <option value="all">All modes</option>
                    {pathModeOptions.map((mode) => (
                      <option key={mode} value={mode}>
                        {mode}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="field-stack registry-filter registry-filter-short">
                  <span>Enabled</span>
                  <select className="field-select" onChange={(event) => setPathEnabledFilter(event.target.value)} value={pathEnabledFilter}>
                    <option value="all">All</option>
                    <option value="enabled">enabled</option>
                    <option value="disabled">disabled</option>
                  </select>
                </label>
              </div>
              {filteredPaths.length === 0 ? (
                <AsyncState detail="Adjust the current query or filters to see matching paths." title="No matching access paths" />
              ) : (
                <div className="table-card">
                  <table className="data-table">
                    <thead>
                      <tr>
                        <th>Name</th>
                        <th>Mode</th>
                        <th>Target</th>
                        <th>Entry</th>
                        <th>Relay chain</th>
                        <th>Tasks</th>
                        <th>Status</th>
                        <th>ID</th>
                        <th>Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {filteredPaths.map((path) => {
                        const isActive = path.id === editingPathID;
                        return (
                          <tr className={isActive ? 'is-active-row' : ''} key={path.id}>
                            <td>{path.name}</td>
                            <td>{path.mode}</td>
                            <td>{describePathTarget(path, nodesByID)}</td>
                            <td>{describeNodeLabel(path.entryNodeId, nodesByID)}</td>
                            <td>{path.relayNodeIds.length > 0 ? path.relayNodeIds.join(' -> ') : <span className="muted-text">direct</span>}</td>
                            <td>
                              <div className="registry-name-cell">
                                <strong>{taskCountByPathID.get(path.id) || 0}</strong>
                                <span className="muted-text">{describePathTaskSummary(taskSummaryByPathID.get(path.id))}</span>
                              </div>
                            </td>
                            <td>
                              <span className={`badge ${path.enabled ? 'is-good-soft' : 'is-neutral'}`}>{path.enabled ? 'enabled' : 'disabled'}</span>
                            </td>
                            <td className="mono registry-id-cell">{path.id}</td>
                            <td>
                              <div className="registry-actions">
                                <button
                                  className="secondary-button"
                                  onClick={() => {
                                    if (isActive) {
                                      setEditingPathID('');
                                      setPathEditorState(null);
                                      return;
                                    }
                                    setEditingPathID(path.id);
                                    setPathEditorState({
                                      name: path.name,
                                      mode: path.mode,
                                      targetNodeId: path.targetNodeId,
                                      entryNodeId: path.entryNodeId,
                                      relayNodeIds: joinList(path.relayNodeIds),
                                      targetHost: path.targetHost,
                                      targetPort: path.targetPort > 0 ? String(path.targetPort) : '',
                                      enabled: path.enabled
                                    });
                                  }}
                                  type="button"
                                >
                                  {isActive ? 'Cancel' : 'Edit'}
                                </button>
                                <button
                                  className="danger-button"
                                  disabled={deletePathMutation.isPending}
                                  onClick={() => {
                                    if (!window.confirm(`Delete access path ${path.name} (${path.id})?`)) {
                                      return;
                                    }
                                    deletePathMutation.mutate(path.id);
                                  }}
                                  type="button"
                                >
                                  Delete
                                </button>
                              </div>
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              )}
              {editingPath && pathEditorState ? (
                <section className="node-editor-card">
                  <div className="panel-toolbar">
                    <div>
                      <p className="section-kicker">Update</p>
                      <h3>Edit access path</h3>
                      <p className="section-copy">Tune path routing, relay order, and enablement without recreating the whole record.</p>
                    </div>
                    <span className="badge mono">{editingPath.id}</span>
                  </div>
                  <div className="forms-grid">
                    <label className="field-stack">
                      <span>Name</span>
                      <input className="field-input" onChange={(event) => setPathEditorState((current) => current ? {...current, name: event.target.value} : current)} value={pathEditorState.name} />
                    </label>
                    <label className="field-stack">
                      <span>Mode</span>
                      <select className="field-select" onChange={(event) => setPathEditorState((current) => current ? {...current, mode: event.target.value} : current)} value={pathEditorState.mode}>
                        {pathModeOptions.map((mode) => (
                          <option key={mode} value={mode}>
                            {mode}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label className="field-stack">
                      <span>Target node</span>
                      <select className="field-select" onChange={(event) => setPathEditorState((current) => current ? {...current, targetNodeId: event.target.value} : current)} value={pathEditorState.targetNodeId}>
                        <option value="">Optional target node</option>
                        {nodes.map((node) => (
                          <option key={node.id} value={node.id}>
                            {node.name}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label className="field-stack">
                      <span>Entry node</span>
                      <select className="field-select" onChange={(event) => setPathEditorState((current) => current ? {...current, entryNodeId: event.target.value} : current)} value={pathEditorState.entryNodeId}>
                        <option value="">Optional entry node</option>
                        {nodes.map((node) => (
                          <option key={node.id} value={node.id}>
                            {node.name}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label className="field-stack">
                      <span>Relay node ids</span>
                      <input className="field-input" onChange={(event) => setPathEditorState((current) => current ? {...current, relayNodeIds: event.target.value} : current)} value={pathEditorState.relayNodeIds} />
                    </label>
                    <label className="field-stack">
                      <span>Target host</span>
                      <input className="field-input" onChange={(event) => setPathEditorState((current) => current ? {...current, targetHost: event.target.value} : current)} value={pathEditorState.targetHost} />
                    </label>
                    <label className="field-stack">
                      <span>Target port</span>
                      <input className="field-input" inputMode="numeric" onChange={(event) => setPathEditorState((current) => current ? {...current, targetPort: event.target.value} : current)} value={pathEditorState.targetPort} />
                    </label>
                    <label className="field-stack">
                      <span>Enabled</span>
                      <select className="field-select" onChange={(event) => setPathEditorState((current) => current ? {...current, enabled: event.target.value === 'true'} : current)} value={String(pathEditorState.enabled)}>
                        <option value="true">enabled</option>
                        <option value="false">disabled</option>
                      </select>
                    </label>
                  </div>
                  <div className="submit-row">
                    <button
                      className="primary-button"
                      disabled={updatePathMutation.isPending || pathEditorState.name.trim().length === 0}
                      onClick={() =>
                        updatePathMutation.mutate({
                          pathID: editingPath.id,
                          name: pathEditorState.name.trim(),
                          mode: pathEditorState.mode,
                          targetNodeId: pathEditorState.targetNodeId.trim(),
                          entryNodeId: pathEditorState.entryNodeId.trim(),
                          relayNodeIds: splitList(pathEditorState.relayNodeIds),
                          targetHost: pathEditorState.targetHost.trim(),
                          targetPort: pathEditorState.targetPort.trim() ? Number(pathEditorState.targetPort) : 0,
                          enabled: pathEditorState.enabled
                        })
                      }
                      type="button"
                    >
                      Save changes
                    </button>
                    <button className="secondary-button" onClick={() => { setEditingPathID(''); setPathEditorState(null); }} type="button">
                      Close
                    </button>
                  </div>
                </section>
              ) : null}
            </div>
          )}
        </section>

        <section className="panel-card">
          <div className="panel-toolbar">
            <div>
              <p className="section-kicker">Task Registry</p>
              <h3>Onboarding tasks</h3>
              <p className="section-copy">Track execution state, inspect task intent, and manually advance status while node-side automation is still evolving.</p>
            </div>
            <div className="inline-cluster">
              <span className="badge">{filteredTasks.length} shown</span>
              <span className="badge">{tasks.length} total</span>
            </div>
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
            <div className="registry-stack">
              <div className="registry-toolbar">
                <label className="field-stack registry-filter">
                  <span>Search</span>
                  <input
                    className="field-input"
                    onChange={(event) => setTaskQuery(event.target.value)}
                    placeholder="Search by id, target, path, mode, or status message"
                    type="search"
                    value={taskQuery}
                  />
                </label>
                <label className="field-stack registry-filter registry-filter-short">
                  <span>Status</span>
                  <select className="field-select" onChange={(event) => setTaskStatusFilter(event.target.value)} value={taskStatusFilter}>
                    <option value="all">All statuses</option>
                    {availableTaskStatuses.map((status) => (
                      <option key={status} value={status}>
                        {status}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="field-stack registry-filter registry-filter-short">
                  <span>Mode</span>
                  <select className="field-select" onChange={(event) => setTaskModeFilter(event.target.value)} value={taskModeFilter}>
                    <option value="all">All modes</option>
                    {pathModeOptions.map((mode) => (
                      <option key={mode} value={mode}>
                        {mode}
                      </option>
                    ))}
                  </select>
                </label>
              </div>
              {filteredTasks.length === 0 ? (
                <AsyncState detail="Adjust the current query or filters to see matching onboarding tasks." title="No matching tasks" />
              ) : (
                <div className="table-card">
                  <table className="data-table">
                    <thead>
                      <tr>
                        <th>Target</th>
                        <th>Status</th>
                        <th>Mode</th>
                        <th>Path</th>
                        <th>Requested at</th>
                        <th>Requested by</th>
                        <th>Updated</th>
                        <th>ID</th>
                        <th>Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {filteredTasks.map((task) => {
                        const isActive = task.id === editingTaskID;
                        return (
                          <tr className={isActive ? 'is-active-row' : ''} key={task.id}>
                            <td>
                              <div className="registry-name-cell">
                                <strong>{describeTaskTarget(task, nodesByID)}</strong>
                                <span className="muted-text">{task.statusMessage || 'no-status-message'}</span>
                              </div>
                            </td>
                            <td>
                              <span className={taskBadgeClassName(task.status, enums)}>{task.status}</span>
                            </td>
                            <td>{task.mode}</td>
                            <td>{describeTaskPath(task.pathId, pathsByID)}</td>
                            <td className="mono">{formatISODateTime(task.createdAt)}</td>
                            <td>{task.requestedByAccountId || <span className="muted-text">system</span>}</td>
                            <td className="mono">{formatISODateTime(task.updatedAt || task.createdAt)}</td>
                            <td className="mono registry-id-cell">{task.id}</td>
                            <td>
                              <div className="registry-actions">
                                <button
                                  className="secondary-button"
                                  onClick={() => {
                                    if (isActive) {
                                      setEditingTaskID('');
                                      setTaskEditorState(null);
                                      return;
                                    }
                                    setEditingTaskID(task.id);
                                    setTaskEditorState({
                                      status: task.status,
                                      statusMessage: task.statusMessage
                                    });
                                  }}
                                  type="button"
                                >
                                  {isActive ? 'Cancel' : 'Update'}
                                </button>
                              </div>
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              )}
              {editingTask && taskEditorState ? (
                <section className="node-editor-card">
                  <div className="panel-toolbar">
                    <div>
                      <p className="section-kicker">Update</p>
                      <h3>Advance task status</h3>
                      <p className="section-copy">Use explicit status and operator notes so onboarding progress remains queryable before automated relay execution is complete.</p>
                    </div>
                    <span className="badge mono">{editingTask.id}</span>
                  </div>
                  <div className="forms-grid">
                    <label className="field-stack">
                      <span>Status</span>
                      <select className="field-select" onChange={(event) => setTaskEditorState((current) => current ? {...current, status: event.target.value} : current)} value={taskEditorState.status}>
                        {taskStatusOptions.map((status) => (
                          <option key={status} value={status}>
                            {status}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label className="field-stack">
                      <span>Status message</span>
                      <input className="field-input" onChange={(event) => setTaskEditorState((current) => current ? {...current, statusMessage: event.target.value} : current)} value={taskEditorState.statusMessage} />
                    </label>
                  </div>
                  <div className="submit-row">
                    <button
                      className="primary-button"
                      disabled={updateTaskStatusMutation.isPending || taskEditorState.status.trim().length === 0}
                      onClick={() =>
                        updateTaskStatusMutation.mutate({
                          taskID: editingTask.id,
                          status: taskEditorState.status,
                          statusMessage: taskEditorState.statusMessage.trim()
                        })
                      }
                      type="button"
                    >
                      Save task status
                    </button>
                    <button className="secondary-button" onClick={() => { setEditingTaskID(''); setTaskEditorState(null); }} type="button">
                      Close
                    </button>
                  </div>
                </section>
              ) : null}
            </div>
          )}
        </section>
      </div>
    </AuthGate>
  );
}

function describeNodeLabel(nodeID: string, nodesByID: Map<string, Node>) {
  if (!nodeID) {
    return <span className="muted-text">none</span>;
  }
  return nodesByID.get(nodeID)?.name || nodeID;
}

function describePathTarget(path: NodeAccessPath, nodesByID: Map<string, Node>) {
  if (path.targetNodeId) {
    return describeNodeLabel(path.targetNodeId, nodesByID);
  }
  if (path.targetHost) {
    return `${path.targetHost}${path.targetPort > 0 ? `:${path.targetPort}` : ''}`;
  }
  return <span className="muted-text">unassigned</span>;
}

function describeTaskTarget(task: NodeOnboardingTask, nodesByID: Map<string, Node>) {
  if (task.targetNodeId) {
    return nodesByID.get(task.targetNodeId)?.name || task.targetNodeId;
  }
  if (task.targetHost) {
    return `${task.targetHost}${task.targetPort > 0 ? `:${task.targetPort}` : ''}`;
  }
  return task.id;
}

function describeTaskPath(pathID: string, pathsByID: Map<string, NodeAccessPath>) {
  if (!pathID) {
    return <span className="muted-text">no-path</span>;
  }
  return pathsByID.get(pathID)?.name || pathID;
}

function describePathTaskSummary(summary?: {pending: number; failed: number; connected: number}) {
  if (!summary) {
    return 'no tasks';
  }
  const parts = [];
  if (summary.pending > 0) {
    parts.push(`${summary.pending} pending`);
  }
  if (summary.failed > 0) {
    parts.push(`${summary.failed} failed`);
  }
  if (summary.connected > 0) {
    parts.push(`${summary.connected} connected`);
  }
  return parts.length > 0 ? parts.join(' · ') : 'no tasks';
}

function taskBadgeClassName(status: string, enums: FieldEnumMap | undefined): string {
  const entry = enums?.task_status?.[status];
  if (entry?.meta?.className) {
    return `badge ${entry.meta.className}`;
  }
  return 'badge is-neutral';
}
