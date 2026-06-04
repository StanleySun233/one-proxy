import type {ReactNode} from 'react';

type NameTagKind = 'node' | 'chain' | 'route' | 'scope' | 'account' | 'group' | 'certificate';

export function NameTag({kind, children}: {kind: NameTagKind; children: ReactNode}) {
  return <span className={`name-tag is-${kind}`}>{children}</span>;
}
