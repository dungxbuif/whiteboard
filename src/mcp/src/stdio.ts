import {StdioServerTransport} from '@modelcontextprotocol/sdk/server/stdio.js';
import {GoClient} from './go-client.js';
import {createMcpServer} from './server.js';
async function main(){const url=process.env.WHITEBOARD_API_URL;const key=process.env.WHITEBOARD_AGENT_KEY;if(!url||!key)throw new Error('Missing remote configuration');const api=new GoClient(url,key);await api.authenticate();const server=createMcpServer(api);await server.connect(new StdioServerTransport(process.stdin,process.stdout,{maxBufferSize:1024*1024}));process.stdin.on('end',()=>{void server.close()})}
main().catch(()=>{process.stderr.write('Whiteboard MCP could not connect. Check remote URL and agent credentials.\n');process.exitCode=1});
