import { describe, it, expect } from "vitest";
import searchHelper from "./search_helper.ts";

describe("searchHelper", () => {
  describe("matchesQuery", () => {
    const category = ["Public Transportation", "public_transportation"];

    it("matches a substring anywhere in the field, not just the prefix", () => {
      expect(searchHelper.matchesQuery("tran", ...category)).toBe(true);
    });

    it("matches across an underscore boundary", () => {
      expect(searchHelper.matchesQuery("public trans", ...category)).toBe(true);
      expect(
        searchHelper.matchesQuery("public_transportation", ...category),
      ).toBe(true);
    });

    it("ignores token order", () => {
      expect(
        searchHelper.matchesQuery("transportation public", ...category),
      ).toBe(true);
    });

    it("requires every token to match", () => {
      expect(searchHelper.matchesQuery("public groceries", ...category)).toBe(
        false,
      );
    });

    it("folds diacritics", () => {
      expect(searchHelper.matchesQuery("cafe", "Café")).toBe(true);
      expect(searchHelper.matchesQuery("café", "Cafe")).toBe(true);
    });

    it("treats an empty query as a match", () => {
      expect(searchHelper.matchesQuery("   ", ...category)).toBe(true);
    });

    it("ignores null and undefined fields", () => {
      expect(
        searchHelper.matchesQuery("tran", null, undefined, "Transit"),
      ).toBe(true);
    });
  });

  describe("filterByQuery", () => {
    const categories = [
      { display_name: "Public Transportation", name: "public_transportation" },
      { display_name: "Groceries", name: "groceries" },
    ];

    it("returns a copy of every item for an empty query", () => {
      const result = searchHelper.filterByQuery(categories, "", (c) => [
        c.name,
      ]);
      expect(result).toHaveLength(2);
      expect(result).not.toBe(categories);
    });

    it("filters on any of the supplied fields", () => {
      const result = searchHelper.filterByQuery(categories, "tran", (c) => [
        c.display_name,
        c.name,
      ]);
      expect(result).toEqual([categories[0]]);
    });
  });
});
