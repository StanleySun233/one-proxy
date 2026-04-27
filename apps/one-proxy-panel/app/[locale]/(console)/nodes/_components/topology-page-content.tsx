'use client';

import {useMemo} from 'react';
import {useQuery} from '@tanstack/react-query';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {fetchEnums} from '@/lib/api';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

import {useNodeConsole} from './use-node-console';
import {describeNodeName, transportBadgeClassName} from './node-utils';
import {CreateNodeLinkForm} from './create-node-link-form';
import {NodeLinkCard} from './node-link-card';

export function NodeTopologyPageContent() {
  const nodeConsole = useNodeConsole();
  const nodes = nodeConsole.nodesQuery.data || [];
  const links = nodeConsole.linksQuery.data || [];
  const transports = nodeConsole.transportsQuery.data || [];
  const {data: enums} = useQuery({queryKey: ['enums'], queryFn: () => fetchEnums()});
  const nodesByID = useMemo(() => new Map(nodes.map((node) => [node.id, node])), [nodes]);
  // Derive enum value references from the enums object
  const transportTypeKeys = Object.keys(enums?.transport_type || {});
  const PUBLIC_HTTP = transportTypeKeys.find(k => k === 'public_http') || 'public_http';
  const PUBLIC_HTTPS = transportTypeKeys.find(k => k === 'public_https') || 'public_https';
  const REVERSE_WS_PARENT = transportTypeKeys.find(k => k === 'reverse_ws_parent') || 'reverse_ws_parent';
  const CONNECTED = Object.keys(enums?.transport_status || {}).find(k => k === 'connected') || 'connected';
  const LINK_TYPE_RELAY = Object.keys(enums?.link_type || {}).find(k => k === 'relay') || 'relay';
  const TRUST_STATE_TRUSTED = Object.keys(enums?.trust_state || {}).find(k => k === 'trusted') || 'trusted';
  const transportSummary = useMemo(
    () => ({
      publicEndpoints: transports.filter((item) => item.transportType === PUBLIC_HTTP || item.transportType === PUBLIC_HTTPS).length,
      reverseConnected: transports.filter((item) => item.transportType === REVERSE_WS_PARENT && item.status === CONNECTED).length,
      reverseBlocked: transports.filter((item) => item.transportType === REVERSE_WS_PARENT && item.status !== CONNECTED).length
    }),
    [transports]
  );

  return (
    <AuthGate>
      <div className="page-stack">
        <section className="metrics-grid">
          <article className="metric-card panel-card">
            <span className="metric-label">Public transports</span>
            <strong>{transportSummary.publicEndpoints}</strong>
            <span className="metric-foot">Directly reachable node entrypoints synthesized from public endpoint records.</span>
          </article>
          <article className="metric-card panel-card soft-card">
            <span className="metric-label">Reverse tunnels up</span>
            <strong>{transportSummary.reverseConnected}</strong>
            <span className="metric-foot">Parent-child `reverse_ws_parent` sessions currently reporting connected.</span>
          </article>
          <article className="metric-card panel-card warm-card">
            <span className="metric-label">Reverse tunnels blocked</span>
            <strong>{transportSummary.reverseBlocked}</strong>
            <span className="metric-foot">Configured reverse tunnels that are not currently connected.</span>
          </article>
        </section>

        <section className="panel-card">
          <div className="panel-toolbar">
            <div>
              <p className="section-kicker">Topology</p>
              <h3>Relay relationships</h3>
              <p className="section-copy">Inspect parent-child links together with live transport state to verify a → b → c tunnel overlays.</p>
            </div>
            <span className="badge">{links.length}</span>
          </div>
          <CreateNodeLinkForm
            nodes={nodes}
            pending={nodeConsole.createNodeLink.isPending}
            onSubmit={(payload) => nodeConsole.createNodeLink.mutate(payload)}
            defaultLinkType={LINK_TYPE_RELAY}
            defaultTrustState={TRUST_STATE_TRUSTED}
          />
          {nodeConsole.linksQuery.isPending || nodeConsole.nodesQuery.isPending || nodeConsole.transportsQuery.isPending ? (
            <AsyncState detail="Loading" title="Loading topology links" />
          ) : nodeConsole.nodesQuery.error ? (
            <AsyncState
              actionLabel="Retry"
              detail={formatControlPlaneError(nodeConsole.nodesQuery.error)}
              onAction={() => void nodeConsole.nodesQuery.refetch()}
              title="Failed to load node registry"
            />
          ) : nodeConsole.linksQuery.error ? (
            <AsyncState
              actionLabel="Retry"
              detail={formatControlPlaneError(nodeConsole.linksQuery.error)}
              onAction={() => void nodeConsole.linksQuery.refetch()}
              title="Failed to load topology links"
            />
          ) : nodeConsole.transportsQuery.error ? (
            <AsyncState
              actionLabel="Retry"
              detail={formatControlPlaneError(nodeConsole.transportsQuery.error)}
              onAction={() => void nodeConsole.transportsQuery.refetch()}
              title="Failed to load transport registry"
            />
          ) : links.length === 0 ? (
            <AsyncState detail="Links appear after parent-child registration or relay trust setup." title="Empty" />
          ) : (
            <div className="topology-stack">
              <div className="nodes-link-grid">
                {links.map((link) => (
                  <NodeLinkCard key={link.id} link={link} nodesByID={nodesByID} transports={transports} reverseWsType={REVERSE_WS_PARENT} />
                ))}
              </div>
              <div className="table-card">
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>Node</th>
                      <th>Type</th>
                      <th>Direction</th>
                      <th>Status</th>
                      <th>Address</th>
                      <th>Parent</th>
                      <th>Heartbeat</th>
                    </tr>
                  </thead>
                  <tbody>
                    {transports.length === 0 ? (
                      <tr>
                        <td className="muted-text" colSpan={7}>
                          No runtime transports have been reported yet.
                        </td>
                      </tr>
                    ) : (
                      transports.map((transport) => (
                        <tr key={transport.id}>
                          <td>{describeNodeName(transport.nodeId, nodesByID) || transport.nodeId}</td>
                          <td className="mono">{transport.transportType}</td>
                          <td>{transport.direction}</td>
                          <td>
                            <span className={transportBadgeClassName(transport.status, enums)}>{transport.status}</span>
                          </td>
                          <td className="mono">{transport.address}</td>
                          <td>{describeNodeName(transport.parentNodeId, nodesByID) || <span className="muted-text">root</span>}</td>
                          <td className="mono">{transport.lastHeartbeatAt ? formatISODateTime(transport.lastHeartbeatAt) : <span className="muted-text">never</span>}</td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </section>
      </div>
    </AuthGate>
  );
}
