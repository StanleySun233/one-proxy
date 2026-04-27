'use client';

import {Node, NodeLink, NodeTransport} from '@/lib/control-plane-types';
import {formatISODateTime} from '@/lib/presentation';

import {describeNodeName, transportBadgeClassName} from './node-utils';

export function NodeLinkCard({
  link,
  nodesByID,
  transports,
  reverseWsType
}: {
  link: NodeLink;
  nodesByID: Map<string, Node>;
  transports: NodeTransport[];
  reverseWsType: string;
}) {
  const childTunnel = transports.find(
    (transport) =>
      transport.nodeId === link.targetNodeId &&
      transport.parentNodeId === link.sourceNodeId &&
      transport.transportType === reverseWsType
  );

  return (
    <article className="node-record-card">
      <div className="stack-head">
        <strong>
          {describeNodeName(link.sourceNodeId, nodesByID)} → {describeNodeName(link.targetNodeId, nodesByID)}
        </strong>
        <span className="badge">{link.trustState}</span>
      </div>
      <div className="nodes-ledger-meta">
        <span>{link.linkType}</span>
      </div>
      {childTunnel ? (
        <div className="node-approval-meta">
          <span className={transportBadgeClassName(childTunnel.status)}>{childTunnel.status}</span>
          <span className="mono">{childTunnel.transportType}</span>
          <span className="muted-text">{childTunnel.lastHeartbeatAt ? formatISODateTime(childTunnel.lastHeartbeatAt) : 'heartbeat: never'}</span>
        </div>
      ) : (
        <span className="badge is-neutral">no active child tunnel</span>
      )}
      <span className="muted-text">source: {link.sourceNodeId}</span>
      <span className="muted-text">target: {link.targetNodeId}</span>
      <span className="mono">{link.id}</span>
    </article>
  );
}
