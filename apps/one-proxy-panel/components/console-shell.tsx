'use client';

import Image from 'next/image';
import {
  BadgeCheck,
  ChevronRight,
  GitBranch,
  LayoutDashboard,
  Languages,
  Route,
  ShieldCheck,
  Shirt,
  Users,
  Waypoints,
  Workflow
} from 'lucide-react';
import {useLocale, useTranslations} from 'next-intl';
import {useTheme} from 'next-themes';
import {ChangeEvent, ReactNode} from 'react';
import {useQuery} from '@tanstack/react-query';

import {useAuth} from '@/components/auth-provider';
import {CapsuleSelect, CapsuleSelectGroup} from '@/components/common/capsule-select';
import {Link, usePathname, useRouter} from '@/i18n/navigation';
import {getNodeEnrollmentApprovals} from '@/lib/control-plane-api';

export function ConsoleShell({children}: {children: ReactNode}) {
  const t = useTranslations();
  const locale = useLocale();
  const pathname = usePathname();
  const router = useRouter();
  const {resolvedTheme, setTheme} = useTheme();
  const {session, logout} = useAuth();
  const accessToken = session?.accessToken || '';

  const approvalsQuery = useQuery({
    queryKey: ['node-approvals', accessToken],
    queryFn: () => getNodeEnrollmentApprovals(accessToken),
    enabled: !!accessToken,
    refetchInterval: 30000
  });

  const pendingCount = (approvalsQuery.data || []).filter((approval) => approval.status === 'pending').length;
  const navSections = [
    {
      key: 'overview',
      label: t('nav.overview'),
      href: '/',
      icon: LayoutDashboard,
      items: [{label: t('shell.summary'), href: '/'}]
    },
    {
      key: 'nodes',
      label: t('nav.nodes'),
      href: '/nodes/connect',
      icon: Workflow,
      items: [
        {label: t('shell.nodeConnect'), href: '/nodes/connect'},
        {label: t('shell.nodeManual'), href: '/nodes/manual'},
        {label: t('shell.nodeBootstrap'), href: '/nodes/bootstrap'},
        {label: t('shell.nodeApprovals'), href: '/nodes/approvals'},
        {label: t('shell.nodeRegistry'), href: '/nodes/registry'},
        {label: t('shell.nodeTopology'), href: '/nodes/topology'}
      ]
    },
    {
      key: 'onboarding',
      label: t('nav.onboarding'),
      href: '/onboarding',
      icon: Waypoints,
      items: [{label: t('shell.taskConsole'), href: '/onboarding'}]
    },
    {
      key: 'chains',
      label: t('nav.chains'),
      href: '/chains',
      icon: GitBranch,
      items: [{label: t('shell.chainStudio'), href: '/chains'}]
    },
    {
      key: 'routes',
      label: t('nav.routes'),
      href: '/routes',
      icon: Route,
      items: [{label: t('shell.routeBoard'), href: '/routes'}]
    },
    {
      key: 'health',
      label: t('nav.health'),
      href: '/health',
      icon: ShieldCheck,
      items: [{label: t('shell.healthBoard'), href: '/health'}]
    },
    {
      key: 'accounts',
      label: t('nav.accounts'),
      href: '/accounts/create',
      icon: Users,
      items: [
        {label: t('shell.accountCreate'), href: '/accounts/create'},
        {label: t('shell.accountList'), href: '/accounts/list'}
      ]
    },
    {
      key: 'certificates',
      label: t('nav.certificates'),
      href: '/certificates',
      icon: BadgeCheck,
      items: [{label: t('shell.certificateBoard'), href: '/certificates'}]
    }
  ];
  const activeSection =
    navSections.find((section) =>
      section.items.some((item) => (item.href === '/' ? pathname === '/' : pathname === item.href || pathname.startsWith(`${item.href}/`)))
    ) || navSections[0];
  const accountInitial = session?.account.account?.slice(0, 1).toUpperCase() || 'U';
  const themeValue = resolvedTheme === 'light' ? 'light' : 'dark';

  const handleThemeChange = (event: ChangeEvent<HTMLSelectElement>) => {
    setTheme(event.target.value);
  };

  const handleLocaleChange = (event: ChangeEvent<HTMLSelectElement>) => {
    router.replace(pathname, {locale: event.target.value});
  };

  return (
    <div className="console-shell">
      <header className="console-topbar">
        <div className="console-topbar-brand">
          <div className="console-topbar-favicon">
            <Image alt="One Proxy favicon" height={34} priority src="/favicon.svg" width={34} />
          </div>
          <div className="console-topbar-wordmark">
            <h1>{t('shell.product')}</h1>
          </div>
        </div>

        <div className="console-topbar-actions">
          <CapsuleSelectGroup>
            <CapsuleSelect aria-label="Theme" icon={<Shirt size={16} />} onChange={handleThemeChange} value={themeValue}>
              <option value="dark">{t('shell.themeDark')}</option>
              <option value="light">{t('shell.themeLight')}</option>
            </CapsuleSelect>

            <CapsuleSelect aria-label="Language" icon={<Languages size={16} />} onChange={handleLocaleChange} value={locale}>
              <option value="zh">{t('shell.localeZh')}</option>
              <option value="en">{t('shell.localeEn')}</option>
            </CapsuleSelect>
          </CapsuleSelectGroup>

          <div className="console-user-card">
            <div className="console-user-avatar">{accountInitial}</div>
            <div className="console-user-copy">
              <strong>{session?.account.account || t('shell.name')}</strong>
              <span>{session?.account.role || t('shell.tagline')}</span>
            </div>
            {session ? (
              <button className="secondary-button" onClick={() => void logout()} type="button">
                {t('auth.logout')}
              </button>
            ) : null}
          </div>
        </div>
      </header>

      <div className="console-workspace">
        <aside className="console-rail">
          <div className="brand-panel">
            <div className="brand-mark">
              <Image alt="One Proxy favicon" height={56} priority src="/favicon.svg" width={56} />
            </div>
            <div className="brand-copy-block">
              <p className="brand-kicker">{t('shell.product')}</p>
              <h2>{t('shell.name')}</h2>
              <p className="brand-copy">{t('shell.tagline')}</p>
            </div>
          </div>

          <nav className="nav-panel">
            {navSections.map((section) => {
              const sectionActive = section.key === activeSection.key;
              const SectionIcon = section.icon;

              return (
                <div className={`menu-group${sectionActive ? ' is-active' : ''}`} key={section.key}>
                  <Link className={`menu-link${sectionActive ? ' is-active' : ''}`} href={section.href}>
                    <span className="menu-link-main">
                      <SectionIcon size={16} />
                      <span>{section.label}</span>
                    </span>
                    <ChevronRight className={`menu-link-arrow${sectionActive ? ' is-open' : ''}`} size={14} />
                  </Link>
                  {sectionActive ? (
                    <div className="submenu-list">
                      {section.items.map((item) => {
                        const itemActive = item.href === '/' ? pathname === '/' : pathname === item.href || pathname.startsWith(`${item.href}/`);
                        const showBadge = item.href === '/nodes/approvals' && pendingCount > 0;

                        return (
                          <Link className={`submenu-link${itemActive ? ' is-active' : ''}`} href={item.href} key={item.href}>
                            <span>{item.label}</span>
                            {showBadge && <span className="badge is-warn">{pendingCount}</span>}
                          </Link>
                        );
                      })}
                    </div>
                  ) : null}
                </div>
              );
            })}
          </nav>
        </aside>

        <main className="console-main">{children}</main>
      </div>
    </div>
  );
}
