const normalize = (value: unknown): string => {
  if (value === null || value === undefined) return "";
  return String(value)
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "")
    .toLowerCase()
    .replace(/[_\-/.]+/g, " ")
    .replace(/\s+/g, " ")
    .trim();
};

const searchHelper = {
  normalize,
  matchesQuery(query: string, ...fields: unknown[]): boolean {
    const tokens = normalize(query).split(" ").filter(Boolean);
    if (!tokens.length) return true;

    const haystack = fields.map(normalize).filter(Boolean).join(" ");
    return tokens.every((token) => haystack.includes(token));
  },
  filterByQuery<T>(
    items: T[],
    query: string,
    getFields: (item: T) => unknown[],
  ): T[] {
    if (!normalize(query)) return [...items];
    return items.filter((item) =>
      searchHelper.matchesQuery(query, ...getFields(item)),
    );
  },
};

export default searchHelper;
