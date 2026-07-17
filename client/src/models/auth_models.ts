export interface AuthForm {
  display_name?: string;
  email: string;
  password: string;
  password_confirmation?: string;
  remember_me?: boolean;
}

export interface SessionInfo {
  id: string;
  device: string;
  ip: string;
  created_at: string;
  last_seen: string;
  current: boolean;
}
