import type { FilterObj } from "../models/shared_models.ts";

type Key = string;

const makeGroupKey = (f: FilterObj): Key =>
  `${f.source}::${f.field}::${f.operator ?? ""}`;
const makeValueKey = (f: FilterObj): Key =>
  `${makeGroupKey(f)}::${JSON.stringify(f.value)}`;

const filterHelper = {
  initSort(field: string = "") {
    if (field === "") field = "created_at";

    return {
      order: -1,
      field: field,
    };
  },
  toggleSort(sortValue: number): number {
    switch (sortValue) {
      case 1:
        return -1;
      case -1:
        return 1;
      default:
        return 1;
    }
  },
  mergeFilters(existing: FilterObj[], incoming: FilterObj[]): FilterObj[] {
    // Ops that replace the whole group for (source, field, operator)
    const replaceOps = new Set([">=", "<=", "in", "like", "=", "equals"]);
    // Ops that can have multiple entries in the same group (keep all values)
    const multiValueOps = new Set(["=", "equals"]);

    const groupsToReplace = new Set<Key>();
    for (const f of incoming) {
      const op = f.operator ?? "";
      if (replaceOps.has(op)) groupsToReplace.add(makeGroupKey(f));
    }

    const map = new Map<Key, FilterObj>();
    for (const f of existing) {
      const op = f.operator ?? "";
      const gk = makeGroupKey(f);
      if (replaceOps.has(op) && groupsToReplace.has(gk)) {
        continue; // incoming will replace this whole group
      }
      map.set(replaceOps.has(op) ? gk : makeValueKey(f), f);
    }

    for (const f of incoming) {
      const op = f.operator ?? "";
      if (replaceOps.has(op)) {
        const key = multiValueOps.has(op) ? makeValueKey(f) : makeGroupKey(f);
        map.set(key, f);
      } else {
        map.set(makeValueKey(f), f);
      }
    }

    return Array.from(map.values());
  },
};

export default filterHelper;
