// Loaded only by the E1 child process, never by a production entrypoint.
import { createRequire } from 'node:module';
import { existsSync, writeFileSync, unlinkSync } from 'node:fs';
import { join } from 'node:path';
const require = createRequire(new URL('../../src/collab/package.json', import.meta.url));
const WebSocket = require('ws');
const directory = process.env.E1_FAULT_DIR;
if (!directory || !/\/collab-e1-[^/]+$/.test(directory)) throw new Error('Unsafe E1 fault directory');
const send = WebSocket.prototype.send;
WebSocket.prototype.send = function (data, ...args) {
  if (typeof data === 'string' && existsSync(join(directory, 'kill-before-ack'))) {
    const frame = JSON.parse(data);
    if (frame.type === 'ack') {
      unlinkSync(join(directory, 'kill-before-ack'));
      writeFileSync(join(directory, 'committed-before-ack.json'), JSON.stringify(frame), {mode:0o600});
      process.kill(process.pid, 'SIGKILL');
    }
  }
  return send.call(this, data, ...args);
};

// Optional diagnostic instrumentation records only durations and SQL classes.
// Credentials, SQL bodies and scene bytes never enter this file.
if (process.env.E1_PROFILE === 'true') {
  const { syncBuiltinESMExports } = await import('node:module');
  const workerThreads = require('node:worker_threads');
  const OriginalWorker = workerThreads.Worker;
  const metrics = { queries: {}, errors: {}, workers: 0, onlineMs: [], validationMs: [], cpu: {}, memory: {} };
  workerThreads.Worker = class extends OriginalWorker {
    constructor(...args) {
      const started = performance.now(); super(...args); metrics.workers++;
      this.once('online', () => metrics.onlineMs.push(performance.now() - started));
      this.jobStarted = started;
      this.on('message', () => metrics.validationMs.push(performance.now() - this.jobStarted));
    }
    postMessage(...args) { this.jobStarted = performance.now(); return super.postMessage(...args); }
  };
  syncBuiltinESMExports();
  const Client = require('pg').Client, query = Client.prototype.query;
  Client.prototype.query = function (...args) {
    const sql = typeof args[0] === 'string' ? args[0] : args[0]?.text ?? '';
    const kind = /SELECT acl_version.*FOR UPDATE/.test(sql) ? 'workspace_lock' : /SELECT id FROM scenes.*FOR UPDATE/.test(sql) ? 'scene_lock' : /SELECT \* FROM collab_documents/.test(sql) ? 'document_load' : /SELECT update_bytes/.test(sql) ? 'update_load' : /pg_locks/.test(sql) ? 'ownership_check' : /INSERT INTO collab_receipts/.test(sql) ? 'receipt_insert' : 'other';
    const started = performance.now(); const result = query.apply(this, args);
    if (!result?.then) return result;
    return result.then(value => { const bucket = metrics.queries[kind] ??= { count: 0, totalMs: 0, maxMs: 0 }; const elapsed = performance.now() - started; bucket.count++; bucket.totalMs += elapsed; bucket.maxMs = Math.max(bucket.maxMs, elapsed); return value; }, error => { metrics.errors[error.code ?? 'unknown'] = (metrics.errors[error.code ?? 'unknown'] ?? 0) + 1; throw error; });
  };
  const flush = () => { metrics.cpu = process.cpuUsage(); metrics.memory = process.memoryUsage(); writeFileSync(join(directory, 'runtime-profile.json'), JSON.stringify(metrics)); };
  setInterval(flush, 1000).unref(); process.on('SIGTERM', flush);
}
