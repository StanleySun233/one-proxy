export function validateMatchValue(matchType: string, value: string, t: (key: string) => string): string | true {
  const trimmed = value.trim();
  if (!trimmed) return t('matchValueRequired');

  switch (matchType) {
    case 'domain':
      if (!/^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*$/i.test(trimmed)) {
        return t('invalidDomain');
      }
      break;
    case 'domain_suffix':
      if (!/^\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*$/i.test(trimmed)) {
        return t('invalidDomainSuffix');
      }
      break;
    case 'ip_cidr':
      if (!/^(\d{1,3}\.){3}\d{1,3}\/\d{1,2}$/.test(trimmed)) {
        return t('invalidCidr');
      }
      break;
    case 'ip_range':
      if (!/^(\d{1,3}\.){3}\d{1,3}-(\d{1,3}\.){3}\d{1,3}$/.test(trimmed)) {
        return t('invalidIpRange');
      }
      break;
    case 'port':
      const port = Number(trimmed);
      if (isNaN(port) || port < 1 || port > 65535) {
        return t('invalidPort');
      }
      break;
    case 'url_regex':
      try {
        new RegExp(trimmed);
      } catch {
        return t('invalidRegex');
      }
      break;
  }
  return true;
}
