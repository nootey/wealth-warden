export interface Causer {
  id: number;
  name: string;
}

export interface ActivityLog {
  event: string | null;
  category: string | null;
  causer: object | null;
  metadata: object | null;
}
