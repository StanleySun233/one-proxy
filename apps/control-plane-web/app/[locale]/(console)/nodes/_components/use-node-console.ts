'use client';

import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {useForm} from 'react-hook-form';
import {toast} from 'sonner';

import {useAuth} from '@/components/auth-provider';
import {approveNode, connectNode, createBootstrapToken, createNode, deleteNode, getNodeHealth, getNodeLinks, getNodes, updateNode} from '@/lib/control-plane-api';
import {formatControlPlaneError} from '@/lib/presentation';

import {BootstrapFormValues, NodeFormValues, QuickConnectFormValues} from './types';

export function useNodeConsole() {
  const {session} = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';

  const nodeForm = useForm<NodeFormValues>({
    defaultValues: {
      name: '',
      mode: 'relay',
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
      mode: 'relay',
      scopeKey: '',
      parentNodeId: '',
      publicHost: '',
      publicPort: '',
      controlPlaneUrl: typeof window === 'undefined' ? 'http://127.0.0.1:2887' : `${window.location.protocol}//${window.location.hostname}:2887`
    }
  });

  const bootstrapForm = useForm<BootstrapFormValues>({
    defaultValues: {
      targetId: ''
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
      quickConnectForm.reset({
        address: '',
        password: '',
        newPassword: '',
        name: '',
        mode: 'relay',
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
    mutationFn: (targetId: string) => createBootstrapToken(accessToken, {targetType: 'node', targetId}),
    onSuccess: (result) => {
      toast.success('bootstrap token created');
      bootstrapForm.reset();
      bootstrapForm.setValue('targetId', '');
      queryClient.setQueryData(['latest-bootstrap-token'], result.token);
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const approveMutation = useMutation({
    mutationFn: (nodeID: string) => approveNode(accessToken, nodeID),
    onSuccess: () => {
      toast.success('node approved');
      queryClient.invalidateQueries({queryKey: ['nodes']});
      queryClient.invalidateQueries({queryKey: ['node-links']});
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
    latestToken: (queryClient.getQueryData(['latest-bootstrap-token']) as string | undefined) || '',
    createNode: createNodeMutation,
    quickConnect: quickConnectMutation,
    bootstrap: bootstrapMutation,
    approve: approveMutation,
    updateNode: updateNodeMutation,
    deleteNode: deleteNodeMutation
  };
}
