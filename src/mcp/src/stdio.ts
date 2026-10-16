import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import { homedir } from 'node:os';
import { join } from 'node:path';
import { GoClient } from './go-client.js';
import { createMcpServer } from './server.js';
import { AuditJournal } from './audit.js';

async function main(): Promise<void> {
  const url = process.env.WHITEBOARD_API_URL;
  const key = process.env.WHITEBOARD_AGENT_KEY;
  if (!url || !key) throw new Error('Missing remote configuration');
  const auditPath = process.env.MCP_AUDIT_PATH || join(homedir(), '.local', 'state', 'whiteboard-mcp', 'audit.jsonl');
  const audit = new AuditJournal(auditPath);
  const api = new GoClient(url, key);
  await api.authenticate();
  const server = createMcpServer(api, audit);
  await server.connect(new StdioServerTransport(process.stdin, process.stdout, { maxBufferSize: 1024 * 1024 }));
  process.stdin.on('end', () => { void server.close(); });
}
main().catch(() => {
  process.stderr.write('Whiteboard MCP could not connect. Check remote URL, agent credentials and audit journal configuration.\n');
  process.exitCode = 1;
});
