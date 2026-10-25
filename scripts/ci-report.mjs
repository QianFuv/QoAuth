/** Report stage identity and step outcomes without raw logs or environment dumps. */
import fs from "node:fs";
import path from "node:path";

const STEPS = JSON.parse(process.env.CI_CHECKS || "{}");
const REPORT = {
  stage: process.env.CI_STAGE,
  source: process.env.GITHUB_SHA,
  platform: process.env.CI_PLATFORM || "independent",
  status: process.env.CI_STATUS,
  checks: Object.fromEntries(Object.entries(STEPS).map(([name, step]) => [name, step.conclusion])),
};
if (process.env.CI_REPORT_FILE) {
  fs.mkdirSync(path.dirname(process.env.CI_REPORT_FILE), { recursive: true });
  fs.writeFileSync(process.env.CI_REPORT_FILE, JSON.stringify(REPORT, null, 2) + "\n");
}
if (process.env.GITHUB_STEP_SUMMARY) fs.appendFileSync(process.env.GITHUB_STEP_SUMMARY, `### ${REPORT.stage}\n\nSource: \`${REPORT.source}\` · ${REPORT.platform} · ${REPORT.status}\n\n| Check | Result |\n|---|---|\n${Object.entries(REPORT.checks).map(([name, result]) => `| ${name} | ${result} |`).join("\n")}\n`);
console.log(JSON.stringify(REPORT));
