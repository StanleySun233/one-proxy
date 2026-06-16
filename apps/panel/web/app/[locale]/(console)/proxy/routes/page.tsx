import {redirect} from 'next/navigation';

export default async function RoutesIndexPage({params}: {params: Promise<{locale: string}>}) {
  const {locale} = await params;
  redirect(`/${locale}/proxy/routes/rules`);
}
