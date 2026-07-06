'use client';

import {
  Building2,
  ChevronRight,
  FileSearch,
  GitBranch,
  Languages,
  LayoutDashboard,
  Monitor,
  ShieldCheck,
  Shirt,
  Users,
  Workflow
} from 'lucide-react';
import {useLocale, useTranslations} from 'next-intl';
import {useTheme} from 'next-themes';
import {MouseEvent, ReactNode, useEffect, useState} from 'react';
import {useQuery, useQueryClient} from '@tanstack/react-query';

import {AccountSettingsPopover} from '@/components/account-settings-popover';
import {useAuth} from '@/components/auth-provider';
import {CapsuleSelect, CapsuleSelectGroup} from '@/components/common/capsule-select';
import {Link, usePathname, useRouter} from '@/i18n/navigation';
import {getPendingNodes, getTenants} from '@/lib/api';

export function ConsoleShell({children}: {children: ReactNode}) {
  const t = useTranslations();
  const locale = useLocale();
  const pathname = usePathname();
  const router = useRouter();
  const {resolvedTheme, setTheme} = useTheme();
  const {session, tenantMemberships, activeTenant, switchTenant} = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';
  const activeTenantId = session?.activeTenantId || null;
  const isSuperAdmin = session?.account.role === 'super_admin';
  const superAdminTenantsQuery = useQuery({
    queryKey: ['tenants', accessToken, 'shell'],
    queryFn: () => getTenants(accessToken),
    enabled: !!accessToken && isSuperAdmin
  });
  const tenantOptions = (isSuperAdmin ? superAdminTenantsQuery.data || [] : tenantMemberships).map((tenant) => ({
    value: 'tenantId' in tenant ? tenant.tenantId : tenant.id,
    label: 'tenantName' in tenant ? tenant.tenantName : tenant.name
  }));
  const selectedTenantOption = tenantOptions.find((option) => option.value === activeTenantId);
  const tenantSelectOptions = selectedTenantOption ? tenantOptions : [{value: '', label: t('shell.tenantPlaceholder')}, ...tenantOptions];
  const showTenantSwitcher = tenantOptions.length > 0;
  const isTenantAdmin = tenantMemberships.some((membership) => membership.role === 'tenant_admin');
  const pendingQuery = useQuery({
    queryKey: ['pending-nodes', accessToken, activeTenantId],
    queryFn: () => getPendingNodes(accessToken, activeTenantId),
    enabled: !!accessToken && (!!activeTenantId || tenantMemberships.length === 0),
    refetchInterval: 30000
  });
  const pendingCount = (pendingQuery.data || []).length;
  const navSections = [
    {
      key: 'overview',
      label: t('nav.overview'),
      href: '/overview/dashboard',
      icon: LayoutDashboard,
      items: [
        {label: t('shell.overviewDashboard'), href: '/overview/dashboard'}
      ]
    },
    {
      key: 'proxy',
      label: t('nav.proxy'),
      href: '/proxy/scopes',
      icon: GitBranch,
      items: [
        {label: t('shell.scopeBoard'), href: '/proxy/scopes'},
        {label: t('shell.nodeTopology'), href: '/proxy/topology'},
        {label: t('shell.chainStudio'), href: '/proxy/studio'},
        {label: t('shell.routeGroups'), href: '/proxy/routes/groups'},
        {label: t('shell.routeRules'), href: '/proxy/routes/rules'},
        {label: t('shell.routePublish'), href: '/proxy/routes/publish'},
        {label: t('shell.accessPaths'), href: '/proxy/network'}
      ]
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
      key: 'remote',
      label: t('nav.remote'),
      href: '/remote/ssh',
      icon: Monitor,
      items: [
        {label: t('shell.remoteSSH'), href: '/remote/ssh'},
        {label: t('shell.remoteRDP'), href: '/remote/rdp'}
      ]
    },
    {
      key: 'health',
      label: t('nav.health'),
      href: '/health/overview',
      icon: ShieldCheck,
      items: [
        {label: t('shell.healthOverview'), href: '/health/overview'},
        {label: t('shell.healthHeartbeat'), href: '/health/heartbeat'},
        {label: t('shell.healthSLA'), href: '/health/sla'}
      ]
    },
    {
      key: 'audit',
      label: t('nav.audit'),
      href: '/audit/dashboard',
      icon: FileSearch,
      items: [
        {label: t('shell.auditDashboard'), href: '/audit/dashboard'},
        {label: t('shell.auditNetwork'), href: '/audit/network'},
        {label: t('shell.auditBusiness'), href: '/audit/business'}
      ]
    },
    ...(isSuperAdmin || isTenantAdmin ? [{
      key: 'accounts',
      label: t('nav.accounts'),
      href: isSuperAdmin ? '/accounts/list' : '/accounts/tenants',
      icon: Users,
      items: [
        ...(isSuperAdmin ? [{label: t('shell.accountList'), href: '/accounts/list'}] : []),
        {label: t('shell.tenantList'), href: '/accounts/tenants'}
      ]
    }] : [])
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
  const accountRoleLabel = activeTenant?.role || session?.account.role || t('shell.tagline');
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
      queryClient.clear();
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
          {showTenantSwitcher ? (
            <CapsuleSelect
              aria-label={t('shell.tenantLabel')}
              icon={<Building2 size={16} />}
              onChange={handleTenantChange}
              options={tenantSelectOptions}
              value={activeTenantId || ''}
            />
          ) : null}
        </div>

        <div className="console-topbar-actions">
          <CapsuleSelectGroup>
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

          <AccountSettingsPopover accountInitial={accountInitial} accountRoleLabel={accountRoleLabel} />
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
