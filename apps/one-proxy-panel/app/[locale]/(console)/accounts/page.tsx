import {redirect} from 'next/navigation';

export default async function AccountsPage({params}: {params: Promise<{locale: string}>}) {
  const {locale} = await params;
  redirect(`/${locale}/accounts/create`);
}
