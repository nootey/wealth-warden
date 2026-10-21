import http from "k6/http";
import { check, fail } from "k6";
import { Counter } from "k6/metrics";

const BASE_URL = __ENV.BASE_URL || "http://localhost:2000";
const EMAIL = __ENV.LEDGER_EMAIL || "bulk00001@seed.local";
const PASSWORD = __ENV.PASSWORD || "password";
const RACERS = Number(__ENV.RACERS || 50);
const PG_CONTAINER = __ENV.PG_CONTAINER || "postgres-postgres-1";
const PG_DB = __ENV.PG_DB || "wealth_warden";
const PG_USER = __ENV.PG_USER || "postgres";

// Each racer spends SPEND from an account that can only afford AFFORDABLE of them
const OPENING = 1000;
const SPEND = 100;
const AFFORDABLE = OPENING / SPEND;
const CHURN_OPENING = 100000;

const acceptedExpenses = new Counter("accepted_expenses");
const acceptedTransfers = new Counter("accepted_transfers");
const serverErrors = new Counter("server_errors");

export const options = {
  scenarios: {
    expense_race: { executor: "shared-iterations", vus: RACERS, iterations: RACERS, exec: "expenseRace" },
    transfer_race: { executor: "shared-iterations", vus: RACERS, iterations: RACERS, exec: "transferRace" },
    churn: { executor: "constant-vus", vus: 20, duration: "1m", startTime: "10s", exec: "churn" },
  },
  thresholds: {
    accepted_expenses: [`count<=${AFFORDABLE}`],
    accepted_transfers: [`count<=${AFFORDABLE}`],
    server_errors: ["count==0"],
  },
};

const api = (path) => `${BASE_URL}/api/${path}`;
const json = { headers: { "Content-Type": "application/json" } };
const body = (o) => JSON.stringify(o);
const rand = (min, max) => Math.round((min + Math.random() * (max - min)) * 100) / 100;
const pick = (arr) => arr[Math.floor(Math.random() * arr.length)];

export function setup() {
  const login = http.post(api("auth/login"), body({ email: EMAIL, password: PASSWORD }), json);
  if (login.status !== 200) fail(`login failed for ${EMAIL}: ${login.status} ${login.body}`);

  const checking = http
    .get(api("accounts/types"))
    .json()
    .find((t) => t.type === "cash" && t.sub_type === "checking" && t.classification === "asset");
  if (!checking) fail("no cash/checking account type");

  const run = Date.now();
  const openings = {
    expense: OPENING,
    source: OPENING,
    destination: 0,
    churnA: CHURN_OPENING,
    churnB: CHURN_OPENING,
  };

  for (const [key, balance] of Object.entries(openings)) {
    const res = http.put(
      api("accounts"),
      body({
        name: `k6 ledger ${run} ${key}`,
        account_type_id: checking.id,
        type: checking.type,
        sub_type: checking.sub_type,
        classification: checking.classification,
        balance,
      }),
      json,
    );
    if (res.status !== 200) fail(`creating ${key} account failed: ${res.status} ${res.body}`);
  }

  // Inserts don't return ids, so look the new accounts up by name
  const accounts = http.get(api("accounts/all")).json();
  const ids = {};
  for (const key of Object.keys(openings)) {
    const acc = accounts.find((a) => a.name === `k6 ledger ${run} ${key}`);
    if (!acc) fail(`created ${key} account not found`);
    ids[key] = acc.id;
  }

  return { session: login.cookies.session[0].value, ids, run };
}

function auth(data) {
  http.cookieJar().set(BASE_URL, "session", data.session);
}

function record(res, op, accepted) {
  if (res.status >= 500) serverErrors.add(1, { op });
  if (accepted && res.status === 200) accepted.add(1);
}

export function expenseRace(data) {
  auth(data);
  const res = http.put(
    api("transactions"),
    body({
      account_id: data.ids.expense,
      direction: "expense",
      amount: SPEND,
      txn_date: new Date().toISOString(),
      description: "k6 expense race",
    }),
    { ...json, tags: { name: "expense race" } },
  );
  record(res, "expense race", acceptedExpenses);
  check(res, { "expense race: no server error": (r) => r.status < 500 });
}

export function transferRace(data) {
  auth(data);
  const res = http.put(
    api("transactions/transfers"),
    body({ source_id: data.ids.source, destination_id: data.ids.destination, amount: SPEND }),
    { ...json, tags: { name: "transfer race" } },
  );
  record(res, "transfer race", acceptedTransfers);
  check(res, { "transfer race: no server error": (r) => r.status < 500 });
}

// Mixed writes on two well-funded accounts; transfers run in both directions to provoke lock-order deadlocks
export function churn(data) {
  auth(data);
  const accounts = [data.ids.churnA, data.ids.churnB];
  const account = pick(accounts);
  const roll = Math.random();
  let op;
  let res;

  if (roll < 0.4) {
    op = "insert";
    res = http.put(
      api("transactions"),
      body({
        account_id: account,
        direction: pick(["income", "expense"]),
        amount: rand(1, 50),
        txn_date: new Date().toISOString(),
        description: "k6 churn",
      }),
      { ...json, tags: { name: "churn insert" } },
    );
  } else if (roll < 0.85) {
    const list = http.get(api(`transactions?account_id=${account}&page=1&rowsPerPage=25`), {
      tags: { name: "churn list" },
    });
    const own = (list.json("data.records") || []).filter((t) => t.description === "k6 churn");
    if (!own.length) return;
    const txn = pick(own);

    if (roll < 0.65) {
      op = "update";
      res = http.put(
        api(`transactions/${txn.id}`),
        body({
          account_id: account,
          direction: txn.direction,
          amount: rand(1, 50),
          txn_date: txn.txn_date,
          description: "k6 churn",
        }),
        { ...json, tags: { name: "churn update" } },
      );
    } else {
      op = "delete";
      res = http.del(api(`transactions/${txn.id}`), null, { tags: { name: "churn delete" } });
    }
  } else {
    op = "transfer";
    const [source, destination] = Math.random() < 0.5 ? accounts : [...accounts].reverse();
    res = http.put(
      api("transactions/transfers"),
      body({ source_id: source, destination_id: destination, amount: rand(1, 50) }),
      { ...json, tags: { name: "churn transfer" } },
    );
  }

  record(res, `churn ${op}`);
  // 404: another VU deleted the same transaction first
  check(res, { [`churn ${op}: ok`]: (r) => r.status === 200 || r.status === 404 });
}

export function teardown(data) {
  console.log(
    `\nLedger run ${data.run}, account ids ${JSON.stringify(data.ids)}\n` +
      `Expected: at most ${AFFORDABLE} accepted expenses and transfers, no server errors.\n` +
      `Check stored balances against the ledger with:\n` +
      `  docker exec -i ${PG_CONTAINER} psql -U ${PG_USER} -d ${PG_DB} -v run="k6 ledger ${data.run}%" < scripts/k6/ledger_check.sql\n`,
  );
}
