import {NextResponse} from 'next/server';

const CONTROL_PLANE_URL = process.env.CONTROL_PLANE_URL || 'http://127.0.0.1:2887';

export async function GET() {
  const url = `${CONTROL_PLANE_URL}/healthz`;

  let response: Response;
  try {
    response = await fetch(url, {
      method: 'GET',
      cache: 'no-store',
      signal: AbortSignal.timeout(3000)
    });
  } catch {
    return NextResponse.json({code: 503, message: 'control_plane_unavailable'}, {status: 503});
  }

  const body = await response.text();

  return new NextResponse(body, {
    status: response.status,
    headers: {
      'content-type': response.headers.get('content-type') || 'application/json; charset=utf-8'
    }
  });
}
