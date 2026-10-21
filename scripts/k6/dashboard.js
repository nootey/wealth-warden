import { sleep } from "k6";
import { endpointThresholds, loadDashboard } from "./lib/dashboard.js";

export const options = {
  noCookiesReset: true,
  stages: [
    { duration: "30s", target: 10 },
    { duration: "1m", target: 10 },
    { duration: "1m", target: 50 },
    { duration: "2m", target: 50 },
    { duration: "30s", target: 0 },
  ],
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<500"],
    checks: ["rate>0.99"],
    ...endpointThresholds("p(95)<500"),
  },
};

export default function () {
  loadDashboard();
  sleep(1 + Math.random() * 2);
}
