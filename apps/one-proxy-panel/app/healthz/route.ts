import {NextRequest, NextResponse} from 'next/server';

const CONTROL_PLANE_URL = process.env.CONTROL_PLANE_URL || 'http://127.0.0.1:2887';

export async function GET(request: NextRequest) {
  const url = `${CONTROL_PLANE_URL}/healthz`;

  const response = await fetch(url, {
    method: 'GET',
    cache: 'no-store'
  });

  const body = await response.text();

  return new NextResponse(body, {
    status: response.status,
    headers: {
      'content-type': response.headers.get('content-type') || 'application/json; charset=utf-8'
    }
  });
}
