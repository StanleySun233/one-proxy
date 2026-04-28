'use client';

import {useQuery} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {PageHero} from '@/components/page-hero';
import {fetchEnums, getCertificates} from '@/lib/api';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

export default function CertificatesPage() {
  const t = useTranslations('pages');
  const certT = useTranslations('certificates');
  const common = useTranslations('common');
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
        <PageHero eyebrow={certT('title')} title={t('certificatesTitle')} description={t('certificatesDesc')} />
        <article className="panel-card">
          {certificatesQuery.isPending ? (
            <AsyncState detail={certT('loadingDetail')} title={certT('loading')} />
          ) : certificatesQuery.isError ? (
            <AsyncState actionLabel={common('retry')} detail={formatControlPlaneError(certificatesQuery.error)} onAction={() => void certificatesQuery.refetch()} title={certT('failedToLoad')} />
          ) : certificates.length === 0 ? (
            <AsyncState detail={certT('emptyDetail')} title={certT('emptyTitle')} />
          ) : (
            <div className="table-card">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>{certT('owner')}</th>
                    <th>{certT('type')}</th>
                    <th>{certT('provider')}</th>
                    <th>{certT('status')}</th>
                    <th>{certT('notAfter')}</th>
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
