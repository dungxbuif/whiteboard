import { createMcpHTTP } from './server.js';
import { apiBase } from './auth.js';
import { AuditJournal } from './audit.js';

const apiURL = process.env.WHITEBOARD_API_URL;
if (!apiURL) throw new Error('WHITEBOARD_API_URL required');
apiBase(apiURL);
const auditPath = process.env.MCP_AUDIT_PATH;
if (!auditPath) throw new Error('MCP_AUDIT_PATH must name a persistent journal');
const audit = new AuditJournal(auditPath);
const allowedOrigins = (process.env.MCP_ALLOWED_ORIGINS || '').split(',').map(v => v.trim()).filter(Boolean);
for (const origin of allowedOrigins) {
  if (new URL(origin).origin !== origin) throw new Error('MCP_ALLOWED_ORIGINS requires exact origins');
}
const port = Number(process.env.PORT || 3002);
if (!Number.isInteger(port) || port < 1 || port > 65535) throw new Error('Invalid PORT');
const server = createMcpHTTP({ apiURL, allowedOrigins, audit });
server.listen(port, process.env.MCP_BIND_HOST || '127.0.0.1');
for (const signal of ['SIGINT', 'SIGTERM'] as const) {
  process.on(signal, () => { server.closeAllConnections(); server.close(); });
}
