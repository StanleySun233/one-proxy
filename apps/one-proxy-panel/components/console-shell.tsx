'use client';

import {
  Building2,
  ChevronRight,
  GitBranch,
  LayoutDashboard,
  Languages,
  ShieldCheck,
  Shirt,
  Users,
  Workflow
} from 'lucide-react';
import {useLocale, useTranslations} from 'next-intl';
import {useTheme} from 'next-themes';
import {MouseEvent, ReactNode, useEffect, useState} from 'react';
import {useQuery} from '@tanstack/react-query';

import {useAuth} from '@/components/auth-provider';
import {CapsuleSelect, CapsuleSelectGroup} from '@/components/common/capsule-select';
import {Link, usePathname, useRouter} from '@/i18n/navigation';
import {getPendingNodes} from '@/lib/api';

export function ConsoleShell({children}: {children: ReactNode}) {
  const t = useTranslations();
  const locale = useLocale();
  const pathname = usePathname();
  const router = useRouter();
  const {resolvedTheme, setTheme} = useTheme();
  const {session, tenantMemberships, activeTenant, switchTenant, logout} = useAuth();
  const accessToken = session?.accessToken || '';
  const tenantOptions = tenantMemberships.map((membership) => ({
    value: membership.tenantId,
    label: membership.tenantName
  }));
  const showTenantSwitcher = tenantMemberships.length > 1;
  const tenantSelectOptions = activeTenant ? tenantOptions : [{value: '', label: t('shell.tenantPlaceholder')}, ...tenantOptions];

  const pendingQuery = useQuery({
    queryKey: ['pending-nodes', accessToken],
    queryFn: () => getPendingNodes(accessToken),
    enabled: !!accessToken,
    refetchInterval: 30000
  });

  const pendingCount = (pendingQuery.data || []).length;
  const navSections = [
    {
      key: 'chains',
      label: t('nav.chains'),
      href: '/chains/scopes',
      icon: GitBranch,
      items: [
        {label: t('shell.scopeBoard'), href: '/chains/scopes'},
        {label: t('shell.nodeTopology'), href: '/chains/topology'},
        {label: t('shell.chainStudio'), href: '/chains/studio'},
        {label: t('shell.routeBoard'), href: '/chains/routes'}
      ]
    },
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
      href: '/nodes/bootstrap',
      icon: Workflow,
      items: [
        {label: t('shell.nodeBootstrap'), href: '/nodes/bootstrap'},
        {label: t('shell.nodeApprovals'), href: '/nodes/approvals'},
        {label: t('shell.nodeRegistry'), href: '/nodes/registry'}
      ]
    },
    {
      key: 'health',
      label: t('nav.health'),
      href: '/health/overview',
      icon: ShieldCheck,
      items: [
        {label: t('shell.healthOverview'), href: '/health/overview'},
        {label: t('shell.healthHeartbeat'), href: '/health/heartbeat'}
      ]
    },
    {
      key: 'accounts',
      label: t('nav.accounts'),
      href: '/accounts/create',
      icon: Users,
      items: [
        {label: t('shell.accountCreate'), href: '/accounts/create'},
        {label: t('shell.accountList'), href: '/accounts/list'},
        {label: t('shell.groupList'), href: '/accounts/groups'}
      ]
    }
  ];
  const activeSection =
    navSections.find((section) =>
      section.items.some((item) => (item.href === '/' ? pathname === '/' : pathname === item.href || pathname.startsWith(`${item.href}/`)))
    ) || navSections[0];

  const [collapsedSection, setCollapsedSection] = useState<string | null>(null);

  useEffect(() => {
    setCollapsedSection(null);
  }, [pathname]);

  const expandedKey = collapsedSection === activeSection.key ? null : activeSection.key;

  const handleSectionClick = (e: MouseEvent, sectionKey: string) => {
    if (collapsedSection === sectionKey) {
      setCollapsedSection(null);
    } else if (expandedKey === sectionKey) {
      e.preventDefault();
      setCollapsedSection(sectionKey);
    }
  };

  const accountInitial = session?.account.account?.slice(0, 1).toUpperCase() || 'U';
  const themeValue = resolvedTheme === 'light' ? 'light' : 'dark';

  const handleThemeChange = (value: string) => {
    setTheme(value);
  };

  const handleLocaleChange = (value: string) => {
    router.replace(pathname, {locale: value});
  };

  const handleTenantChange = (value: string) => {
    if (value) {
      switchTenant(value);
    }
  };

  return (
    <div className="console-shell">
      <header className="console-topbar">
        <div className="console-topbar-brand">
          <span className="console-topbar-favicon">
            <img alt="" src="/favicon.svg" />
          </span>
          <span className="console-topbar-wordmark">One Proxy</span>
        </div>

        <div className="console-topbar-actions">
          <CapsuleSelectGroup>
            {showTenantSwitcher ? (
              <CapsuleSelect
                aria-label={t('shell.tenantLabel')}
                icon={<Building2 size={16} />}
                onChange={handleTenantChange}
                options={tenantSelectOptions}
                value={activeTenant?.tenantId || ''}
              />
            ) : null}

            <CapsuleSelect
              aria-label={t('shell.themeLabel')}
              icon={<Shirt size={16} />}
              onChange={handleThemeChange}
              options={[
                {value: 'dark', label: t('shell.themeDark')},
                {value: 'light', label: t('shell.themeLight')}
              ]}
              value={themeValue}
            />

            <CapsuleSelect
              aria-label={t('shell.languageLabel')}
              icon={<Languages size={16} />}
              onChange={handleLocaleChange}
              options={[
                {value: 'zh', label: t('shell.localeZh')},
                {value: 'en', label: t('shell.localeEn')}
              ]}
              value={locale}
            />
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
          <nav className="nav-panel">
            {navSections.map((section) => {
              const sectionActive = section.key === expandedKey;
              const SectionIcon = section.icon;

              return (
                <div className={`menu-group${sectionActive ? ' is-active' : ''}`} key={section.key}>
                  <Link
                    className={`menu-link${sectionActive ? ' is-active' : ''}`}
                    href={section.href}
                    onClick={(e) => handleSectionClick(e, section.key)}
                  >
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
