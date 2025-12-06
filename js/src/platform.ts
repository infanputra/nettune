import os from 'node:os';
import type { PlatformInfo, OS, Arch } from './types.js';

/**
 * Map Node.js platform strings to our supported OS types
 */
const PLATFORM_MAP: Record<string, OS> = {
  darwin: 'darwin',
  linux: 'linux',
  win32: 'win32',
};

/**
 * Map Node.js arch strings to our supported arch types
 */
const ARCH_MAP: Record<string, Arch> = {
  x64: 'x64',
  arm64: 'arm64',
  // Map amd64 style names if needed
  amd64: 'x64',
};

/**
 * Detect the current platform (OS and architecture)
 * @throws Error if platform is not supported
 */
export function detectPlatform(): PlatformInfo {
  const platform = os.platform();
  const arch = os.arch();

  const mappedOS = PLATFORM_MAP[platform];
  const mappedArch = ARCH_MAP[arch];

  if (!mappedOS) {
    throw new Error(
      `Unsupported operating system: ${platform}. Supported: darwin, linux, win32`
    );
  }

  if (!mappedArch) {
    throw new Error(
      `Unsupported architecture: ${arch}. Supported: x64, arm64`
    );
  }

  return {
    os: mappedOS,
    arch: mappedArch,
  };
}

/**
 * Get the binary name for a given platform
 * Format: nettune-{os}-{arch} (e.g., nettune-darwin-arm64, nettune-linux-x64)
 */
export function getBinaryName(platform: PlatformInfo): string {
  // Map our arch names to Go-style names used in releases
  const archName = platform.arch === 'x64' ? 'amd64' : platform.arch;
  const osName = platform.os === 'win32' ? 'windows' : platform.os;
  const ext = platform.os === 'win32' ? '.exe' : '';
  return `nettune-${osName}-${archName}${ext}`;
}

/**
 * Get the checksums file name for a given version
 */
export function getChecksumsFileName(): string {
  return 'checksums.txt';
}

/**
 * Check if the current platform is supported for nettune
 */
export function isPlatformSupported(): boolean {
  try {
    const platform = detectPlatform();
    // Windows support is limited, warn but allow
    if (platform.os === 'win32') {
      console.error('[nettune-mcp] Warning: Windows support is experimental');
    }
    return true;
  } catch {
    return false;
  }
}
