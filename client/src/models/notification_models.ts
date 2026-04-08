export type NotificationType = "info" | "success" | "warning" | "error";

export interface Notification {
  id: number;
  user_id: number;
  title: string;
  message: string;
  type: NotificationType;
  read_at: string | null;
  created_at: string;
  updated_at: string;
}
