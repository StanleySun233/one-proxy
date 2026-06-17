import {NextResponse} from 'next/server';

const OWNER = 'StanleySun233';
const REPO = 'one-proxy';
const NODE_IMAGE_REPO = 'ghcr.io/stanleysun233/oneproxy-node';
const BUILD_NODE_IMAGE = process.env.NEXT_PUBLIC_ONEPROXY_NODE_IMAGE || '';
const RELEASE_TAG = /^v(\d+)\.(\d+)\.(\d+)$/;

type GitHubTag = {
  name?: string;
};

function compareReleaseTags(left: string, right: string) {
  const a = RELEASE_TAG.exec(left);
  const b = RELEASE_TAG.exec(right);
  if (!a || !b) {
    return right.localeCompare(left);
  }
  for (let index = 1; index <= 3; index += 1) {
    const diff = Number(b[index]) - Number(a[index]);
    if (diff !== 0) {
      return diff;
    }
  }
  return 0;
}

function buildFallbackTags() {
  const tag = BUILD_NODE_IMAGE.startsWith(`${NODE_IMAGE_REPO}:`) ? BUILD_NODE_IMAGE.slice(NODE_IMAGE_REPO.length + 1) : '';
  return RELEASE_TAG.test(tag) ? [tag] : [];
}

export async function GET() {
  const fallbackTags = buildFallbackTags();
  let tags = fallbackTags;

  try {
    const response = await fetch(`https://api.github.com/repos/${OWNER}/${REPO}/tags?per_page=100`, {
      headers: {
        accept: 'application/vnd.github+json',
        'user-agent': 'one-proxy-panel'
      },
      next: {revalidate: 300}
    });
    if (response.ok) {
      const payload = await response.json() as GitHubTag[];
      tags = Array.from(new Set(payload.map((item) => item.name || '').filter((name) => RELEASE_TAG.test(name)))).sort(compareReleaseTags);
    }
  } catch {
    tags = fallbackTags;
  }

  if (tags.length === 0) {
    tags = fallbackTags;
  }

  if (tags.length === 0) {
    return NextResponse.json({
      code: 1,
      message: 'github_tags_unavailable',
      data: null
    }, {status: 502});
  }

  return NextResponse.json({
    code: 0,
    message: 'ok',
    data: {
      imageRepo: NODE_IMAGE_REPO,
      latestTag: tags[0] || '',
      tags
    }
  });
}
