import { describe, expect, test } from 'bun:test';
import { detectPlatform, getBinaryName, isPlatformSupported } from '../src/platform';
import type { PlatformInfo } from '../src/types';

describe('platform detection', () => {
  test('detectPlatform returns valid platform info', () => {
    const platform = detectPlatform();

    expect(platform).toBeDefined();
    expect(['darwin', 'linux', 'win32']).toContain(platform.os);
    expect(['x64', 'arm64']).toContain(platform.arch);
  });

  test('isPlatformSupported returns true for supported platforms', () => {
    expect(isPlatformSupported()).toBe(true);
  });
});

describe('getBinaryName', () => {
  test('generates correct name for darwin arm64', () => {
    const platform: PlatformInfo = { os: 'darwin', arch: 'arm64' };
    expect(getBinaryName(platform)).toBe('nettune-darwin-arm64');
  });

  test('generates correct name for darwin x64', () => {
    const platform: PlatformInfo = { os: 'darwin', arch: 'x64' };
    expect(getBinaryName(platform)).toBe('nettune-darwin-amd64');
  });

  test('generates correct name for linux arm64', () => {
    const platform: PlatformInfo = { os: 'linux', arch: 'arm64' };
    expect(getBinaryName(platform)).toBe('nettune-linux-arm64');
  });

  test('generates correct name for linux x64', () => {
    const platform: PlatformInfo = { os: 'linux', arch: 'x64' };
    expect(getBinaryName(platform)).toBe('nettune-linux-amd64');
  });

  test('generates correct name for windows with .exe extension', () => {
    const platform: PlatformInfo = { os: 'win32', arch: 'x64' };
    expect(getBinaryName(platform)).toBe('nettune-win32-amd64.exe');
  });
});
