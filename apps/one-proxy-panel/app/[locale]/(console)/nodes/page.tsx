import {redirect} from 'next/navigation';

export default async function NodesPage({params}: {params: Promise<{locale: string}>}) {
  const {locale} = await params;
  redirect(`/${locale}/nodes/connect`);
}
