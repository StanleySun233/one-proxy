import type {AccountRole, AccountStatus} from './common';

export type Account = {
  id: string;
  account: string;
  role: AccountRole;
  status: AccountStatus;
  mustRotatePassword: boolean;
};

export type LoginResult = {
  account: Account;
  accessToken: string;
  refreshToken: string;
  expiresAt: string;
  mustRotatePassword: boolean;
};
