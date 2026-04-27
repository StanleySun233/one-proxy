'use client';

import { useCallback, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';

import { useAuth } from '@/components/auth-provider';
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
} from '@/lib/api';
import type { FieldEnumMap, Node, NodeAccessPath, NodeOnboardingTask } from '@/lib/types';
import { formatControlPlaneError, joinList, splitList } from '@/lib/presentation';

export type AccessPathFormValues = {
  name: string;
  mode: string;
  targetNodeId: string;
  entryNodeId: string;
  relayNodeIds: string;
  targetHost: string;
  targetPort: string;
};

export type OnboardingTaskFormValues = {
  mode: string;
  pathId: string;
  targetNodeId: string;
  targetHost: string;
  targetPort: string;
};

export type PathEditorState = {
  name: string;
  mode: string;
  targetNodeId: string;
  entryNodeId: string;
  relayNodeIds: string;
  targetHost: string;
  targetPort: string;
  enabled: boolean;
};

export type TaskEditorState = {
  status: string;
  statusMessage: string;
};

export function useOnboarding() {
  const t = useTranslations();
  const { session } = useAuth();
  const { data: enums } = useQuery({ queryKey: ['enums'], queryFn: () => fetchEnums() });
  const pathModeOptions = enums?.path_mode ? Object.keys(enums.path_mode) : [];
  const taskStatusOptions = enums?.task_status ? Object.keys(enums.task_status) : [];
  const pathModeKeys = Object.keys(enums?.path_mode || {});
  const UPSTREAM_PULL = pathModeKeys.find(k => k === 'upstream_pull') || 'upstream_pull';
  const RELAY_CHAIN = pathModeKeys.find(k => k === 'relay_chain') || 'relay_chain';
  const DIRECT = pathModeKeys.find(k => k === 'direct') || 'direct';
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
      mode: UPSTREAM_PULL,
      targetNodeId: '',
      entryNodeId: '',
      relayNodeIds: '',
      targetHost: '',
      targetPort: ''
    }
  });
  const taskForm = useForm<OnboardingTaskFormValues>({
    defaultValues: {
      mode: UPSTREAM_PULL,
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
      name: string; mode: string; targetNodeId: string; entryNodeId: string;
      relayNodeIds: string[]; targetHost: string; targetPort: number;
    }) => createNodeAccessPath(accessToken, payload),
    onSuccess: () => {
      toast.success('access path created');
      queryClient.invalidateQueries({ queryKey: ['node-access-paths'] });
      pathForm.reset({
        name: '', mode: UPSTREAM_PULL, targetNodeId: '', entryNodeId: '',
        relayNodeIds: '', targetHost: '', targetPort: ''
      });
    },
    onError: (error) => { toast.error(formatControlPlaneError(error)); }
  });

  const updatePathMutation = useMutation({
    mutationFn: (payload: {
      pathID: string; name: string; mode: string; targetNodeId: string; entryNodeId: string;
      relayNodeIds: string[]; targetHost: string; targetPort: number; enabled: boolean;
    }) => updateNodeAccessPath(accessToken, payload.pathID, {
      name: payload.name, mode: payload.mode, targetNodeId: payload.targetNodeId,
      entryNodeId: payload.entryNodeId, relayNodeIds: payload.relayNodeIds,
      targetHost: payload.targetHost, targetPort: payload.targetPort, enabled: payload.enabled
    }),
    onSuccess: () => {
      toast.success('access path updated');
      queryClient.invalidateQueries({ queryKey: ['node-access-paths'] });
      setEditingPathID('');
      setPathEditorState(null);
    },
    onError: (error) => { toast.error(formatControlPlaneError(error)); }
  });

  const deletePathMutation = useMutation({
    mutationFn: (pathID: string) => deleteNodeAccessPath(accessToken, pathID),
    onSuccess: () => {
      toast.success('access path deleted');
      queryClient.invalidateQueries({ queryKey: ['node-access-paths'] });
      setEditingPathID('');
      setPathEditorState(null);
    },
    onError: (error) => { toast.error(formatControlPlaneError(error)); }
  });

  const createTaskMutation = useMutation({
    mutationFn: (payload: {
      mode: string; pathId: string; targetNodeId: string; targetHost: string; targetPort: number;
    }) => createNodeOnboardingTask(accessToken, payload),
    onSuccess: () => {
      toast.success('onboarding task created');
      queryClient.invalidateQueries({ queryKey: ['node-onboarding-tasks'] });
      taskForm.reset({
        mode: UPSTREAM_PULL, pathId: '', targetNodeId: '', targetHost: '', targetPort: ''
      });
    },
    onError: (error) => { toast.error(formatControlPlaneError(error)); }
  });

  const updateTaskStatusMutation = useMutation({
    mutationFn: (payload: { taskID: string; status: string; statusMessage: string }) =>
      updateNodeOnboardingTaskStatus(accessToken, payload.taskID, {
        status: payload.status, statusMessage: payload.statusMessage
      }),
    onSuccess: () => {
      toast.success('task status updated');
      queryClient.invalidateQueries({ queryKey: ['node-onboarding-tasks'] });
      setEditingTaskID('');
      setTaskEditorState(null);
    },
    onError: (error) => { toast.error(formatControlPlaneError(error)); }
  });

  const nodes = nodesQuery.data || [];
  const paths = pathsQuery.data || [];
  const tasks = tasksQuery.data || [];

  const nodesByID = useMemo(() => new Map(nodes.map((node) => [node.id, node])), [nodes]);
  const pathsByID = useMemo(() => new Map(paths.map((path) => [path.id, path])), [paths]);
  const taskCountByPathID = useMemo(() => {
    const counts = new Map<string, number>();
    tasks.forEach((task) => {
      if (!task.pathId) return;
      counts.set(task.pathId, (counts.get(task.pathId) || 0) + 1);
    });
    return counts;
  }, [tasks]);
  const taskStatusClassName = useCallback((status: string) => enums?.task_status?.[status]?.meta?.className, [enums]);
  const taskSummaryByPathID = useMemo(() => {
    const summaries = new Map<string, { pending: number; failed: number; connected: number }>();
    tasks.forEach((task) => {
      if (!task.pathId) return;
      const current = summaries.get(task.pathId) || { pending: 0, failed: 0, connected: 0 };
      const className = taskStatusClassName(task.status);
      if (className === 'is-warn') current.pending += 1;
      else if (className === 'is-danger') current.failed += 1;
      else if (className === 'is-good') current.connected += 1;
      summaries.set(task.pathId, current);
    });
    return summaries;
  }, [tasks, taskStatusClassName]);
  const onboardingSummary = useMemo(() => {
    const enabledPaths = paths.filter((path) => path.enabled).length;
    const relayPaths = paths.filter((path) => path.mode === RELAY_CHAIN).length;
    const pendingTasks = tasks.filter((task) => taskStatusClassName(task.status) === 'is-warn').length;
    const failedTasks = tasks.filter((task) => taskStatusClassName(task.status) === 'is-danger').length;
    return { enabledPaths, relayPaths, pendingTasks, failedTasks };
  }, [paths, tasks, taskStatusClassName, RELAY_CHAIN]);
  const availableTaskStatuses = useMemo(
    () => Array.from(new Set(tasks.map((task) => task.status))).sort(),
    [tasks]
  );

  const filteredPaths = useMemo(() => {
    const query = pathQuery.trim().toLowerCase();
    return paths.filter((path) => {
      if (pathModeFilter !== 'all' && path.mode !== pathModeFilter) return false;
      if (pathEnabledFilter !== 'all') {
        const enabled = pathEnabledFilter === 'enabled';
        if (path.enabled !== enabled) return false;
      }
      if (!query) return true;
      return [path.id, path.name, path.mode, path.targetNodeId, path.entryNodeId,
        path.targetHost, String(path.targetPort), path.relayNodeIds.join(' ')]
        .join(' ').toLowerCase().includes(query);
    });
  }, [pathEnabledFilter, pathModeFilter, pathQuery, paths]);

  const filteredTasks = useMemo(() => {
    const query = taskQuery.trim().toLowerCase();
    return tasks.filter((task) => {
      if (taskModeFilter !== 'all' && task.mode !== taskModeFilter) return false;
      if (taskStatusFilter !== 'all' && task.status !== taskStatusFilter) return false;
      if (!query) return true;
      return [task.id, task.mode, task.pathId, task.targetNodeId, task.targetHost,
        task.status, task.statusMessage, task.requestedByAccountId, task.createdAt]
        .join(' ').toLowerCase().includes(query);
    });
  }, [taskModeFilter, taskQuery, taskStatusFilter, tasks]);

  const editingPath = paths.find((path) => path.id === editingPathID) || null;
  const editingTask = tasks.find((task) => task.id === editingTaskID) || null;

  return {
    t,
    enums,
    pathModeOptions,
    taskStatusOptions,
    UPSTREAM_PULL,
    RELAY_CHAIN,
    DIRECT,
    pathForm,
    taskForm,
    pathMode,
    taskMode,
    nodes,
    paths,
    tasks,
    queryClient,
    pathQuery,
    setPathQuery,
    pathModeFilter,
    setPathModeFilter,
    pathEnabledFilter,
    setPathEnabledFilter,
    editingPathID,
    setEditingPathID,
    pathEditorState,
    setPathEditorState,
    taskQuery,
    setTaskQuery,
    taskModeFilter,
    setTaskModeFilter,
    taskStatusFilter,
    setTaskStatusFilter,
    editingTaskID,
    setEditingTaskID,
    taskEditorState,
    setTaskEditorState,
    nodesQuery,
    pathsQuery,
    tasksQuery,
    createPathMutation,
    updatePathMutation,
    deletePathMutation,
    createTaskMutation,
    updateTaskStatusMutation,
    nodesByID,
    pathsByID,
    taskCountByPathID,
    taskSummaryByPathID,
    onboardingSummary,
    availableTaskStatuses,
    filteredPaths,
    filteredTasks,
    editingPath,
    editingTask,
  };
}

