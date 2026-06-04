import {NextRequest, NextResponse} from 'next/server';

const CONTROL_PLANE_URL = process.env.CONTROL_PLANE_URL || 'http://127.0.0.1:2887';

async function proxy(request: NextRequest, params: {path: string[]}) {
  const targetPath = params.path.join('/');
  const search = request.nextUrl.search || '';
  const url = `${CONTROL_PLANE_URL}/api/audit/${targetPath}${search}`;
  const headers = new Headers();

  const contentType = request.headers.get('content-type');

  if (contentType) {
    headers.set('content-type', contentType);
  }
  request.headers.forEach((value, key) => {
    if (key.toLowerCase().startsWith('x-one-proxy-')) {
      headers.set(key, value);
    }
  });

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

export async function PUT(request: NextRequest, {params}: {params: Promise<{path: string[]}>}) {
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

export async function HEAD(request: NextRequest, {params}: {params: Promise<{path: string[]}>}) {
  const resolved = await params;
  return proxy(request, resolved);
}

export async function OPTIONS(request: NextRequest, {params}: {params: Promise<{path: string[]}>}) {
  const resolved = await params;
  return proxy(request, resolved);
}
