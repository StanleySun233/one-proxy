import assert from 'node:assert/strict';
import { PassThrough } from 'node:stream';
import test from 'node:test';

import { promptPassword } from '../src/prompt.ts';

test('password prompt shows label but does not echo entered password', async () => {
  const input = new PassThrough();
  const output = new PassThrough();
  let written = '';
  output.write = ((chunk, encoding, callback) => {
    written += String(chunk);
    if (typeof encoding === 'function') {
      encoding();
    } else if (callback) {
      callback();
    }
    return true;
  });

  const answer = promptPassword('Password: ', { input, output });
  input.write('secret-password\n');
  input.end();

  assert.equal(await answer, 'secret-password');
  assert.equal(written.includes('Password: '), true);
  assert.equal(written.includes('secret-password'), false);
});
