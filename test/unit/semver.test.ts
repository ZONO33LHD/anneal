import { describe, it, expect } from 'vitest';
import {
  classifyUpdate,
  cleanVersion,
  compare,
  isUpgrade,
  parse,
} from '../../src/util/semver.js';

describe('semver', () => {
  it('cleans range prefixes', () => {
    expect(cleanVersion('^1.6.2')).toBe('1.6.2');
    expect(cleanVersion('~v2.0.0')).toBe('2.0.0');
    expect(cleanVersion('1.2.3-beta.1')).toBe('1.2.3');
  });

  it('parses semver', () => {
    expect(parse('v1.9.1')).toEqual({ major: 1, minor: 9, patch: 1 });
    expect(parse('not-a-version')).toBeNull();
  });

  it('compares versions', () => {
    expect(compare('1.0.0', '1.0.1')).toBe(-1);
    expect(compare('2.0.0', '1.9.9')).toBe(1);
    expect(compare('1.2.3', '1.2.3')).toBe(0);
  });

  it('classifies update type', () => {
    expect(classifyUpdate('1.6.2', '1.6.3')).toBe('patch');
    expect(classifyUpdate('1.6.2', '1.7.0')).toBe('minor');
    expect(classifyUpdate('4.1.2', '5.3.0')).toBe('major');
  });

  it('detects upgrades', () => {
    expect(isUpgrade('1.6.2', '1.7.0')).toBe(true);
    expect(isUpgrade('1.7.0', '1.6.2')).toBe(false);
    expect(isUpgrade('1.7.0', '1.7.0')).toBe(false);
  });
});
