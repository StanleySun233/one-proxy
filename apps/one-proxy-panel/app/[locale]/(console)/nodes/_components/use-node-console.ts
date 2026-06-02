'use client';

import {useState} from 'react';
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {useForm} from 'react-hook-form';
import {toast} from 'sonner';

import {useAuth} from '@/components/auth-provider';
import {BootstrapToken} from '@/lib/types';
import {
  approveNode,
  createBootstrapToken,
  createNodeLink,
  deleteBootstrapToken,
  deleteNode,
  deleteNodeLink,
  fetchEnums,
  getNodeHealth,
  getNodeLinks,
  getNodes,
  getNodeTransports,
  getPendingNodes,
  getScopes,
  getUnconsumedBootstrapTokens,
  rejectNode,
  updateNode,
  updateNodeLink
} from '@/lib/api';
import {formatControlPlaneError} from '@/lib/presentation';

import {BootstrapFormValues} from './types';

export function useNodeConsole() {
  const {session, activeTenant} = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';
  const activeTenantId = session?.activeTenantId || null;
  const canWrite = session?.account.role === 'super_admin' || activeTenant?.role === 'tenant_admin';

  const {data: enums} = useQuery({queryKey: ['enums'], queryFn: () => fetchEnums()});
  const nodeModeKeys = Object.keys(enums?.node_mode || {});
  const DEFAULT_BOOTSTRAP_MODE = nodeModeKeys.find(k => k === 'edge') || 'edge';
  const bootstrapTargetKeys = Object.keys(enums?.bootstrap_target_type || {});
  const DEFAULT_TARGET_TYPE = bootstrapTargetKeys.find(k => k === 'node') || 'node';

  const [latestToken, setLatestToken] = useState<BootstrapToken | null>(null);

  const bootstrapForm = useForm<BootstrapFormValues>({
    defaultValues: {
      targetId: '',
      nodeName: '',
      nodeMode: DEFAULT_BOOTSTRAP_MODE,
      scopeKey: '',
      parentNodeId: '',
      parentReachableUrl: '',
      publicHost: '',
      publicPort: '2988'
    }
  });

  const nodesQuery = useQuery({
    queryKey: ['nodes', accessToken, activeTenantId],
    queryFn: () => getNodes(accessToken),
    enabled: !!accessToken
  });

  const scopesQuery = useQuery({
    queryKey: ['scopes', accessToken, activeTenantId],
    queryFn: () => getScopes(accessToken),
    enabled: !!accessToken
  });

  const linksQuery = useQuery({
    queryKey: ['node-links', accessToken, activeTenantId],
    queryFn: () => getNodeLinks(accessToken),
    enabled: !!accessToken
  });

  const healthQuery = useQuery({
    queryKey: ['node-health', accessToken, activeTenantId],
    queryFn: () => getNodeHealth(accessToken),
    enabled: !!accessToken,
    refetchInterval: 5000
  });

  const transportsQuery = useQuery({
    queryKey: ['node-transports', accessToken, activeTenantId],
    queryFn: () => getNodeTransports(accessToken),
    enabled: !!accessToken,
    refetchInterval: 5000
  });

  const pendingNodesQuery = useQuery({
    queryKey: ['pending-nodes', accessToken, activeTenantId],
    queryFn: () => getPendingNodes(accessToken),
    enabled: !!accessToken,
    refetchInterval: 30000
  });

  const unconsumedTokensQuery = useQuery({
    queryKey: ['unconsumed-bootstrap-tokens', accessToken, activeTenantId],
    queryFn: () => getUnconsumedBootstrapTokens(accessToken),
    enabled: !!accessToken,
    refetchInterval: 30000
  });

  const bootstrapMutation = useMutation({
    mutationFn: (payload: {targetId: string; nodeName: string; nodeMode: string; scopeKey: string; parentNodeId: string; parentReachableUrl: string; publicHost: string; publicPort: number}) =>
      createBootstrapToken(accessToken, {
        targetType: DEFAULT_TARGET_TYPE,
        targetId: payload.targetId,
        nodeName: payload.nodeName,
        nodeMode: payload.nodeMode,
        scopeKey: payload.scopeKey,
        parentNodeId: payload.parentNodeId,
        publicHost: payload.publicHost,
        publicPort: payload.publicPort
      }),
    onSuccess: (result, variables) => {
      toast.success('bootstrap token created');
      bootstrapForm.reset({
        targetId: '',
        nodeName: '',
        nodeMode: variables.nodeMode,
        scopeKey: variables.scopeKey,
        parentNodeId: variables.parentNodeId,
        parentReachableUrl: variables.parentReachableUrl,
        publicHost: variables.publicHost,
        publicPort: variables.publicPort > 0 ? String(variables.publicPort) : '2988'
      });
      bootstrapForm.setValue('targetId', '');
      setLatestToken(result);
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
      queryClient.invalidateQueries({queryKey: ['pending-nodes']});
      queryClient.invalidateQueries({queryKey: ['node-links']});
      queryClient.invalidateQueries({queryKey: ['node-transports']});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const deleteBootstrapTokenMutation = useMutation({
    mutationFn: (tokenID: string) => deleteBootstrapToken(accessToken, tokenID),
    onSuccess: () => {
      toast.success('bootstrap token deleted');
      queryClient.invalidateQueries({queryKey: ['unconsumed-bootstrap-tokens']});
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

  const updateNodeLinkMutation = useMutation({
    mutationFn: (payload: {linkID: string; sourceNodeId: string; targetNodeId: string; linkType: string; trustState: string}) => {
      const {linkID, ...body} = payload;

      return updateNodeLink(accessToken, linkID, body);
    },
    onSuccess: () => {
      toast.success('link updated');
      queryClient.invalidateQueries({queryKey: ['node-links']});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const deleteNodeLinkMutation = useMutation({
    mutationFn: (linkID: string) => deleteNodeLink(accessToken, linkID),
    onSuccess: () => {
      toast.success('link deleted');
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
    activeTenantId,
    canWrite,
    bootstrapForm,
    nodesQuery,
    scopesQuery,
    linksQuery,
    healthQuery,
    transportsQuery,
    pendingNodesQuery,
    unconsumedTokensQuery,
    latestToken,
    bootstrap: bootstrapMutation,
    approve: approveMutation,
    rejectNode: rejectNodeMutation,
    updateNode: updateNodeMutation,
    deleteNode: deleteNodeMutation,
    deleteBootstrapToken: deleteBootstrapTokenMutation,
    createNodeLink: createNodeLinkMutation,
    updateNodeLink: updateNodeLinkMutation,
    deleteNodeLink: deleteNodeLinkMutation
  };
}
