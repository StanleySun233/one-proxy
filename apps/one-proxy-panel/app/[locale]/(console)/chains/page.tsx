import {redirect} from 'next/navigation';

export default async function ChainsPage({params}: {params: Promise<{locale: string}>}) {
  const {locale} = await params;
  redirect(`/${locale}/chains/scopes`);
}
