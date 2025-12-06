import { spawn } from 'node:child_process';
import type { SpawnOptions } from './types.js';

/**
 * Spawn the nettune client process with stdio inherited
 * This allows the MCP protocol to pass through transparently
 */
export function spawnNettuneClient(options: SpawnOptions): Promise<number> {
  const { binaryPath, args, env } = options;

  return new Promise((resolve, reject) => {
    // Build the full argument list: "client" subcommand + user args
    const fullArgs = ['client', ...args];

    // Merge environment variables
    const processEnv = {
      ...process.env,
      ...env,
    };

    // Spawn the process with inherited stdio
    // This is critical for MCP - stdin/stdout must pass through cleanly
    const child = spawn(binaryPath, fullArgs, {
      stdio: 'inherit',
      env: processEnv,
    });

    // Handle spawn errors
    child.on('error', (error) => {
      reject(new Error(`Failed to spawn nettune client: ${error.message}`));
    });

    // Handle process exit
    child.on('close', (code) => {
      resolve(code ?? 0);
    });

    // Forward signals to the child process
    const signals: NodeJS.Signals[] = ['SIGINT', 'SIGTERM', 'SIGHUP'];
    for (const signal of signals) {
      process.on(signal, () => {
        child.kill(signal);
      });
    }
  });
}
