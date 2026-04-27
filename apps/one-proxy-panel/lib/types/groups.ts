import type { Account } from './auth';

export type Group = {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
};

export type GroupDetail = Group & {
  accounts: Account[];
  scopes: string[];
};
