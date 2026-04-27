export type Account = {
  id: string;
  account: string;
  role: string;
  status: string;
  mustRotatePassword: boolean;
};

export type LoginResult = {
  account: Account;
  accessToken: string;
  refreshToken: string;
  expiresAt: string;
  mustRotatePassword: boolean;
};
