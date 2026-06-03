import {redirect} from 'next/navigation';

export default async function ChainAccessPathsRedirectPage({params}: {params: Promise<{locale: string}>}) {
  const {locale} = await params;
  redirect(`/${locale}/chains/network`);
}
