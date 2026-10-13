import { endpointThresholds, loadDashboard } from "./lib/dashboard.js";

// Arrival rate fixes dashboard loads per second; a slow server then needs more VUs
// instead of quietly sending fewer requests, so saturation shows up as latency and dropped_iterations
export const options = {
  noCookiesReset: true,
  scenarios: {
    stress: {
      executor: "ramping-arrival-rate",
      startRate: 10,
      timeUnit: "1s",
      preAllocatedVUs: 50,
      // Capped at the seeded user count so no two VUs share a login
      maxVUs: Number(__ENV.SEED_USERS || 1000),
      stages: [
        { duration: "30s", target: 25 },
        { duration: "1m", target: 25 },
        { duration: "30s", target: 50 },
        { duration: "1m", target: 50 },
        { duration: "30s", target: 100 },
        { duration: "1m", target: 100 },
        { duration: "30s", target: 200 },
        { duration: "1m", target: 200 },
        { duration: "30s", target: 400 },
        { duration: "1m", target: 400 },
      ],
    },
  },
  thresholds: {
    http_req_failed: [{ threshold: "rate<0.05", abortOnFail: true, delayAbortEval: "30s" }],
    http_req_duration: [{ threshold: "p(95)<1000", abortOnFail: true, delayAbortEval: "30s" }],
    ...endpointThresholds("p(95)<1000"),
  },
};

export default function () {
  loadDashboard();
}
