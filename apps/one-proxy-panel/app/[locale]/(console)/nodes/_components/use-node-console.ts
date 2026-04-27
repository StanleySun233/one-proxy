'use client';

import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {useForm} from 'react-hook-form';
import {toast} from 'sonner';

import {useAuth} from '@/components/auth-provider';
import {BootstrapToken} from '@/lib/types';
import {
  approveNode,
  connectNode,
  createBootstrapToken,
  createNode,
  createNodeLink,
  deleteNode,
  fetchEnums,
  getNodeHealth,
  getNodeLinks,
  getNodes,
  getNodeTransports,
  getPendingNodes,
  getUnconsumedBootstrapTokens,
  rejectNode,
  updateNode
} from '@/lib/api';
import {formatControlPlaneError} from '@/lib/presentation';

import {BootstrapFormValues, NodeFormValues, QuickConnectFormValues} from './types';

export function useNodeConsole() {
  const {session} = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';

  const {data: enums} = useQuery({queryKey: ['enums'], queryFn: () => fetchEnums()});
  const nodeModeKeys = Object.keys(enums?.node_mode || {});
  const DEFAULT_MODE = nodeModeKeys.find(k => k === 'relay') || 'relay';
  const bootstrapTargetKeys = Object.keys(enums?.bootstrap_target_type || {});
  const DEFAULT_TARGET_TYPE = bootstrapTargetKeys.find(k => k === 'node') || 'node';

  const nodeForm = useForm<NodeFormValues>({
    defaultValues: {
      name: '',
      mode: DEFAULT_MODE,
      scopeKey: '',
      parentNodeId: '',
      publicHost: '',
      publicPort: ''
    }
  });

  const quickConnectForm = useForm<QuickConnectFormValues>({
    defaultValues: {
      address: '',
      password: '',
      newPassword: '',
      name: '',
      mode: DEFAULT_MODE,
      scopeKey: '',
      parentNodeId: '',
      publicHost: '',
      publicPort: '',
      controlPlaneUrl: ''
    }
  });

  const bootstrapForm = useForm<BootstrapFormValues>({
    defaultValues: {
      targetId: '',
      nodeName: ''
    }
  });

  const nodesQuery = useQuery({
    queryKey: ['nodes', accessToken],
    queryFn: () => getNodes(accessToken),
    enabled: !!accessToken
  });

  const linksQuery = useQuery({
    queryKey: ['node-links', accessToken],
    queryFn: () => getNodeLinks(accessToken),
    enabled: !!accessToken
  });

  const healthQuery = useQuery({
    queryKey: ['node-health', accessToken],
    queryFn: () => getNodeHealth(accessToken),
    enabled: !!accessToken,
    refetchInterval: 5000
  });

  const transportsQuery = useQuery({
    queryKey: ['node-transports', accessToken],
    queryFn: () => getNodeTransports(accessToken),
    enabled: !!accessToken,
    refetchInterval: 5000
  });

  const pendingNodesQuery = useQuery({
    queryKey: ['pending-nodes', accessToken],
    queryFn: () => getPendingNodes(accessToken),
    enabled: !!accessToken,
    refetchInterval: 30000
  });

  const unconsumedTokensQuery = useQuery({
    queryKey: ['unconsumed-bootstrap-tokens', accessToken],
    queryFn: () => getUnconsumedBootstrapTokens(accessToken),
    enabled: !!accessToken,
    refetchInterval: 30000
  });

  const createNodeMutation = useMutation({
    mutationFn: (payload: {
      name: string;
      mode: string;
      scopeKey: string;
      parentNodeId: string;
      publicHost: string;
      publicPort: number;
    }) => createNode(accessToken, payload),
    onSuccess: () => {
      toast.success('node created');
      queryClient.invalidateQueries({queryKey: ['nodes']});
      nodeForm.reset();
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const quickConnectMutation = useMutation({
    mutationFn: (payload: {
      address: string;
      password: string;
      newPassword: string;
      name: string;
      mode: string;
      scopeKey: string;
      parentNodeId: string;
      publicHost: string;
      publicPort: number;
      controlPlaneUrl: string;
    }) => connectNode(accessToken, payload),
    onSuccess: (result) => {
      toast.success(`node connected ${result.node.id}`);
      queryClient.invalidateQueries({queryKey: ['nodes']});
      queryClient.invalidateQueries({queryKey: ['node-links']});
      queryClient.invalidateQueries({queryKey: ['node-transports']});
      quickConnectForm.reset({
        address: '',
        password: '',
        newPassword: '',
        name: '',
        mode: DEFAULT_MODE,
        scopeKey: '',
        parentNodeId: '',
        publicHost: '',
        publicPort: '',
        controlPlaneUrl: quickConnectForm.getValues('controlPlaneUrl')
      });
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const bootstrapMutation = useMutation({
    mutationFn: ({targetId, nodeName}: {targetId: string; nodeName: string}) => createBootstrapToken(accessToken, {targetType: DEFAULT_TARGET_TYPE, targetId, nodeName}),
    onSuccess: (result) => {
      toast.success('bootstrap token created');
      bootstrapForm.reset();
      bootstrapForm.setValue('targetId', '');
      queryClient.setQueryData(['latest-bootstrap-token'], result);
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const approveMutation = useMutation({
    mutationFn: (nodeID: string) => approveNode(accessToken, nodeID),
    onSuccess: () => {
      toast.success('node approved');
      queryClient.invalidateQueries({queryKey: ['pending-nodes']});
      queryClient.invalidateQueries({queryKey: ['nodes']});
      queryClient.invalidateQueries({queryKey: ['node-links']});
      queryClient.invalidateQueries({queryKey: ['node-transports']});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const updateNodeMutation = useMutation({
    mutationFn: (payload: {
      nodeID: string;
      name: string;
      mode: string;
      scopeKey: string;
      parentNodeId: string;
      publicHost: string;
      publicPort: number;
      enabled: boolean;
      status: string;
    }) => {
      const {nodeID, ...body} = payload;

      return updateNode(accessToken, nodeID, body);
    },
    onSuccess: () => {
      toast.success('node updated');
      queryClient.invalidateQueries({queryKey: ['nodes']});
      queryClient.invalidateQueries({queryKey: ['node-links']});
      queryClient.invalidateQueries({queryKey: ['node-transports']});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const deleteNodeMutation = useMutation({
    mutationFn: (nodeID: string) => deleteNode(accessToken, nodeID),
    onSuccess: () => {
      toast.success('node deleted');
      queryClient.invalidateQueries({queryKey: ['nodes']});
      queryClient.invalidateQueries({queryKey: ['node-links']});
      queryClient.invalidateQueries({queryKey: ['node-transports']});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const createNodeLinkMutation = useMutation({
    mutationFn: (payload: {sourceNodeId: string; targetNodeId: string; linkType: string; trustState: string}) =>
      createNodeLink(accessToken, payload),
    onSuccess: () => {
      toast.success('link created');
      queryClient.invalidateQueries({queryKey: ['node-links']});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const rejectNodeMutation = useMutation({
    mutationFn: ({nodeId, reason}: {nodeId: string; reason?: string}) =>
      rejectNode(accessToken, nodeId, reason),
    onSuccess: () => {
      toast.success('node rejected');
      queryClient.invalidateQueries({queryKey: ['pending-nodes']});
      queryClient.invalidateQueries({queryKey: ['nodes']});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  return {
    accessToken,
    nodeForm,
    quickConnectForm,
    bootstrapForm,
    nodesQuery,
    linksQuery,
    healthQuery,
    transportsQuery,
    pendingNodesQuery,
    unconsumedTokensQuery,
    latestToken: (queryClient.getQueryData(['latest-bootstrap-token']) as BootstrapToken | undefined) || null,
    createNode: createNodeMutation,
    quickConnect: quickConnectMutation,
    bootstrap: bootstrapMutation,
    approve: approveMutation,
    rejectNode: rejectNodeMutation,
    updateNode: updateNodeMutation,
    deleteNode: deleteNodeMutation,
    createNodeLink: createNodeLinkMutation
  };
}
