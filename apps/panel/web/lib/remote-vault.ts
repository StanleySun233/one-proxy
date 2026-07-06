import type {RemoteSecret} from '@/lib/types';

const encoder = new TextEncoder();
const decoder = new TextDecoder();
const ITERATIONS = 210000;

type EncryptedPayload = {
  version: 1;
  kdf: 'PBKDF2-SHA256';
  iterations: number;
  salt: string;
  iv: string;
  ciphertext: string;
};

export async function encryptRemoteSecret(secret: RemoteSecret, passphrase: string) {
  const salt = crypto.getRandomValues(new Uint8Array(16));
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const key = await deriveRemoteVaultKey(passphrase, salt, ITERATIONS);
  const ciphertext = await crypto.subtle.encrypt({name: 'AES-GCM', iv: toArrayBuffer(iv)}, key, toArrayBuffer(encoder.encode(JSON.stringify(secret))));
  const payload: EncryptedPayload = {
    version: 1,
    kdf: 'PBKDF2-SHA256',
    iterations: ITERATIONS,
    salt: bytesToBase64(salt),
    iv: bytesToBase64(iv),
    ciphertext: bytesToBase64(new Uint8Array(ciphertext))
  };
  return JSON.stringify(payload);
}

export async function decryptRemoteSecret(encryptedPayload: string, passphrase: string): Promise<RemoteSecret> {
  const payload = JSON.parse(encryptedPayload) as EncryptedPayload;
  if (payload.version !== 1 || payload.kdf !== 'PBKDF2-SHA256') {
    throw new Error('unsupported_remote_secret_payload');
  }
  const salt = base64ToBytes(payload.salt);
  const iv = base64ToBytes(payload.iv);
  const ciphertext = base64ToBytes(payload.ciphertext);
  const key = await deriveRemoteVaultKey(passphrase, salt, payload.iterations);
  const plaintext = await crypto.subtle.decrypt({name: 'AES-GCM', iv: toArrayBuffer(iv)}, key, toArrayBuffer(ciphertext));
  const secret = JSON.parse(decoder.decode(plaintext)) as Partial<RemoteSecret>;
  return {
    password: secret.password || '',
    privateKey: secret.privateKey || '',
    passphrase: secret.passphrase || ''
  };
}

async function deriveRemoteVaultKey(passphrase: string, salt: Uint8Array, iterations: number) {
  const baseKey = await crypto.subtle.importKey('raw', toArrayBuffer(encoder.encode(passphrase)), 'PBKDF2', false, ['deriveKey']);
  return crypto.subtle.deriveKey(
    {name: 'PBKDF2', salt: toArrayBuffer(salt), iterations, hash: 'SHA-256'},
    baseKey,
    {name: 'AES-GCM', length: 256},
    false,
    ['encrypt', 'decrypt']
  );
}

function toArrayBuffer(bytes: Uint8Array) {
  return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer;
}

function bytesToBase64(bytes: Uint8Array) {
  let binary = '';
  bytes.forEach((value) => {
    binary += String.fromCharCode(value);
  });
  return btoa(binary);
}

function base64ToBytes(value: string) {
  const binary = atob(value);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index);
  }
  return bytes;
}
