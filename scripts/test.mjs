// Full test suite in an isolated Compose project; run from any directory.
import { spawnSync } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import { mkdirSync, openSync, closeSync, writeFileSync, writeSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const project = `marketplace-test-${randomUUID().slice(0, 8)}`;
const reports = join(root, 'reports', project);
mkdirSync(reports, { recursive: true });
const transcript = openSync(join(reports, 'tests.log'), 'w');
const env = { ...process.env, ERP_BASE_URL: '', PUSH_BASE_URL: '', QUOTE_TTL: '15m' };
// Ignore a caller's active Compose project so cleanup can only affect this run.
for (const key of ['COMPOSE_FILE', 'COMPOSE_PROJECT_NAME', 'COMPOSE_PROFILES', 'COMPOSE_ENV_FILES']) delete env[key];
let configFile;

function run(command, args, options = {}) {
  const label = `\n> ${command} ${args.join(' ')}\n`;
  console.log(label.trimEnd());
  writeSync(transcript, label);
  const result = spawnSync(command, args, {
    cwd: root, env, encoding: 'utf8', timeout: 30 * 60 * 1000,
    // Write long build output directly to the report; capture only small structured responses.
    stdio: options.capture ? 'pipe' : ['ignore', transcript, transcript],
    ...options,
  });
  if (options.capture) writeSync(transcript, (result.stdout || '') + (result.stderr || ''));
  if (result.error || result.status !== 0) {
    throw new Error(`${command} failed (${result.status ?? result.error?.message}). See ${reports}`, { cause: result.error });
  }
  return result.stdout?.trim();
}

function compose(...args) {
  return run('docker', ['compose', '-f', configFile, '-p', project, ...args]);
}

function port(service, target) {
  const address = run('docker', ['compose', '-f', configFile, '-p', project, 'port', service, String(target)], { capture: true });
  return `http://${address}`;
}

async function ready(url) {
  for (let attempt = 0; attempt < 90; attempt++) {
    try {
      if ((await fetch(url, { signal: AbortSignal.timeout(5000) })).ok) return;
    } catch { /* The application may still be starting. */ }
    await new Promise(resolve => setTimeout(resolve, 2000));
  }
  throw new Error(`Readiness timed out: ${url}`);
}

try {
  if (Number(process.versions.node.split('.')[0]) < 20) throw new Error('Node.js 20+ is required (CI uses 22).');
  const config = JSON.parse(run('docker', ['compose', '-f', join(root, 'docker-compose.yml'), '--profile', '*', 'config', '--format', 'json'], { capture: true }));
  config.name = project;
  for (const [name, service] of Object.entries(config.services)) {
    delete service.ports;
    if (service.build) service.image = `marketplace-verification-${name}`;
  }
  for (const [service, target] of [['frontend', 80], ['api', 8080], ['mailpit', 8025]]) {
    config.services[service].ports = [{ target, published: '0', host_ip: '127.0.0.1', protocol: 'tcp' }];
  }
  for (const section of ['volumes', 'networks']) {
    for (const [name, resource] of Object.entries(config[section] || {})) {
      if (resource.external) throw new Error(`Refusing shared external ${section}: ${name}`);
      resource.name = `${project}_${name}`;
    }
  }
  configFile = join(reports, 'compose.json');
  writeFileSync(configFile, JSON.stringify(config, null, 2));
  env.COMPOSE_FILE = configFile;
  env.COMPOSE_PROJECT_NAME = project;
  compose('--profile', 'test', '--profile', 'frontend-test', 'build');
  compose('up', '--no-build', '-d', '--wait', '--wait-timeout', '180', 'mongo', 'pubsub-emulator');
  compose('--profile', 'test', 'run', '--no-deps', '--rm', 'tests', 'sh', '-ec',
    'test -z "$(gofmt -l .)"; go vet -tags=integration ./...; sh scripts/test-download-go-modules.sh; go test -race -tags=integration -count=1 ./...');
  compose('--profile', 'frontend-test', 'run', '--no-deps', '--rm',
    '-v', `${reports}:/reports`, 'frontend-tests', 'sh', '-ec',
    'trap \'if [ -d test/failures ]; then cp -R test/failures /reports/flutter-failures; fi\' EXIT; dart format --output=none --set-exit-if-changed lib test; flutter analyze --no-pub; flutter test --no-pub --reporter expanded');
  compose('up', '--no-build', '-d');
  env.APP_URL = port('frontend', 80);
  env.API_URL = port('api', 8080);
  await ready(`${env.APP_URL}/api/ready`);
  // npm.cmd needs cmd.exe on Windows; this is a fixed command without input interpolation.
  if (process.platform === 'win32') run('cmd.exe', ['/d', '/s', '/c', 'npm ci'], { cwd: join(root, 'frontend') });
  else run('npm', ['ci'], { cwd: join(root, 'frontend') });
  const playwright = join(root, 'frontend', 'node_modules', '@playwright', 'test', 'cli.js');
  for (const file of ['frontend/playwright.config.js', 'frontend/e2e/shop.spec.js', 'backend/scripts/worker-smoke.mjs', 'backend/scripts/dlq-smoke.mjs']) {
    run(process.execPath, ['--check', join(root, file)]);
  }
  run(process.execPath, [playwright, 'install', '--with-deps', 'chrome']);
  run(process.execPath, [playwright, 'test', '--config', join(root, 'frontend', 'playwright.config.js'), '--output', join(reports, 'browser')]);
  run(process.execPath, [join(root, 'backend', 'scripts', 'worker-smoke.mjs')]);
  run(process.execPath, [join(root, 'backend', 'scripts', 'dlq-smoke.mjs')]);

} catch (error) {
  console.error(error.message);
  process.exitCode = 1;
} finally {
  if (configFile) {
    for (const args of [['ps', '--all'], ['logs', '--no-color'], ['down', '--volumes', '--remove-orphans']]) {
      try { compose(...args); } catch (error) { console.error(error.message); process.exitCode = 1; }
    }
  }
  closeSync(transcript);
  console.log(`Reports: ${reports}`);
}

if (!process.exitCode) console.log('PASS: Go, integration/race, Flutter/goldens, Chrome, worker recovery and DLQ/email.');