export function describeNodeLabel(nodeID: string, nodesByID: Map<string, Node>) {
  if (!nodeID) return <span className="muted-text">none</span>;
  return nodesByID.get(nodeID)?.name || nodeID;
}

export function describePathTarget(path: NodeAccessPath, nodesByID: Map<string, Node>) {
  if (path.targetNodeId) return describeNodeLabel(path.targetNodeId, nodesByID);
  if (path.targetHost) return `${path.targetHost}${path.targetPort > 0 ? `:${path.targetPort}` : ''}`;
  return <span className="muted-text">unassigned</span>;
}

export function describeTaskTarget(task: NodeOnboardingTask, nodesByID: Map<string, Node>) {
  if (task.targetNodeId) return nodesByID.get(task.targetNodeId)?.name || task.targetNodeId;
  if (task.targetHost) return `${task.targetHost}${task.targetPort > 0 ? `:${task.targetPort}` : ''}`;
  return task.id;
}

export function describeTaskPath(pathID: string, pathsByID: Map<string, NodeAccessPath>) {
  if (!pathID) return <span className="muted-text">no-path</span>;
  return pathsByID.get(pathID)?.name || pathID;
}

export function describePathTaskSummary(summary?: { pending: number; failed: number; connected: number }) {
  if (!summary) return 'no tasks';
  const parts = [];
  if (summary.pending > 0) parts.push(`${summary.pending} pending`);
  if (summary.failed > 0) parts.push(`${summary.failed} failed`);
  if (summary.connected > 0) parts.push(`${summary.connected} connected`);
  return parts.length > 0 ? parts.join(' · ') : 'no tasks';
}

export function taskBadgeClassName(status: string, enums: FieldEnumMap | undefined): string {
  const entry = enums?.task_status?.[status];
  if (entry?.meta?.className) return `badge ${entry.meta.className}`;
  return 'badge is-neutral';
}
