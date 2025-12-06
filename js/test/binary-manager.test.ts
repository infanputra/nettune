import { describe, expect, test } from 'bun:test';
import { getDefaultCacheDir, createDefaultConfig } from '../src/binary-manager';
import os from 'node:os';
import path from 'node:path';

describe('getDefaultCacheDir', () => {
  test('returns a path under home directory', () => {
    const cacheDir = getDefaultCacheDir();
    const homeDir = os.homedir();

    expect(cacheDir).toContain('nettune');
    // Should be under home or XDG cache
    expect(
      cacheDir.startsWith(homeDir) ||
      cacheDir.startsWith(process.env.XDG_CACHE_HOME || '')
    ).toBe(true);
  });

  test('returns path ending with nettune', () => {
    const cacheDir = getDefaultCacheDir();
    expect(path.basename(cacheDir)).toBe('nettune');
  });
});

describe('createDefaultConfig', () => {
  test('creates config with default values', () => {
    const config = createDefaultConfig();

    expect(config.cacheDir).toBeDefined();
    expect(config.githubRepo).toBe('jtsang4/nettune');
    expect(config.version).toBe('latest');
  });

  test('allows overriding cacheDir', () => {
    const config = createDefaultConfig({ cacheDir: '/custom/path' });

    expect(config.cacheDir).toBe('/custom/path');
    expect(config.githubRepo).toBe('jtsang4/nettune');
    expect(config.version).toBe('latest');
  });

  test('allows overriding version', () => {
    const config = createDefaultConfig({ version: 'v1.0.0' });

    expect(config.version).toBe('v1.0.0');
    expect(config.githubRepo).toBe('jtsang4/nettune');
  });

  test('allows overriding githubRepo', () => {
    const config = createDefaultConfig({ githubRepo: 'other/repo' });

    expect(config.githubRepo).toBe('other/repo');
    expect(config.version).toBe('latest');
  });
});
