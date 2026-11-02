export interface FilterObj {
  source: string;
  field: string | null;
  operator: string;
  value: unknown;
  display?: string;
}

export interface SortObj {
  order: number;
  field: string;
}

export type PaginatorState = {
  total: number;
  from: number;
  to: number;
  rowsPerPage: number;
};

export interface CursorPage {
  rows_per_page: number;
  total_records: number;
  next_cursor: string | null;
  prev_cursor: string | null;
}

export const NAV_WIDTH = { collapsed: "76px", expanded: "248px" };
