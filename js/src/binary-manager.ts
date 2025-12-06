import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import crypto from 'node:crypto';
import type {
  BinaryManagerConfig,
  PlatformInfo,
  GithubRelease,
  ChecksumInfo,
} from './types.js';
import { getBinaryName, getChecksumsFileName } from './platform.js';

const DEFAULT_GITHUB_REPO = 'jtsang4/nettune';
const DEFAULT_VERSION = 'latest';

/**
 * Get the default cache directory for storing binaries
 */
export function getDefaultCacheDir(): string {
  const homeDir = os.homedir();
  // Follow XDG Base Directory Specification on Linux
  if (process.platform === 'linux') {
    const xdgCache = process.env.XDG_CACHE_HOME;
    if (xdgCache) {
      return path.join(xdgCache, 'nettune');
    }
  }
  return path.join(homeDir, '.cache', 'nettune');
}

/**
 * Create default configuration for BinaryManager
 */
export function createDefaultConfig(
  overrides?: Partial<BinaryManagerConfig>
): BinaryManagerConfig {
  return {
    cacheDir: overrides?.cacheDir ?? getDefaultCacheDir(),
    githubRepo: overrides?.githubRepo ?? DEFAULT_GITHUB_REPO,
    version: overrides?.version ?? DEFAULT_VERSION,
  };
}

/**
 * BinaryManager handles downloading, caching, and verifying the nettune binary
 */
export class BinaryManager {
  private config: BinaryManagerConfig;

  constructor(config: BinaryManagerConfig) {
    this.config = config;
  }

  /**
   * Ensure the binary exists and is ready to use
   * Returns the path to the binary
   */
  async ensureBinary(platform: PlatformInfo): Promise<string> {
    const binaryName = getBinaryName(platform);
    const version = await this.resolveVersion();
    const binaryDir = path.join(this.config.cacheDir, version);
    const binaryPath = path.join(binaryDir, binaryName);

    // Check if binary already exists and is valid
    if (await this.isValidBinary(binaryPath, platform, version)) {
      this.log(`Using cached binary: ${binaryPath}`);
      return binaryPath;
    }

    // Download the binary
    this.log(`Downloading nettune ${version} for ${platform.os}-${platform.arch}...`);
    await this.downloadBinary(binaryPath, platform, version);

    // Make executable
    await this.makeExecutable(binaryPath);

    this.log(`Binary ready: ${binaryPath}`);
    return binaryPath;
  }

