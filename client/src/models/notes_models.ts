export interface Note {
  id?: number;
  content: string;
  resolved_at: Date | null;
  created_at: Date;
}
