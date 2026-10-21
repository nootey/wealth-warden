import http from "k6/http";
import exec from "k6/execution";
import { check, fail, sleep } from "k6";

const BASE_URL = __ENV.BASE_URL || "http://localhost:2000";
const SEED_USERS = Number(__ENV.SEED_USERS || 1000);
const EMAIL_DOMAIN = __ENV.EMAIL_DOMAIN || "seed.local";
const PASSWORD = __ENV.PASSWORD || "password";

const N = {
  login: "auth/login",
  current: "auth/current",
  categories: "transactions/categories",
  accountsAll: "accounts/all",
  txnCount: "transactions/count",
  accountTypes: "accounts/types",
  years: "analytics/years",
  subtype: "accounts/subtype",
  sync: "accounts/sync",
  networth: "analytics/networth",
  accountsPage: "accounts/paginated",
  cashFlow: "analytics/cash-flow",
  categoryBreakdown: "analytics/category-breakdown",
  sankey: "analytics/sankey",
};

// A threshold on each sub-metric is what makes k6 print per-endpoint timings in the summary
export function endpointThresholds(rule) {
  return Object.fromEntries(Object.values(N).map((n) => [`http_req_duration{name:${n}}`, [rule]]));
}

const api = (path) => `${BASE_URL}/api/${path}`;

let loggedIn = false;

function login() {
  const n = ((exec.vu.idInTest - 1) % SEED_USERS) + 1;
  const email = `bulk${String(n).padStart(5, "0")}@${EMAIL_DOMAIN}`;

  const res = http.post(
    api("auth/login"),
    JSON.stringify({ email, password: PASSWORD, remember_me: false }),
    { headers: { "Content-Type": "application/json" }, tags: { name: N.login } },
  );

  if (!check(res, { [`${N.login} 200`]: (r) => r.status === 200 })) {
    // Without a pause, a failing login retries in a tight loop and floods the server
    sleep(1);
    fail(`login failed for ${email}: ${res.status} ${res.body}`);
  }
  loggedIn = true;

  // The client loads the current user once at app start, not on every page change
  getAll([["auth/current", N.current]]);
}

// Parallel GETs, like the browser fires them; `name` groups URLs with different query strings
function getAll(reqs) {
  const responses = http.batch(
    reqs.map(([path, name]) => ({ method: "GET", url: api(path), params: { tags: { name } } })),
  );
  responses.forEach((r, i) => check(r, { [`${reqs[i][1]} 200`]: (x) => x.status === 200 }));
  return responses;
}

// Requires `noCookiesReset: true` in the script options, or every iteration logs in again
export function loadDashboard() {
  if (!loggedIn) login();

  const [, , , , years, checking] = getAll([
    ["transactions/categories?deleted=false", N.categories],
    ["accounts/all", N.accountsAll],
    ["transactions?page=1&rowsPerPage=1", N.txnCount],
    ["accounts/types", N.accountTypes],
    ["analytics/years", N.years],
    ["accounts/subtype/checking", N.subtype],
  ]);

  const allYears = (years.json() || []).map((y) => y.year).sort((a, b) => b - a);
  const year = allYears[0] || new Date().getFullYear();
  const breakdownYears = allYears.slice(0, 3).join(",") || year;
  const defaultChecking = (checking.json() || []).find((a) => a.is_default);
  const account = defaultChecking ? `&account=${defaultChecking.id}` : "";

  const sync = http.post(api("accounts/sync/balances"), null, { tags: { name: N.sync } });
  check(sync, { [`${N.sync} 200`]: (r) => r.status === 200 });

  const page = "page=1&rowsPerPage=25&sort%5Bfield%5D=id&sort%5Border%5D=asc";
  getAll([
    ["analytics/networth?range=ytd", N.networth],
    [`accounts?${page}&classification=asset`, N.accountsPage],
    [`accounts?${page}&classification=liability`, N.accountsPage],
    [`analytics/yearly-cash-flow-breakdown?year=${year}${account}`, N.cashFlow],
    [
      `analytics/monthly-category-breakdown?years=${breakdownYears}&class=expense&percent=false${account}`,
      N.categoryBreakdown,
    ],
    [`analytics/sankey?year=${year}${account}`, N.sankey],
  ]);
}
