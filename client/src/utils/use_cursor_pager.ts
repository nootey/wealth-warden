import { ref } from "vue";
import type { CursorPage, PaginatorState } from "../models/shared_models.ts";

export function useCursorPager(rowsPerPage: number) {
  const paginator = ref<PaginatorState>({
    total: 0,
    from: 0,
    to: 0,
    rowsPerPage,
  });
  const page = ref(0);
  const cursor = ref<string | null>(null);
  let neighbours: Record<number, string | null> = { 1: null, [-1]: null };

  function reset() {
    page.value = 0;
    cursor.value = null;
  }

  function goTo(target: number, rows = paginator.value.rowsPerPage): boolean {
    if (rows !== paginator.value.rowsPerPage) {
      paginator.value.rowsPerPage = rows;
      reset();
      return true;
    }
    if (target === page.value) return false;

    // Rows deleted under an open page can leave no way back but the start.
    if (target <= 0 || (target < page.value && !neighbours[-1])) {
      reset();
      return true;
    }

    const step = Math.sign(target - page.value);
    const next = neighbours[step];
    if (!next) return false;

    cursor.value = next;
    page.value += step;
    return true;
  }

  function update(res: CursorPage, count: number) {
    neighbours = {
      1: res.next_cursor ?? null,
      [-1]: res.prev_cursor ?? null,
    };

    const from = count > 0 ? page.value * paginator.value.rowsPerPage + 1 : 0;
    paginator.value.total = res.total_records ?? 0;
    paginator.value.from = from;
    paginator.value.to = count > 0 ? from + count - 1 : 0;
  }

  return { paginator, page, cursor, reset, goTo, update };
}
