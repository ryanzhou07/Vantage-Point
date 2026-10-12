import { spawnSync } from 'node:child_process';

// Preserve the complete scan. No pnpm-wide advisory ignore flag is used.
const scan = spawnSync('pnpm', ['audit', '--json'], { encoding: 'utf8', timeout: 120_000 });
if (scan.error || ![0, 1].includes(scan.status)) {
  console.error('Dependency audit unavailable or failed to execute');
  process.exit(1);
}
let report;
try { report = JSON.parse(scan.stdout); } catch {
  console.error('Dependency audit returned invalid JSON');
  process.exit(1);
}
if (!report.advisories || !report.metadata?.vulnerabilities || report.error) {
  console.error('Unexpected dependency audit response');
  process.exit(1);
}
const path = 'apps__web>eslint-config-next>@next/eslint-plugin-next>fast-glob>micromatch>braces';
let failed = false;
for (const advisory of Object.values(report.advisories)) {
  const id = advisory.github_advisory_id;
  const approved = id === 'GHSA-vfj7-8cjw-p6xm' && advisory.module_name === 'braces'
    && advisory.severity === 'high' && advisory.patched_versions === '<0.0.0'
    && Date.now() < Date.parse('2026-11-11T00:00:00Z')
    && advisory.findings?.length > 0
    && advisory.findings.every(f => f.version === '3.0.3' && f.paths?.length > 0 && f.paths.every(p => p === path));
  console.log(`${id ?? 'unknown'}: ${advisory.module_name} (${advisory.severity})${approved ? ' — temporary owner-approved lint-only exception; revisit by 2026-11-11' : ''}`);
  if (['high', 'critical'].includes(advisory.severity) && !approved) failed = true;
}
console.log('Audit totals (including exception):', JSON.stringify(report.metadata.vulnerabilities));
if (failed) process.exit(1);
