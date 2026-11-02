import { describe, it, expect } from "vitest";
import { useCursorPager } from "./use_cursor_pager.ts";

const page = (next: string | null, prev: string | null, total = 25) => ({
  rows_per_page: 10,
  total_records: total,
  next_cursor: next,
  prev_cursor: prev,
});

describe("useCursorPager", () => {
  it("starts on the first page with no cursor", () => {
    const pager = useCursorPager(10);

    expect(pager.cursor.value).toBeNull();
    expect(pager.page.value).toBe(0);
  });

  it("reports the rows on screen and the total", () => {
    const pager = useCursorPager(10);
    pager.update(page("n1", null), 10);

    expect(pager.paginator.value).toEqual({
      total: 25,
      from: 1,
      to: 10,
      rowsPerPage: 10,
    });
  });

  it("moves forward with the next cursor and jumps back to the start", () => {
    const pager = useCursorPager(10);
    pager.update(page("n1", null), 10);

    expect(pager.goTo(1)).toBe(true);
    expect(pager.cursor.value).toBe("n1");
    pager.update(page("n2", "p2"), 10);
    expect(pager.paginator.value.from).toBe(11);
    expect(pager.paginator.value.to).toBe(20);

    expect(pager.goTo(0)).toBe(true);
    expect(pager.cursor.value).toBeNull();
  });

  it("walks back one page with the previous cursor", () => {
    const pager = useCursorPager(10);
    pager.update(page("n1", null), 10);
    pager.goTo(1);
    pager.update(page("n2", "p2"), 10);
    pager.goTo(2);
    pager.update(page(null, "p3"), 5);

    expect(pager.goTo(1)).toBe(true);
    expect(pager.cursor.value).toBe("p3");
    expect(pager.page.value).toBe(1);
  });

  it("does nothing past the last page", () => {
    const pager = useCursorPager(10);
    pager.update(page(null, null, 3), 3);

    expect(pager.goTo(1)).toBe(false);
    expect(pager.page.value).toBe(0);
  });

  it("returns to the first page when going back has no cursor", () => {
    const pager = useCursorPager(10);
    pager.update(page("n1", null), 10);
    pager.goTo(1);
    pager.update(page("n2", "p2"), 10);
    pager.goTo(2);
    pager.update(page(null, null), 0);

    expect(pager.goTo(1)).toBe(true);
    expect(pager.cursor.value).toBeNull();
    expect(pager.page.value).toBe(0);
  });

  it("starts over when the page size changes", () => {
    const pager = useCursorPager(10);
    pager.update(page("n1", null), 10);
    pager.goTo(1);

    expect(pager.goTo(1, 25)).toBe(true);
    expect(pager.cursor.value).toBeNull();
    expect(pager.paginator.value.rowsPerPage).toBe(25);
  });

  it("shows an empty range for an empty page", () => {
    const pager = useCursorPager(10);
    pager.update(page(null, null, 0), 0);

    expect(pager.paginator.value.from).toBe(0);
    expect(pager.paginator.value.to).toBe(0);
  });
});
