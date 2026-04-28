import {NextRequest, NextResponse} from 'next/server';

const CONTROL_PLANE_URL = process.env.CONTROL_PLANE_URL || 'http://127.0.0.1:2887';

async function proxy(request: NextRequest, params: {path: string[]}) {
  const targetPath = params.path.join('/');
  const search = request.nextUrl.search || '';
  const url = `${CONTROL_PLANE_URL}/api/v1/${targetPath}${search}`;
  const headers = new Headers();

  const authorization = request.headers.get('authorization');
  const contentType = request.headers.get('content-type');

  if (authorization) {
    headers.set('authorization', authorization);
  }
  if (contentType) {
    headers.set('content-type', contentType);
  }

  const method = request.method;
  const init: RequestInit = {
    method,
    headers,
    cache: 'no-store'
  };

  if (method !== 'GET' && method !== 'HEAD') {
    init.body = await request.text();
  }

  const response = await fetch(url, init);
  const body = await response.text();

  return new NextResponse(body, {
    status: response.status,
    headers: {
      'content-type': response.headers.get('content-type') || 'application/json; charset=utf-8'
    }
  });
}

export async function GET(request: NextRequest, {params}: {params: Promise<{path: string[]}>}) {
  const resolved = await params;
  return proxy(request, resolved);
}

export async function POST(request: NextRequest, {params}: {params: Promise<{path: string[]}>}) {
  const resolved = await params;
  return proxy(request, resolved);
}

export async function PATCH(request: NextRequest, {params}: {params: Promise<{path: string[]}>}) {
  const resolved = await params;
  return proxy(request, resolved);
}

export async function DELETE(request: NextRequest, {params}: {params: Promise<{path: string[]}>}) {
  const resolved = await params;
  return proxy(request, resolved);
}
