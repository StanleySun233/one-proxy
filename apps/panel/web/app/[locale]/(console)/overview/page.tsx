import {redirect} from 'next/navigation';

export default async function OverviewIndexPage({params}: {params: Promise<{locale: string}>}) {
  const {locale} = await params;
  redirect(`/${locale}/overview/dashboard`);
}
