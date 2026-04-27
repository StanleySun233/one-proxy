'use client';

import {useQuery} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {PageHero} from '@/components/page-hero';
import {fetchEnums, getCertificates} from '@/lib/control-plane-api';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

export default function CertificatesPage() {
  const t = useTranslations('pages');
  const {session} = useAuth();
  const accessToken = session?.accessToken || '';
  const certificatesQuery = useQuery({
    queryKey: ['certificates', accessToken],
    queryFn: () => getCertificates(accessToken),
    enabled: !!accessToken
  });
  const enumsQuery = useQuery({queryKey: ['enums'], queryFn: () => fetchEnums()});
  const enums = enumsQuery.data;
  const certificates = certificatesQuery.data || [];

  return (
    <AuthGate>
      <div className="page-stack">
        <PageHero eyebrow="Certificates" title={t('certificatesTitle')} description={t('certificatesDesc')} />
        <article className="panel-card">
          {certificatesQuery.isPending ? (
            <AsyncState detail="Certificate inventory is loading." title="Loading certificates" />
          ) : certificatesQuery.isError ? (
            <AsyncState actionLabel="Retry" detail={formatControlPlaneError(certificatesQuery.error)} onAction={() => void certificatesQuery.refetch()} title="Failed to load certificates" />
          ) : certificates.length === 0 ? (
            <AsyncState detail="Certificate rows will appear after node or panel certificates are registered." title="No certificates yet" />
          ) : (
            <div className="table-card">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>Owner</th>
                    <th>Type</th>
                    <th>Provider</th>
                    <th>Status</th>
                    <th>Not After</th>
                  </tr>
                </thead>
                <tbody>
                  {certificates.map((cert) => (
                    <tr key={cert.id}>
                      <td>{cert.ownerId}</td>
                      <td>{cert.certType}</td>
                      <td>{cert.provider}</td>
                      <td>
                        <span className={`badge ${enums?.cert_status?.[cert.status]?.meta?.className || 'is-neutral'}`}>{cert.status}</span>
                      </td>
                      <td className="mono">{formatISODateTime(cert.notAfter, '-')}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </article>
      </div>
    </AuthGate>
  );
}