  /**
   * Resolve the actual version tag from "latest" or return as-is
   */
  private async resolveVersion(): Promise<string> {
    if (this.config.version !== 'latest') {
      return this.config.version;
    }

    const releaseUrl = `https://api.github.com/repos/${this.config.githubRepo}/releases/latest`;
    try {
      const response = await fetch(releaseUrl, {
        headers: {
          Accept: 'application/vnd.github.v3+json',
          'User-Agent': 'nettune-mcp',
        },
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch latest release: ${response.statusText}`);
      }

      const release = (await response.json()) as GithubRelease;
      return release.tag_name;
    } catch (error) {
      throw new Error(
        `Failed to resolve latest version: ${error instanceof Error ? error.message : String(error)}`
      );
    }
  }

  /**
   * Check if a binary exists and has valid checksum
   */
  private async isValidBinary(
    binaryPath: string,
    platform: PlatformInfo,
    version: string
  ): Promise<boolean> {
    try {
      await fs.promises.access(binaryPath, fs.constants.X_OK);
    } catch {
      return false;
    }

    // Try to verify checksum if available
    try {
      const checksumInfo = await this.fetchChecksum(platform, version);
      if (checksumInfo) {
        const actualHash = await this.computeFileHash(binaryPath);
        return actualHash === checksumInfo.sha256;
      }
    } catch {
      // If checksum verification fails, still use cached binary
      this.log('Warning: Could not verify checksum, using cached binary');
    }

    return true;
  }

  /**
   * Download the binary from GitHub releases
   */
  private async downloadBinary(
    destPath: string,
    platform: PlatformInfo,
    version: string
  ): Promise<void> {
    const binaryName = getBinaryName(platform);
    const downloadUrl = `https://github.com/${this.config.githubRepo}/releases/download/${version}/${binaryName}`;

    // Ensure directory exists
    const dir = path.dirname(destPath);
    await fs.promises.mkdir(dir, { recursive: true });

    // Download to temp file first
    const tempPath = `${destPath}.tmp`;

    try {
      const response = await fetch(downloadUrl, {
        headers: {
          'User-Agent': 'nettune-mcp',
        },
        redirect: 'follow',
      });

      if (!response.ok) {
        if (response.status === 404) {
          throw new Error(
            `Binary not found for ${platform.os}-${platform.arch} version ${version}. ` +
            `Please check if the release exists at: https://github.com/${this.config.githubRepo}/releases`
          );
        }
        throw new Error(`Failed to download binary: ${response.statusText}`);
      }

      if (!response.body) {
        throw new Error('No response body received');
      }

      // Stream the response to file
      const arrayBuffer = await response.arrayBuffer();
      await fs.promises.writeFile(tempPath, Buffer.from(arrayBuffer));

      // Verify checksum if available
      const checksumInfo = await this.fetchChecksum(platform, version);
      if (checksumInfo) {
        const actualHash = await this.computeFileHash(tempPath);
        if (actualHash !== checksumInfo.sha256) {
          await fs.promises.unlink(tempPath);
          throw new Error(
            `Checksum mismatch: expected ${checksumInfo.sha256}, got ${actualHash}`
          );
        }
        this.log('Checksum verified successfully');
      }

      // Move temp file to final location
      await fs.promises.rename(tempPath, destPath);
    } catch (error) {
      // Clean up temp file on error
      try {
        await fs.promises.unlink(tempPath);
      } catch {
        // Ignore cleanup errors
      }
      throw error;
    }
  }

  /**
   * Fetch checksum information from the release
   */
  private async fetchChecksum(
    platform: PlatformInfo,
    version: string
  ): Promise<ChecksumInfo | null> {
    const checksumFileName = getChecksumsFileName();
    const checksumUrl = `https://github.com/${this.config.githubRepo}/releases/download/${version}/${checksumFileName}`;

    try {
      const response = await fetch(checksumUrl, {
        headers: {
          'User-Agent': 'nettune-mcp',
        },
      });

      if (!response.ok) {
        return null;
      }

      const content = await response.text();
      const binaryName = getBinaryName(platform);

      // Parse checksums file (format: "sha256  filename")
      for (const line of content.split('\n')) {
        const parts = line.trim().split(/\s+/);
        if (parts.length >= 2 && parts[1] === binaryName) {
          return {
            sha256: parts[0],
            filename: parts[1],
          };
        }
      }

      return null;
    } catch {
      return null;
    }
  }

  /**
   * Compute SHA256 hash of a file
   */
  private async computeFileHash(filePath: string): Promise<string> {
    const hash = crypto.createHash('sha256');
    const stream = fs.createReadStream(filePath);

    return new Promise((resolve, reject) => {
      stream.on('data', (data: string | Buffer) => hash.update(data));
      stream.on('end', () => resolve(hash.digest('hex')));
      stream.on('error', reject);
    });
  }

  /**
   * Make a file executable (chmod +x)
   */
  private async makeExecutable(filePath: string): Promise<void> {
    if (process.platform === 'win32') {
      // Windows doesn't need chmod
      return;
    }
    await fs.promises.chmod(filePath, 0o755);
  }

  /**
   * Log a message to stderr (to avoid polluting MCP stdout)
   */
  private log(message: string): void {
    console.error(`[nettune-mcp] ${message}`);
  }
}
