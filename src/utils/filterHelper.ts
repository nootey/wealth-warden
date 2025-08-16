import type {FilterObj} from "../models/shared_models.ts";

type Key = string;

function makeKey(f: FilterObj): Key {
    return `${f.source}::${f.field}::${f.operator ?? ''}`;
}

const filterHelper = {
    mergeFilters(existing: FilterObj[], incoming: FilterObj[]): FilterObj[] {
    const replaceOps = new Set(['>=', '<=', 'in', 'like']);
    const map = new Map<Key, FilterObj>();

    for (const f of existing) {
        const key = replaceOps.has(f.operator ?? '') ? makeKey(f) : Symbol() as unknown as Key;
        map.set(key, f);
    }

    for (const f of incoming) {
        const op = f.operator ?? '';
        if (replaceOps.has(op)) {
            map.set(makeKey(f), f);
        } else {
            map.set(Symbol() as unknown as Key, f);
        }
    }

    return Array.from(map.values());
}
}

export default filterHelper;